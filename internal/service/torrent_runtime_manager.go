package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"BTDown_MA/internal/config"
	"BTDown_MA/internal/model"
	"BTDown_MA/internal/repository"

	"github.com/anacrolix/dht/v2"
	"github.com/anacrolix/missinggo/v2/pubsub"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"golang.org/x/time/rate"
)

const (
	metadataResolveTimeout       = 10 * time.Minute
	metadataRetryInterval        = 20 * time.Second
	metricsRefreshInterval       = time.Second
	streamReaderReadahead        = int64(8 << 20)
	streamPreheatPieces          = 8
	bufferHitThreshold           = 200 * time.Millisecond
	metadataSoftWarningWindow    = 2 * time.Minute
	metadataFallbackInterval     = 3 * metadataRetryInterval
	metadataFallbackTimeout      = 8 * time.Second
	metadataFallbackMaxBytes     = 8 << 20
	metadataDNSResolveTimeout    = 3 * time.Second
	windowAdvanceThrottle        = 150 * time.Millisecond
	pieceEventIdlePollInterval   = 250 * time.Millisecond
	defaultPlaybackWindowPieces  = 6
	defaultPrefetchWindowPieces  = 12
	defaultRetentionWindowPieces = 4
)

type TorrentRuntimeManager = torrentRuntimeManager

type torrentRuntimeManager struct {
	sessionRepository  repository.SessionRepository
	client             *torrent.Client
	settings           config.ApplicationSettings
	metadataHTTPClient *http.Client

	mutex    sync.RWMutex
	runtimes map[string]*sessionRuntime
}

type sessionRuntime struct {
	sessionID string
	torrent   *torrent.Torrent
	ctx       context.Context
	cancel    context.CancelFunc

	mutex                    sync.RWMutex
	selectedFile             *torrent.File
	selectedFileName         string
	activeStreamRequests     int
	paused                   bool
	lastUsefulDataBytes      int64
	lastUsefulBytesDelta     int64
	metadataResolveStartedAt time.Time
	lastStatsAt              time.Time
	lastMetricsLogAt         time.Time
	lastRangeRequestAt       time.Time
	lastRangeResponseAt      time.Time
	lastRangeDurationMs      int64
	seekRecoveryMs           int64
	firstFrameLatencyMs      int64
	hasLastRangeStart        bool
	lastRangeStart           int64
	successfulStreamRequests int64
	bufferHitRequests        int64
	originalTrackerCount     int
	appendedTrackerCount     int
	appendedTrackerSet       map[string]struct{}
	torrentPublicity         string
	currentReadaheadBytes    int64
	lastPriorityWindowBegin  int
	lastPriorityWindowEnd    int
	playbackWindowBegin      int
	playbackWindowEnd        int
	prefetchWindowBegin      int
	prefetchWindowEnd        int
	retentionWindowBegin     int
	retentionWindowEnd       int
	windowGeneration         int64
	lastPreparedRangeStart   int64
	hasPreparedRangeStart    bool
	lastPreparedAt           time.Time
	lastSeekAt               time.Time
	seekTargetGeneration     int64
	pendingSeekRecovery      bool
	pieceSubscription        *pubsub.Subscription[torrent.PieceStateChange]
	pieceSubscriptionActive  bool
	lastWindowAdvanceAt      time.Time
	lastPieceEventAt         time.Time
}

type priorityWindowPlan struct {
	generation          int64
	rangeStart          int64
	pieceIndex          int
	playbackBegin       int
	playbackEnd         int
	prefetchBegin       int
	prefetchEnd         int
	retentionBegin      int
	retentionEnd        int
	previousWindowBegin int
	previousWindowEnd   int
	isSeek              bool
}

type streamResource struct {
	name    string
	modTime time.Time
	size    int64
	content io.ReadSeekCloser
}

func NewTorrentRuntimeManager(sessionRepository repository.SessionRepository, settings config.ApplicationSettings) (*torrentRuntimeManager, error) {
	if err := os.MkdirAll(settings.TorrentDataDir, 0o755); err != nil {
		return nil, fmt.Errorf("创建下载目录失败: %w", err)
	}

	clientConfig := torrent.NewDefaultClientConfig()
	clientConfig.DataDir = filepath.Clean(settings.TorrentDataDir)
	clientConfig.ListenPort = settings.BTListenPort
	clientConfig.NoDefaultPortForwarding = !settings.EnablePortForwarding
	clientConfig.NoUpload = false
	clientConfig.UploadRateLimiter = buildRateLimiter(settings.UploadRateLimitKiBps)
	clientConfig.DownloadRateLimiter = buildRateLimiter(settings.DownloadRateLimitKiBps)

	client, err := torrent.NewClient(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("初始化 torrent client 失败: %w", err)
	}
	bootstrapNodes := resolveBootstrapDHTNodes(defaultBootstrapDHTNodes())
	if len(bootstrapNodes) > 0 {
		client.AddDhtNodes(bootstrapNodes)
	}
	log.Printf("[runtime-client-init] listenPort=%d listenAddrs=%v dhtServers=%d bootstrapNodes=%d", settings.BTListenPort, client.ListenAddrs(), len(client.DhtServers()), len(bootstrapNodes))

	return &torrentRuntimeManager{
		sessionRepository:  sessionRepository,
		client:             client,
		settings:           settings,
		metadataHTTPClient: buildMetadataHTTPClient(settings),
		runtimes:           make(map[string]*sessionRuntime),
	}, nil
}

func (manager *torrentRuntimeManager) ApplySettings(settings config.ApplicationSettings) {
	manager.mutex.Lock()
	manager.settings = settings
	manager.metadataHTTPClient = buildMetadataHTTPClient(settings)
	manager.mutex.Unlock()
}

func (manager *torrentRuntimeManager) currentSettings() config.ApplicationSettings {
	manager.mutex.RLock()
	defer manager.mutex.RUnlock()
	return manager.settings
}

func (manager *torrentRuntimeManager) metadataClient() *http.Client {
	manager.mutex.RLock()
	defer manager.mutex.RUnlock()
	if manager.metadataHTTPClient != nil {
		return manager.metadataHTTPClient
	}
	return http.DefaultClient
}

func (manager *torrentRuntimeManager) startSession(session model.Session) {
	go manager.bootstrapSession(session)
}

func (manager *torrentRuntimeManager) recoverSession(session model.Session) {
	if _, exists := manager.getRuntime(session.ID); exists {
		return
	}
	log.Printf("[session-recover] session=%s downloadState=%s metadataState=%s", session.ID, session.DownloadState, session.MetadataState)
	go manager.bootstrapSession(session)
}

func (manager *torrentRuntimeManager) bootstrapSession(session model.Session) {
	torrentHandle, err := manager.client.AddMagnet(session.MagnetURI)
	if err != nil {
		manager.failSession(session.ID, fmt.Sprintf("添加 magnet 失败: %v", err))
		return
	}

	runtimeCtx, runtimeCancel := context.WithCancel(context.Background())
	settings := manager.currentSettings()
	runtime := &sessionRuntime{
		sessionID:                session.ID,
		torrent:                  torrentHandle,
		ctx:                      runtimeCtx,
		cancel:                   runtimeCancel,
		metadataResolveStartedAt: time.Now(),
		lastStatsAt:              time.Now(),
		originalTrackerCount:     trackerCount(torrentHandle),
		appendedTrackerSet:       make(map[string]struct{}),
		torrentPublicity:         "unknown",
		currentReadaheadBytes:    clampInt64(streamReaderReadahead, settings.StreamReadaheadMinBytes, settings.StreamReadaheadMaxBytes),
		lastPriorityWindowBegin:  -1,
		lastPriorityWindowEnd:    -1,
		playbackWindowBegin:      -1,
		playbackWindowEnd:        -1,
		prefetchWindowBegin:      -1,
		prefetchWindowEnd:        -1,
		retentionWindowBegin:     -1,
		retentionWindowEnd:       -1,
	}
	if runtime.originalTrackerCount == 0 {
		runtime.appendedTrackerCount = appendMissingPublicTrackers(torrentHandle, settings.PublicTrackers, runtime.appendedTrackerSet)
	}
	totalTrackerCount := trackerCount(torrentHandle)
	log.Printf("[session-bootstrap] session=%s listenPort=%d trackers(original=%d appended=%d total=%d)", session.ID, settings.BTListenPort, runtime.originalTrackerCount, runtime.appendedTrackerCount, totalTrackerCount)

	manager.mutex.Lock()
	manager.runtimes[session.ID] = runtime
	manager.mutex.Unlock()

	_, updateErr := manager.sessionRepository.UpdateByID(session.ID, func(current *model.Session) error {
		current.MetadataState = model.SessionMetadataStateResolving
		current.DownloadState = model.SessionDownloadStateQueued
		current.StreamState = model.SessionStreamStateUnavailable
		current.DeadState = model.SessionDeadStateUnknown
		current.HealthReport = model.SessionHealthReport{
			Summary: "metadata 中",
			Reason:  "正在等待 tracker 与 metadata",
		}
		current.TrackerCount = totalTrackerCount
		current.OriginalTrackerCount = runtime.originalTrackerCount
		current.AppendedTrackerCount = runtime.appendedTrackerCount
		current.TorrentPublicity = runtime.torrentPublicity
		current.ListenPort = settings.BTListenPort
		current.DHTStatus = "unknown"
		current.DHTNodes = 0
		current.UDPReachable = "unknown"
		current.IncomingConnections = 0
		current.LastError = ""
		refreshSessionDerivedFields(current, 0)
		return nil
	})
	if updateErr != nil {
		log.Printf("update session %s failed: %v", session.ID, updateErr)
	}

	go manager.waitForMetadata(runtime)
	go manager.refreshMetrics(runtime)
}

func (manager *torrentRuntimeManager) waitForMetadata(runtime *sessionRuntime) {
	deadline := time.Now().Add(metadataResolveTimeout)
	ticker := time.NewTicker(metadataRetryInterval)
	defer ticker.Stop()

	retryCount := 0
	for {
		if runtime.torrent.Info() != nil {
			log.Printf("[metadata-ready] session=%s retries=%d", runtime.sessionID, retryCount)
			break
		}

		if time.Now().After(deadline) {
			log.Printf("[metadata-timeout] session=%s timeout=%s retries=%d", runtime.sessionID, metadataResolveTimeout, retryCount)
			_, err := manager.sessionRepository.UpdateByID(runtime.sessionID, func(current *model.Session) error {
				if current.LastError == "" {
					current.MetadataState = model.SessionMetadataStateResolving
					current.DownloadState = model.SessionDownloadStateQueued
					current.StreamState = model.SessionStreamStateUnavailable
					current.HealthReport = model.SessionHealthReport{
						Summary: "metadata 较慢",
						Reason:  "超过等待窗口仍未拿到 metadata，后台继续重试中",
					}
					refreshSessionDerivedFields(current, manager.activeStreamRequests(runtime.sessionID))
				}
				return nil
			})
			if err != nil {
				log.Printf("mark metadata slow for session %s failed: %v", runtime.sessionID, err)
			}
			deadline = time.Now().Add(metadataResolveTimeout)
		}

		select {
		case <-runtime.ctx.Done():
			return
		case <-ticker.C:
			retryCount++
			settings := manager.currentSettings()
			runtime.mutex.Lock()
			appended := appendMissingPublicTrackers(runtime.torrent, settings.PublicTrackers, runtime.appendedTrackerSet)
			runtime.appendedTrackerCount += appended
			trackerTotal := trackerCount(runtime.torrent)
			metadataResolveStartedAt := runtime.metadataResolveStartedAt
			runtime.mutex.Unlock()
			if appended > 0 {
				log.Printf("[metadata-retry] session=%s retry=%d appended=%d trackers=%d", runtime.sessionID, retryCount, appended, trackerTotal)
			}

			if metadataResolveStartedAt.IsZero() {
				metadataResolveStartedAt = time.Now()
			}
			if time.Since(metadataResolveStartedAt) > metadataSoftWarningWindow {
				_, err := manager.sessionRepository.UpdateByID(runtime.sessionID, func(current *model.Session) error {
					if current.LastError == "" {
						current.MetadataState = model.SessionMetadataStateResolving
						current.DownloadState = model.SessionDownloadStateQueued
						current.StreamState = model.SessionStreamStateUnavailable
						current.LastError = ""
						refreshSessionDerivedFields(current, manager.activeStreamRequests(runtime.sessionID))
					}
					return nil
				})
				if err != nil {
					log.Printf("update metadata resolving for session %s failed: %v", runtime.sessionID, err)
				}
			}

			if retryCount%3 == 0 {
				manager.forceDHTAnnounce(runtime, retryCount)
				if manager.tryInjectMetadataFallback(runtime) {
					continue
				}
			}
		}
	}

	descriptor := selectPrimarySessionFile(runtime.torrent.Files())
	if descriptor == nil || descriptor.file == nil {
		manager.failSession(runtime.sessionID, "未找到可播放的文件")
		return
	}

	descriptor.file.SetPriority(torrent.PiecePriorityHigh)
	descriptor.file.Download()
	settings := manager.currentSettings()
	preheatHeadPieces := settings.StreamPreheatHeadPieces
	preheatTailPieces := settings.StreamPreheatTailPieces
	if preheatHeadPieces <= 0 {
		preheatHeadPieces = streamPreheatPieces
	}
	if preheatTailPieces <= 0 {
		preheatTailPieces = streamPreheatPieces
	}
	preheatFilePieces(runtime.torrent, descriptor.file, preheatHeadPieces, preheatTailPieces)
	manager.ensurePieceEventSubscription(runtime)

	runtime.mutex.Lock()
	runtime.torrentPublicity = torrentPublicity(runtime.torrent)
	if runtime.appendedTrackerCount == 0 && (runtime.originalTrackerCount == 0 || (settings.AppendPublicTrackersForPublicTorrent && runtime.torrentPublicity == "public")) {
		runtime.appendedTrackerCount = appendMissingPublicTrackers(runtime.torrent, settings.PublicTrackers, runtime.appendedTrackerSet)
	}
	runtime.selectedFile = descriptor.file
	runtime.selectedFileName = descriptor.name
	originalTrackerCount := runtime.originalTrackerCount
	appendedTrackerCount := runtime.appendedTrackerCount
	torrentPublicity := runtime.torrentPublicity
	runtime.mutex.Unlock()

	_, err := manager.sessionRepository.UpdateByID(runtime.sessionID, func(current *model.Session) error {
		current.MetadataState = model.SessionMetadataStateReady
		current.DownloadState = model.SessionDownloadStateDownloading
		current.StreamState = model.SessionStreamStateReady
		current.SelectedFileName = descriptor.name
		current.TrackerCount = trackerCount(runtime.torrent)
		current.OriginalTrackerCount = originalTrackerCount
		current.AppendedTrackerCount = appendedTrackerCount
		current.TorrentPublicity = torrentPublicity
		current.ListenPort = settings.BTListenPort
		current.TotalBytes = descriptor.length
		current.LastError = ""
		refreshSessionDerivedFields(current, manager.activeStreamRequests(runtime.sessionID))
		return nil
	})
	if err != nil {
		log.Printf("update session %s after metadata failed: %v", runtime.sessionID, err)
	}
}

func (manager *torrentRuntimeManager) refreshMetrics(runtime *sessionRuntime) {
	ticker := time.NewTicker(metricsRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-runtime.ctx.Done():
			return
		case <-ticker.C:
		}
		stats := runtime.torrent.Stats()
		settings := manager.currentSettings()

		runtime.mutex.Lock()
		now := time.Now()
		elapsed := now.Sub(runtime.lastStatsAt).Seconds()
		if elapsed <= 0 {
			elapsed = 1
		}
		paused := runtime.paused
		currentUsefulDataBytes := stats.BytesReadUsefulData.Int64()
		usefulBytesDelta := currentUsefulDataBytes - runtime.lastUsefulDataBytes
		if usefulBytesDelta < 0 {
			usefulBytesDelta = 0
		}
		speed := int64(math.Max(0, float64(usefulBytesDelta)/elapsed))
		runtime.lastUsefulDataBytes = currentUsefulDataBytes
		runtime.lastUsefulBytesDelta = usefulBytesDelta
		runtime.lastStatsAt = now
		selectedFile := runtime.selectedFile
		selectedFileName := runtime.selectedFileName
		activeStreams := runtime.activeStreamRequests
		lastRangeRequestAt := runtime.lastRangeRequestAt
		lastRangeResponseAt := runtime.lastRangeResponseAt
		lastRangeDurationMs := runtime.lastRangeDurationMs
		seekRecoveryMs := runtime.seekRecoveryMs
		firstFrameLatencyMs := runtime.firstFrameLatencyMs
		bufferHitRatio := computeBufferHitRatio(runtime.bufferHitRequests, runtime.successfulStreamRequests)
		originalTrackerCount := runtime.originalTrackerCount
		appendedTrackerCount := runtime.appendedTrackerCount
		torrentPublicity := runtime.torrentPublicity
		runtime.mutex.Unlock()

		dhtNodes := dhtNodeCount(manager.client)
		dhtStatus := classifyDHTStatus(stats.ActivePeers, stats.TotalPeers)
		incomingConnections := incomingConnectionCount(stats)
		udpReachable := classifyUDPReachable(stats.ActivePeers, stats.TotalPeers, incomingConnections)

		downloadedBytes := int64(0)
		totalBytes := int64(0)
		metadataState := model.SessionMetadataStateResolving
		streamState := model.SessionStreamStateUnavailable
		if selectedFile != nil {
			downloadedBytes = selectedFile.BytesCompleted()
			totalBytes = selectedFile.Length()
			metadataState = model.SessionMetadataStateReady
			if activeStreams > 0 {
				streamState = model.SessionStreamStateStreaming
			} else {
				streamState = model.SessionStreamStateReady
			}
		}

		downloadState := model.SessionDownloadStateQueued
		if totalBytes > 0 && downloadedBytes >= totalBytes {
			downloadState = model.SessionDownloadStateCompleted
		} else if paused {
			downloadState = model.SessionDownloadStatePaused
		} else if selectedFile != nil {
			downloadState = model.SessionDownloadStateDownloading
		}

		deadState := model.SessionDeadStateUnknown
		if stats.TotalPeers > 0 || stats.ActivePeers > 0 {
			deadState = model.SessionDeadStateAlive
		} else if selectedFile != nil {
			deadState = model.SessionDeadStateNoPeers
		}

		_, err := manager.sessionRepository.UpdateByID(runtime.sessionID, func(current *model.Session) error {
			if current.LastError != "" {
				current.MetadataState = model.SessionMetadataStateError
				current.DownloadState = model.SessionDownloadStateError
				current.StreamState = model.SessionStreamStateError
			} else {
				current.MetadataState = metadataState
				current.DownloadState = downloadState
				current.StreamState = streamState
			}
			current.DeadState = deadState
			current.SelectedFileName = selectedFileName
			current.TrackerCount = trackerCount(runtime.torrent)
			current.OriginalTrackerCount = originalTrackerCount
			current.AppendedTrackerCount = appendedTrackerCount
			current.TorrentPublicity = torrentPublicity
			current.ListenPort = settings.BTListenPort
			current.DHTStatus = dhtStatus
			current.DHTNodes = dhtNodes
			current.UDPReachable = udpReachable
			current.IncomingConnections = incomingConnections
			current.ActivePeers = stats.ActivePeers
			current.TotalPeers = stats.TotalPeers
			current.DownloadSpeedBytesPerSecond = speed
			current.UsefulBytesDelta = usefulBytesDelta
			current.DownloadedBytes = downloadedBytes
			current.LastRangeRequestAt = lastRangeRequestAt
			current.LastRangeResponseAt = lastRangeResponseAt
			current.LastRangeResponseDurationMs = lastRangeDurationMs
			current.SeekRecoveryMs = seekRecoveryMs
			current.BufferHitRatio = bufferHitRatio
			if firstFrameLatencyMs > 0 {
				current.FirstFrameLatencyMs = firstFrameLatencyMs
			}
			if totalBytes > 0 {
				current.TotalBytes = totalBytes
			}
			refreshSessionDerivedFields(current, activeStreams)
			return nil
		})
		if now.Sub(runtime.lastMetricsLogAt) >= 10*time.Second {
			runtime.lastMetricsLogAt = now
			log.Printf("[session-metrics] session=%s peers=%d/%d speed=%d dht=%d paused=%t", runtime.sessionID, stats.ActivePeers, stats.TotalPeers, speed, dhtNodes, paused)
		}
		if err != nil {
			log.Printf("refresh metrics for session %s failed: %v", runtime.sessionID, err)
		}
	}
}

func (manager *torrentRuntimeManager) openStream(ctx context.Context, sessionID string) (*streamResource, error) {
	runtime, ok := manager.getRuntime(sessionID)
	if !ok {
		return nil, errors.New("session runtime 不存在")
	}

	runtime.mutex.Lock()
	selectedFile := runtime.selectedFile
	selectedFileName := runtime.selectedFileName
	currentReadaheadBytes := runtime.currentReadaheadBytes
	paused := runtime.paused
	if currentReadaheadBytes <= 0 {
		currentReadaheadBytes = streamReaderReadahead
	}
	if paused {
		runtime.mutex.Unlock()
		return nil, errors.New("会话已暂停，请先继续下载")
	}
	if selectedFile == nil {
		runtime.mutex.Unlock()
		return nil, errors.New("主文件尚未准备好")
	}
	runtime.activeStreamRequests++
	activeStreams := runtime.activeStreamRequests
	runtime.mutex.Unlock()

	if _, err := manager.sessionRepository.UpdateByID(sessionID, func(current *model.Session) error {
		current.StreamState = model.SessionStreamStateStreaming
		refreshSessionDerivedFields(current, activeStreams)
		return nil
	}); err != nil {
		log.Printf("mark session %s streaming failed: %v", sessionID, err)
	}

	reader := selectedFile.NewReader()
	reader.SetContext(ctx)
	reader.SetResponsive()
	reader.SetReadahead(currentReadaheadBytes)

	return &streamResource{
		name:    filepath.Base(selectedFileName),
		modTime: time.Now(),
		size:    selectedFile.Length(),
		content: &trackedStreamReader{
			ReadSeekCloser: reader,
			closeFn: func() {
				manager.releaseStream(sessionID)
			},
		},
	}, nil
}

func (manager *torrentRuntimeManager) releaseStream(sessionID string) {
	runtime, ok := manager.getRuntime(sessionID)
	if !ok {
		return
	}

	runtime.mutex.Lock()
	if runtime.activeStreamRequests > 0 {
		runtime.activeStreamRequests--
	}
	activeStreams := runtime.activeStreamRequests
	lastRangeRequestAt := runtime.lastRangeRequestAt
	lastRangeResponseAt := runtime.lastRangeResponseAt
	lastRangeDurationMs := runtime.lastRangeDurationMs
	seekRecoveryMs := runtime.seekRecoveryMs
	firstFrameLatencyMs := runtime.firstFrameLatencyMs
	bufferHitRatio := computeBufferHitRatio(runtime.bufferHitRequests, runtime.successfulStreamRequests)
	runtime.mutex.Unlock()

	_, err := manager.sessionRepository.UpdateByID(sessionID, func(current *model.Session) error {
		if current.LastError != "" {
			current.StreamState = model.SessionStreamStateError
		} else if current.MetadataState == model.SessionMetadataStateReady {
			current.StreamState = model.SessionStreamStateReady
		} else {
			current.StreamState = model.SessionStreamStateUnavailable
		}
		current.LastRangeRequestAt = lastRangeRequestAt
		current.LastRangeResponseAt = lastRangeResponseAt
		current.LastRangeResponseDurationMs = lastRangeDurationMs
		current.SeekRecoveryMs = seekRecoveryMs
		current.BufferHitRatio = bufferHitRatio
		if firstFrameLatencyMs > 0 {
			current.FirstFrameLatencyMs = firstFrameLatencyMs
		}
		refreshSessionDerivedFields(current, activeStreams)
		return nil
	})
	if err != nil {
		log.Printf("release stream for session %s failed: %v", sessionID, err)
	}
}

func (manager *torrentRuntimeManager) failSession(sessionID, message string) {
	_, err := manager.sessionRepository.UpdateByID(sessionID, func(current *model.Session) error {
		current.MetadataState = model.SessionMetadataStateError
		current.DownloadState = model.SessionDownloadStateError
		current.StreamState = model.SessionStreamStateError
		current.LastError = message
		refreshSessionDerivedFields(current, manager.activeStreamRequests(sessionID))
		return nil
	})
	if err != nil {
		log.Printf("mark session %s failed: %v", sessionID, err)
	}
}

func (manager *torrentRuntimeManager) activeStreamRequests(sessionID string) int {
	runtime, ok := manager.getRuntime(sessionID)
	if !ok {
		return 0
	}
	runtime.mutex.RLock()
	defer runtime.mutex.RUnlock()
	return runtime.activeStreamRequests
}

func (manager *torrentRuntimeManager) getRuntime(sessionID string) (*sessionRuntime, bool) {
	manager.mutex.RLock()
	defer manager.mutex.RUnlock()
	runtime, exists := manager.runtimes[sessionID]
	return runtime, exists
}

func (manager *torrentRuntimeManager) cleanupSession(sessionID string) error {
	return manager.stopSession(sessionID)
}

func (manager *torrentRuntimeManager) pauseSession(sessionID string) error {
	runtime, ok := manager.getRuntime(sessionID)
	if !ok {
		return fmt.Errorf("session runtime 不存在")
	}
	runtime.mutex.Lock()
	runtime.paused = true
	runtime.mutex.Unlock()

	_, err := manager.sessionRepository.UpdateByID(sessionID, func(current *model.Session) error {
		current.DownloadState = model.SessionDownloadStatePaused
		current.StreamState = model.SessionStreamStateReady
		refreshSessionDerivedFields(current, manager.activeStreamRequests(sessionID))
		return nil
	})
	if err != nil {
		return err
	}
	log.Printf("[session-pause] session=%s", sessionID)
	return nil
}

func (manager *torrentRuntimeManager) resumeSession(sessionID string) error {
	runtime, ok := manager.getRuntime(sessionID)
	if !ok {
		session, exists := manager.sessionRepository.FindByID(sessionID)
		if !exists {
			return fmt.Errorf("session %s 不存在", sessionID)
		}
		manager.recoverSession(session)
		log.Printf("[session-resume] session=%s action=recover-runtime", sessionID)
		return nil
	}

	runtime.mutex.Lock()
	runtime.paused = false
	selectedFile := runtime.selectedFile
	runtime.mutex.Unlock()
	if selectedFile != nil {
		selectedFile.Download()
	}

	_, err := manager.sessionRepository.UpdateByID(sessionID, func(current *model.Session) error {
		if current.MetadataState == model.SessionMetadataStateReady {
			current.DownloadState = model.SessionDownloadStateDownloading
		}
		refreshSessionDerivedFields(current, manager.activeStreamRequests(sessionID))
		return nil
	})
	if err != nil {
		return err
	}
	log.Printf("[session-resume] session=%s", sessionID)
	return nil
}

func (manager *torrentRuntimeManager) stopSession(sessionID string) error {
	runtime, ok := manager.getRuntime(sessionID)
	if !ok {
		return nil
	}

	runtime.cancel()
	if runtime.torrent != nil {
		runtime.torrent.Drop()
	}

	manager.mutex.Lock()
	delete(manager.runtimes, sessionID)
	manager.mutex.Unlock()

	_, err := manager.sessionRepository.UpdateByID(sessionID, func(current *model.Session) error {
		current.StreamState = model.SessionStreamStateUnavailable
		current.DownloadSpeedBytesPerSecond = 0
		current.UsefulBytesDelta = 0
		refreshSessionDerivedFields(current, 0)
		return nil
	})
	if err != nil {
		log.Printf("stop session %s update failed: %v", sessionID, err)
	}
	return nil
}

func (manager *torrentRuntimeManager) RecordRangeActivity(sessionID, rangeHeader string, statusCode int, bytesWritten int64, duration time.Duration) {
	manager.recordRangeActivity(sessionID, rangeHeader, statusCode, bytesWritten, duration)
}

func (manager *torrentRuntimeManager) PrepareRangeDownload(sessionID, rangeHeader string) {
	runtime, ok := manager.getRuntime(sessionID)
	if !ok {
		return
	}
	if !hasRangeStart(rangeHeader) {
		return
	}
	rangeStart := parseRangeStart(rangeHeader)
	plan, ok := manager.buildPriorityWindowPlan(runtime, rangeStart)
	if !ok {
		return
	}
	manager.ensurePieceEventSubscription(runtime)
	manager.applyPriorityWindowPlan(runtime, plan)
}

func (manager *torrentRuntimeManager) recordRangeActivity(sessionID, rangeHeader string, statusCode int, bytesWritten int64, duration time.Duration) {
	runtime, ok := manager.getRuntime(sessionID)
	if !ok {
		return
	}
	settings := manager.currentSettings()

	now := time.Now()
	runtime.mutex.Lock()
	runtime.lastRangeRequestAt = now
	runtime.lastRangeResponseAt = now.Add(duration)
	runtime.lastRangeDurationMs = duration.Milliseconds()
	shouldApplyPriority := false
	rangeStart := int64(0)
	isSeek := false
	if statusCode >= 200 && statusCode < 300 && bytesWritten > 0 {
		runtime.successfulStreamRequests++
		if duration <= bufferHitThreshold {
			runtime.bufferHitRequests++
		}
		if runtime.firstFrameLatencyMs == 0 {
			runtime.firstFrameLatencyMs = duration.Milliseconds()
		}
		rangeStart = parseRangeStart(rangeHeader)
		if runtime.hasLastRangeStart {
			gap := absInt64(rangeStart - runtime.lastRangeStart)
			seekThreshold := int64(float64(runtime.currentReadaheadBytes) * settings.StreamSeekGapFactor)
			if seekThreshold <= 0 {
				seekThreshold = streamReaderReadahead
			}
			if gap > seekThreshold {
				runtime.seekRecoveryMs = duration.Milliseconds()
				isSeek = true
			}
		}
		runtime.lastRangeStart = rangeStart
		runtime.hasLastRangeStart = true
		if settings.StreamDynamicReadaheadEnabled {
			runtime.currentReadaheadBytes = nextReadahead(runtime.currentReadaheadBytes, duration, isSeek, settings.StreamReadaheadMinBytes, settings.StreamReadaheadMaxBytes)
		}
		shouldApplyPriority = true
	}
	activeStreams := runtime.activeStreamRequests
	lastRangeRequestAt := runtime.lastRangeRequestAt
	lastRangeResponseAt := runtime.lastRangeResponseAt
	lastRangeDurationMs := runtime.lastRangeDurationMs
	seekRecoveryMs := runtime.seekRecoveryMs
	firstFrameLatencyMs := runtime.firstFrameLatencyMs
	bufferHitRatio := computeBufferHitRatio(runtime.bufferHitRequests, runtime.successfulStreamRequests)
	runtime.mutex.Unlock()

	if shouldApplyPriority {
		plan, ok := manager.buildPriorityWindowPlan(runtime, rangeStart)
		if ok {
			manager.applyPriorityWindowPlan(runtime, plan)
		}
	}

	_, err := manager.sessionRepository.UpdateByID(sessionID, func(current *model.Session) error {
		current.LastRangeRequestAt = lastRangeRequestAt
		current.LastRangeResponseAt = lastRangeResponseAt
		current.LastRangeResponseDurationMs = lastRangeDurationMs
		current.SeekRecoveryMs = seekRecoveryMs
		current.BufferHitRatio = bufferHitRatio
		if firstFrameLatencyMs > 0 {
			current.FirstFrameLatencyMs = firstFrameLatencyMs
		}
		refreshSessionDerivedFields(current, activeStreams)
		return nil
	})
	if err != nil {
		log.Printf("record range activity for session %s failed: %v", sessionID, err)
	}
}

func trackerCount(torrentHandle *torrent.Torrent) int {
	if torrentHandle == nil {
		return 0
	}
	info := torrentHandle.Metainfo()
	announceList := info.AnnounceList
	if len(announceList) > 0 {
		count := 0
		for _, tier := range announceList {
			count += len(tier)
		}
		if count > 0 {
			return count
		}
	}
	if info.Announce != "" {
		return 1
	}
	return 0
}

func buildRateLimiter(limitKiBps int) *rate.Limiter {
	if limitKiBps <= 0 {
		return nil
	}
	bytesPerSecond := limitKiBps * 1024
	return rate.NewLimiter(rate.Limit(bytesPerSecond), bytesPerSecond)
}

func appendPublicTrackers(torrentHandle *torrent.Torrent, trackers []string) int {
	if torrentHandle == nil {
		return 0
	}
	normalized := normalizeTrackerURLs(trackers)
	if len(normalized) == 0 {
		return 0
	}
	announceGroups := make([][]string, 0, len(normalized))
	for _, tracker := range normalized {
		announceGroups = append(announceGroups, []string{tracker})
	}
	torrentHandle.AddTrackers(announceGroups)
	return len(normalized)
}

func appendMissingPublicTrackers(torrentHandle *torrent.Torrent, trackers []string, seen map[string]struct{}) int {
	if torrentHandle == nil {
		return 0
	}
	normalized := normalizeTrackerURLs(trackers)
	if len(normalized) == 0 {
		return 0
	}
	if seen == nil {
		seen = make(map[string]struct{})
	}
	announceGroups := make([][]string, 0, len(normalized))
	for _, tracker := range normalized {
		if _, exists := seen[tracker]; exists {
			continue
		}
		seen[tracker] = struct{}{}
		announceGroups = append(announceGroups, []string{tracker})
	}
	if len(announceGroups) == 0 {
		return 0
	}
	torrentHandle.AddTrackers(announceGroups)
	return len(announceGroups)
}

func normalizeTrackerURLs(trackers []string) []string {
	seen := make(map[string]struct{})
	normalized := make([]string, 0, len(trackers))
	for _, tracker := range trackers {
		candidate := strings.TrimSpace(tracker)
		if candidate == "" {
			continue
		}
		parsed, err := url.Parse(candidate)
		if err != nil {
			continue
		}
		scheme := strings.ToLower(parsed.Scheme)
		if scheme != "udp" && scheme != "http" && scheme != "https" {
			continue
		}
		if parsed.Host == "" {
			continue
		}
		if _, exists := seen[candidate]; exists {
			continue
		}
		seen[candidate] = struct{}{}
		normalized = append(normalized, candidate)
	}
	return normalized
}

func torrentPublicity(torrentHandle *torrent.Torrent) string {
	if torrentHandle == nil {
		return "unknown"
	}
	info := torrentHandle.Info()
	if info == nil {
		return "unknown"
	}
	if info.Private != nil && *info.Private {
		return "private"
	}
	return "public"
}

func dhtNodeCount(client *torrent.Client) int {
	if client == nil {
		return 0
	}
	servers := client.DhtServers()
	if len(servers) == 0 {
		return 0
	}
	count := 0
	for _, server := range servers {
		if server == nil {
			continue
		}
		stats := server.Stats()
		if nodes, ok := extractDHTNodes(stats); ok && nodes > 0 {
			count += nodes
		}
	}
	return count
}

func extractDHTNodes(stats any) (int, bool) {
	switch typed := stats.(type) {
	case dht.ServerStats:
		return typed.Nodes, true
	case *dht.ServerStats:
		if typed == nil {
			return 0, false
		}
		return typed.Nodes, true
	}

	value := reflect.ValueOf(stats)
	if !value.IsValid() {
		return 0, false
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return 0, false
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return 0, false
	}
	field := value.FieldByName("Nodes")
	if !field.IsValid() {
		return 0, false
	}
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int(field.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return int(field.Uint()), true
	}
	return 0, false
}

func classifyDHTStatus(activePeers, totalPeers int) string {
	if activePeers > 0 || totalPeers > 0 {
		return "working"
	}
	return "unknown"
}

func classifyUDPReachable(activePeers, totalPeers, incomingConnections int) string {
	if incomingConnections > 0 {
		return "working"
	}
	if activePeers > 0 || totalPeers > 0 {
		return "unknown"
	}
	return "blocked"
}

func incomingConnectionCount(stats torrent.TorrentStats) int {
	if stats.ActivePeers <= 0 {
		return 0
	}
	if stats.HalfOpenPeers < 0 {
		return 0
	}
	incoming := stats.ActivePeers - stats.HalfOpenPeers
	if incoming < 0 {
		return 0
	}
	return incoming
}

func preheatFilePieces(torrentHandle *torrent.Torrent, file *torrent.File, headPieces int, tailPieces int) {
	begin := file.BeginPieceIndex()
	end := file.EndPieceIndex()
	if begin >= end {
		return
	}
	if headPieces <= 0 {
		headPieces = streamPreheatPieces
	}
	if tailPieces <= 0 {
		tailPieces = streamPreheatPieces
	}

	headEnd := minInt(begin+headPieces, end)
	torrentHandle.DownloadPieces(begin, headEnd)

	tailStart := maxInt(begin, end-tailPieces)
	torrentHandle.DownloadPieces(tailStart, end)
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func computeBufferHitRatio(hit, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(hit) / float64(total)
}

func parseRangeStart(rangeHeader string) int64 {
	const prefix = "bytes="
	trimmed := strings.TrimSpace(rangeHeader)
	if !strings.HasPrefix(trimmed, prefix) {
		return 0
	}
	raw := strings.TrimPrefix(trimmed, prefix)
	if raw == "" {
		return 0
	}
	firstPart := strings.SplitN(raw, ",", 2)[0]
	bounds := strings.SplitN(firstPart, "-", 2)
	if len(bounds) == 0 || strings.TrimSpace(bounds[0]) == "" {
		return 0
	}
	v, err := strconv.ParseInt(strings.TrimSpace(bounds[0]), 10, 64)
	if err != nil || v < 0 {
		return 0
	}
	return v
}

func hasRangeStart(rangeHeader string) bool {
	const prefix = "bytes="
	trimmed := strings.TrimSpace(rangeHeader)
	if !strings.HasPrefix(trimmed, prefix) {
		return false
	}
	raw := strings.TrimPrefix(trimmed, prefix)
	if raw == "" {
		return false
	}
	firstPart := strings.SplitN(raw, ",", 2)[0]
	bounds := strings.SplitN(firstPart, "-", 2)
	return len(bounds) > 0 && strings.TrimSpace(bounds[0]) != ""
}

func absInt64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func nextReadahead(current int64, duration time.Duration, isSeek bool, minValue int64, maxValue int64) int64 {
	if minValue <= 0 {
		minValue = 2 << 20
	}
	if maxValue < minValue {
		maxValue = minValue
	}
	if current <= 0 {
		current = streamReaderReadahead
	}
	step := int64(1 << 20)
	next := current
	if isSeek || duration > 800*time.Millisecond {
		next = current + step
	} else if duration <= bufferHitThreshold {
		next = current - step
	}
	return clampInt64(next, minValue, maxValue)
}

func clampInt64(value, minValue, maxValue int64) int64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func (manager *torrentRuntimeManager) buildPriorityWindowPlan(runtime *sessionRuntime, rangeStart int64) (priorityWindowPlan, bool) {
	runtime.mutex.RLock()
	file := runtime.selectedFile
	torrentHandle := runtime.torrent
	currentReadaheadBytes := runtime.currentReadaheadBytes
	lastRangeStart := runtime.lastRangeStart
	hasLastRangeStart := runtime.hasLastRangeStart
	lastPreparedRangeStart := runtime.lastPreparedRangeStart
	hasPreparedRangeStart := runtime.hasPreparedRangeStart
	previousWindowBegin := runtime.lastPriorityWindowBegin
	previousWindowEnd := runtime.lastPriorityWindowEnd
	generation := runtime.windowGeneration
	runtime.mutex.RUnlock()
	if file == nil || torrentHandle == nil || torrentHandle.Info() == nil {
		return priorityWindowPlan{}, false
	}
	if rangeStart < 0 {
		rangeStart = 0
	}
	pieceLength := torrentHandle.Info().PieceLength
	if pieceLength <= 0 {
		return priorityWindowPlan{}, false
	}

	fileBegin := file.Offset()
	fileEnd := fileBegin + file.Length()
	if fileEnd <= fileBegin {
		return priorityWindowPlan{}, false
	}
	if rangeStart < fileBegin {
		rangeStart = fileBegin
	}
	if rangeStart >= fileEnd {
		rangeStart = fileEnd - 1
	}

	seekThreshold := int64(float64(currentReadaheadBytes) * manager.currentSettings().StreamSeekGapFactor)
	if seekThreshold <= 0 {
		seekThreshold = streamReaderReadahead
	}
	isSeek := false
	if hasLastRangeStart && absInt64(rangeStart-lastRangeStart) > seekThreshold {
		isSeek = true
	}
	if !isSeek && hasPreparedRangeStart && absInt64(rangeStart-lastPreparedRangeStart) > seekThreshold {
		isSeek = true
	}
	if isSeek {
		generation++
	} else if generation == 0 {
		generation = 1
	}

	pieceIndex := file.BeginPieceIndex() + int((rangeStart-fileBegin)/pieceLength)
	playbackSize, prefetchSize, retentionSize := manager.windowPieceSizes()
	playbackBegin := clampPieceIndex(pieceIndex-playbackSize/2, file.BeginPieceIndex(), file.EndPieceIndex())
	playbackEnd := clampPieceIndex(playbackBegin+playbackSize, file.BeginPieceIndex(), file.EndPieceIndex())
	if playbackBegin >= playbackEnd {
		return priorityWindowPlan{}, false
	}
	prefetchBegin := playbackEnd
	prefetchEnd := clampPieceIndex(prefetchBegin+prefetchSize, file.BeginPieceIndex(), file.EndPieceIndex())
	retentionBegin := clampPieceIndex(playbackBegin-retentionSize, file.BeginPieceIndex(), file.EndPieceIndex())
	retentionEnd := playbackBegin
	if retentionEnd < retentionBegin {
		retentionEnd = retentionBegin
	}

	return priorityWindowPlan{
		generation:          generation,
		rangeStart:          rangeStart,
		pieceIndex:          pieceIndex,
		playbackBegin:       playbackBegin,
		playbackEnd:         playbackEnd,
		prefetchBegin:       prefetchBegin,
		prefetchEnd:         prefetchEnd,
		retentionBegin:      retentionBegin,
		retentionEnd:        retentionEnd,
		previousWindowBegin: previousWindowBegin,
		previousWindowEnd:   previousWindowEnd,
		isSeek:              isSeek,
	}, true
}

func (manager *torrentRuntimeManager) applyPriorityWindowPlan(runtime *sessionRuntime, plan priorityWindowPlan) {
	runtime.mutex.RLock()
	file := runtime.selectedFile
	torrentHandle := runtime.torrent
	runtime.mutex.RUnlock()
	if file == nil || torrentHandle == nil {
		return
	}
	settings := manager.currentSettings()

	if plan.isSeek && settings.StreamDeprioritizeOldWindow && plan.previousWindowBegin >= 0 && plan.previousWindowEnd > plan.previousWindowBegin {
		for piece := plan.previousWindowBegin; piece < plan.previousWindowEnd; piece++ {
			torrentHandle.Piece(piece).SetPriority(torrent.PiecePriorityNormal)
		}
	}
	for piece := plan.retentionBegin; piece < plan.retentionEnd; piece++ {
		torrentHandle.Piece(piece).SetPriority(torrent.PiecePriorityNormal)
	}
	for piece := plan.prefetchBegin; piece < plan.prefetchEnd; piece++ {
		torrentHandle.Piece(piece).SetPriority(torrent.PiecePriorityReadahead)
	}
	for piece := plan.playbackBegin; piece < plan.playbackEnd; piece++ {
		torrentHandle.Piece(piece).SetPriority(torrent.PiecePriorityHigh)
	}

	torrentHandle.DownloadPieces(plan.playbackBegin, plan.playbackEnd)
	if plan.prefetchEnd > plan.prefetchBegin {
		torrentHandle.DownloadPieces(plan.prefetchBegin, plan.prefetchEnd)
	}

	runtime.mutex.Lock()
	runtime.lastPriorityWindowBegin = plan.playbackBegin
	runtime.lastPriorityWindowEnd = maxInt(plan.playbackEnd, plan.prefetchEnd)
	runtime.playbackWindowBegin = plan.playbackBegin
	runtime.playbackWindowEnd = plan.playbackEnd
	runtime.prefetchWindowBegin = plan.prefetchBegin
	runtime.prefetchWindowEnd = plan.prefetchEnd
	runtime.retentionWindowBegin = plan.retentionBegin
	runtime.retentionWindowEnd = plan.retentionEnd
	runtime.windowGeneration = plan.generation
	runtime.lastPreparedRangeStart = plan.rangeStart
	runtime.hasPreparedRangeStart = true
	runtime.lastPreparedAt = time.Now()
	if plan.isSeek {
		runtime.lastSeekAt = runtime.lastPreparedAt
		runtime.seekTargetGeneration = plan.generation
		runtime.pendingSeekRecovery = true
	}
	runtime.mutex.Unlock()

	log.Printf("[stream-window-apply] session=%s generation=%d seek=%t rangeStart=%d playback=%d-%d prefetch=%d-%d retention=%d-%d", runtime.sessionID, plan.generation, plan.isSeek, plan.rangeStart, plan.playbackBegin, plan.playbackEnd, plan.prefetchBegin, plan.prefetchEnd, plan.retentionBegin, plan.retentionEnd)
}

func (manager *torrentRuntimeManager) applyRangePriority(runtime *sessionRuntime, rangeStart int64, isSeek bool) {
	plan, ok := manager.buildPriorityWindowPlan(runtime, rangeStart)
	if !ok {
		return
	}
	plan.isSeek = isSeek || plan.isSeek
	manager.applyPriorityWindowPlan(runtime, plan)
}

func (manager *torrentRuntimeManager) ensurePieceEventSubscription(runtime *sessionRuntime) {
	if runtime == nil || runtime.torrent == nil {
		return
	}

	runtime.mutex.Lock()
	if runtime.pieceSubscriptionActive {
		runtime.mutex.Unlock()
		return
	}
	sub := runtime.torrent.SubscribePieceStateChanges()
	runtime.pieceSubscription = sub
	runtime.pieceSubscriptionActive = true
	runtime.mutex.Unlock()

	go manager.consumePieceStateChanges(runtime, sub)
}

func (manager *torrentRuntimeManager) consumePieceStateChanges(runtime *sessionRuntime, sub *pubsub.Subscription[torrent.PieceStateChange]) {
	defer func() {
		sub.Close()
		runtime.mutex.Lock()
		if runtime.pieceSubscription == sub {
			runtime.pieceSubscription = nil
			runtime.pieceSubscriptionActive = false
		}
		runtime.mutex.Unlock()
	}()

	for {
		select {
		case event, ok := <-sub.Values:
			if !ok {
				return
			}
			manager.handlePieceStateChange(runtime, event)
		case <-runtime.ctx.Done():
			return
		}
	}
}

func (manager *torrentRuntimeManager) handlePieceStateChange(runtime *sessionRuntime, event torrent.PieceStateChange) {
	runtime.mutex.Lock()
	selectedFile := runtime.selectedFile
	generation := runtime.windowGeneration
	playbackBegin := runtime.playbackWindowBegin
	playbackEnd := runtime.playbackWindowEnd
	prefetchBegin := runtime.prefetchWindowBegin
	prefetchEnd := runtime.prefetchWindowEnd
	runtime.lastPieceEventAt = time.Now()
	runtime.mutex.Unlock()
	if selectedFile == nil {
		return
	}
	fileBegin := selectedFile.BeginPieceIndex()
	fileEnd := selectedFile.EndPieceIndex()
	if event.Index < fileBegin || event.Index >= fileEnd {
		return
	}
	if !event.Complete || !event.Ok {
		return
	}
	if playbackBegin >= 0 && event.Index >= playbackBegin && event.Index < playbackEnd {
		manager.maybeAdvanceWindowFromEvents(runtime, generation)
		return
	}
	if prefetchBegin >= 0 && event.Index >= prefetchBegin && event.Index < prefetchEnd {
		manager.markSeekRecoveryIfSatisfied(runtime, generation)
	}
}

func (manager *torrentRuntimeManager) maybeAdvanceWindowFromEvents(runtime *sessionRuntime, generation int64) {
	runtime.mutex.RLock()
	if runtime.windowGeneration != generation {
		runtime.mutex.RUnlock()
		return
	}
	if time.Since(runtime.lastWindowAdvanceAt) < windowAdvanceThrottle {
		runtime.mutex.RUnlock()
		return
	}
	rangeStart := runtime.lastPreparedRangeStart + runtime.currentReadaheadBytes/2
	runtime.mutex.RUnlock()
	plan, ok := manager.buildPriorityWindowPlan(runtime, rangeStart)
	if !ok {
		return
	}
	plan.isSeek = false
	manager.applyPriorityWindowPlan(runtime, plan)
	runtime.mutex.Lock()
	runtime.lastWindowAdvanceAt = time.Now()
	runtime.mutex.Unlock()
}

func (manager *torrentRuntimeManager) markSeekRecoveryIfSatisfied(runtime *sessionRuntime, generation int64) {
	runtime.mutex.Lock()
	defer runtime.mutex.Unlock()
	if !runtime.pendingSeekRecovery {
		return
	}
	if runtime.seekTargetGeneration != generation || runtime.windowGeneration != generation {
		return
	}
	if time.Since(runtime.lastSeekAt) < pieceEventIdlePollInterval {
		return
	}
	runtime.pendingSeekRecovery = false
}

func (manager *torrentRuntimeManager) windowPieceSizes() (int, int, int) {
	boostWindow := manager.currentSettings().StreamBoostWindowPieces
	if boostWindow <= 0 {
		boostWindow = 12
	}
	playbackSize := maxInt(defaultPlaybackWindowPieces, boostWindow/2)
	prefetchSize := maxInt(defaultPrefetchWindowPieces, boostWindow)
	retentionSize := maxInt(defaultRetentionWindowPieces, playbackSize/2)
	return playbackSize, prefetchSize, retentionSize
}

func clampPieceIndex(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func (manager *torrentRuntimeManager) forceDHTAnnounce(runtime *sessionRuntime, retryCount int) {
	if runtime == nil || runtime.torrent == nil || manager.client == nil {
		return
	}
	servers := manager.client.DhtServers()
	if len(servers) == 0 {
		log.Printf("[dht-announce-skip] session=%s retry=%d reason=no-dht-servers", runtime.sessionID, retryCount)
		return
	}
	success := 0
	for _, server := range servers {
		if server == nil {
			continue
		}
		done, stop, err := runtime.torrent.AnnounceToDht(server)
		if err != nil {
			log.Printf("[dht-announce-failed] session=%s retry=%d err=%v", runtime.sessionID, retryCount, err)
			continue
		}
		success++
		go func(done <-chan struct{}, stop func()) {
			select {
			case <-done:
			case <-time.After(8 * time.Second):
				if stop != nil {
					stop()
				}
			}
		}(done, stop)
	}
	log.Printf("[dht-announce] session=%s retry=%d servers=%d success=%d", runtime.sessionID, retryCount, len(servers), success)
}

func (manager *torrentRuntimeManager) tryInjectMetadataFallback(runtime *sessionRuntime) bool {
	if runtime == nil || runtime.torrent == nil {
		return false
	}
	magnetURI, ok := manager.sessionMagnetURI(runtime.sessionID)
	if !ok || strings.TrimSpace(magnetURI) == "" {
		return false
	}
	infoHash := parseInfoHashFromMagnet(magnetURI)
	if infoHash == "" {
		return false
	}

	settings := manager.currentSettings()
	urls := metadataFallbackSourceURLs(infoHash, settings)
	client := manager.metadataClient()
	for index, sourceURL := range urls {
		metaInfo, err := fetchMetaInfoFromURL(runtime.ctx, client, sourceURL)
		if err != nil {
			log.Printf("[metadata-fallback-fetch-failed] session=%s attempt=%d/%d source=%s err=%v", runtime.sessionID, index+1, len(urls), sourceURL, err)
			continue
		}
		spec, err := torrent.TorrentSpecFromMetaInfoErr(metaInfo)
		if err != nil {
			log.Printf("[metadata-fallback-spec-failed] session=%s attempt=%d/%d source=%s err=%v", runtime.sessionID, index+1, len(urls), sourceURL, err)
			continue
		}
		if err := runtime.torrent.MergeSpec(spec); err != nil {
			log.Printf("[metadata-fallback-merge-failed] session=%s attempt=%d/%d source=%s err=%v", runtime.sessionID, index+1, len(urls), sourceURL, err)
			continue
		}

		runtime.mutex.Lock()
		added := 0
		if len(spec.Trackers) > 0 {
			flattened := make([]string, 0)
			for _, tier := range spec.Trackers {
				flattened = append(flattened, tier...)
			}
			added = appendMissingPublicTrackers(runtime.torrent, flattened, runtime.appendedTrackerSet)
			runtime.appendedTrackerCount += added
		}
		total := trackerCount(runtime.torrent)
		runtime.mutex.Unlock()

		log.Printf("[metadata-fallback-merged] session=%s attempt=%d/%d source=%s trackersAdded=%d trackersTotal=%d", runtime.sessionID, index+1, len(urls), sourceURL, added, total)
		_, _ = manager.sessionRepository.UpdateByID(runtime.sessionID, func(current *model.Session) error {
			if current.LastError == "" {
				current.HealthReport = model.SessionHealthReport{
					Summary: "metadata 后备源已注入",
					Reason:  "主链路较慢，已自动注入外部 metainfo 后继续等待 metadata",
				}
				refreshSessionDerivedFields(current, manager.activeStreamRequests(runtime.sessionID))
			}
			return nil
		})
		return true
	}
	return false
}

func metadataFallbackSourceURLs(infoHash string, settings config.ApplicationSettings) []string {
	upper := strings.ToUpper(strings.TrimSpace(infoHash))
	if upper == "" {
		return nil
	}

	ordered := make([]string, 0, 12)
	seen := make(map[string]struct{})
	push := func(raw string) {
		u := strings.TrimSpace(raw)
		if u == "" {
			return
		}
		if _, exists := seen[u]; exists {
			return
		}
		seen[u] = struct{}{}
		ordered = append(ordered, u)
	}

	if settings.MetadataProxyEnabled && strings.TrimSpace(settings.MetadataProxyURL) != "" {
		base := strings.TrimRight(strings.TrimSpace(settings.MetadataProxyURL), "/")
		push(fmt.Sprintf("%s/torrent/%s.torrent", base, upper))
		push(fmt.Sprintf("%s/torrent/%s.torrent", base, strings.ToLower(upper)))
		push(fmt.Sprintf("%s/%s.torrent", base, upper))
		push(fmt.Sprintf("%s/%s.torrent", base, strings.ToLower(upper)))
	}

	push(fmt.Sprintf("https://itorrents.org/torrent/%s.torrent", upper))
	push(fmt.Sprintf("https://itorrents.org/torrent/%s.torrent", strings.ToLower(upper)))
	push(fmt.Sprintf("https://itorrents.xyz/torrent/%s.torrent", upper))
	push(fmt.Sprintf("https://itorrents.xyz/torrent/%s.torrent", strings.ToLower(upper)))
	push(fmt.Sprintf("https://torrage.info/torrent.php?h=%s", upper))
	push(fmt.Sprintf("https://btcache.me/torrent/%s", upper))
	push(fmt.Sprintf("https://btcache.me/torrent/%s", strings.ToLower(upper)))

	return ordered
}

func (manager *torrentRuntimeManager) sessionMagnetURI(sessionID string) (string, bool) {
	session, exists := manager.sessionRepository.FindByID(sessionID)
	if !exists {
		return "", false
	}
	return session.MagnetURI, true
}

func parseInfoHashFromMagnet(magnetURI string) string {
	parsed, err := url.Parse(strings.TrimSpace(magnetURI))
	if err != nil {
		return ""
	}
	raw := parsed.Query().Get("xt")
	if raw == "" {
		return ""
	}
	const prefix = "urn:btih:"
	if !strings.HasPrefix(strings.ToLower(raw), prefix) {
		return ""
	}
	value := raw[len(prefix):]
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return strings.ToUpper(value)
}

func fetchMetaInfoFromURL(ctx context.Context, client *http.Client, sourceURL string) (*metainfo.MetaInfo, error) {
	requestCtx, cancel := context.WithTimeout(ctx, metadataFallbackTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "BTDown_MA/1.0")
	request.Header.Set("Accept", "application/x-bittorrent, application/octet-stream;q=0.9, */*;q=0.1")
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status %d", response.StatusCode)
	}
	contentType := strings.ToLower(strings.TrimSpace(response.Header.Get("Content-Type")))
	body, err := io.ReadAll(io.LimitReader(response.Body, metadataFallbackMaxBytes))
	if err != nil {
		return nil, err
	}
	if len(body) == 0 {
		return nil, errors.New("empty torrent payload")
	}
	if isLikelyHTMLPayload(contentType, body) {
		return nil, fmt.Errorf("unexpected html payload contentType=%q", contentType)
	}
	if !isLikelyTorrentPayload(body) {
		return nil, fmt.Errorf("unexpected payload prefix=%q", payloadPrefix(body, 24))
	}
	return metainfo.Load(bytes.NewReader(body))
}

func buildMetadataHTTPClient(settings config.ApplicationSettings) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	dialer := &net.Dialer{Timeout: metadataFallbackTimeout}
	dnsServers := []string{"1.1.1.1:53", "8.8.8.8:53", "9.9.9.9:53"}
	dialer.Resolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			resolverDialer := &net.Dialer{Timeout: metadataDNSResolveTimeout}
			for _, server := range dnsServers {
				conn, err := resolverDialer.DialContext(ctx, "udp", server)
				if err == nil {
					return conn, nil
				}
			}
			return nil, fmt.Errorf("all metadata dns resolvers unavailable")
		},
	}
	transport.DialContext = dialer.DialContext
	transport.Proxy = nil
	if settings.MetadataProxyEnabled && strings.TrimSpace(settings.MetadataProxyURL) != "" {
		proxyURL, err := url.Parse(strings.TrimSpace(settings.MetadataProxyURL))
		if err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}
	return &http.Client{
		Transport: transport,
		Timeout:   metadataFallbackTimeout + 2*time.Second,
	}
}

func isLikelyHTMLPayload(contentType string, body []byte) bool {
	if strings.Contains(contentType, "text/html") || strings.Contains(contentType, "application/xhtml") {
		return true
	}
	prefix := strings.ToLower(strings.TrimSpace(payloadPrefix(body, 256)))
	if strings.HasPrefix(prefix, "<") {
		return true
	}
	if strings.Contains(prefix, "<html") || strings.Contains(prefix, "<!doctype html") {
		return true
	}
	return false
}

func isLikelyTorrentPayload(body []byte) bool {
	if len(body) < 4 {
		return false
	}
	prefix := payloadPrefix(body, 64)
	if strings.Contains(prefix, "4:info") {
		return true
	}
	if len(body) > 0 && body[0] == 'd' {
		return true
	}
	return false
}

func payloadPrefix(body []byte, n int) string {
	if n <= 0 {
		return ""
	}
	if len(body) < n {
		n = len(body)
	}
	return string(body[:n])
}

func defaultBootstrapDHTNodes() []string {
	return []string{
		"router.bittorrent.com:6881",
		"router.utorrent.com:6881",
		"dht.transmissionbt.com:6881",
	}
}

func resolveBootstrapDHTNodes(seeds []string) []string {
	resolved := make([]string, 0, len(seeds))
	seen := make(map[string]struct{})
	for _, seed := range seeds {
		host, port, err := net.SplitHostPort(strings.TrimSpace(seed))
		if err != nil {
			continue
		}
		ips, err := net.LookupIP(host)
		if err != nil {
			log.Printf("[dht-bootstrap-resolve-failed] host=%s err=%v", host, err)
			continue
		}
		for _, ip := range ips {
			if ip == nil {
				continue
			}
			v4 := ip.To4()
			if v4 == nil {
				continue
			}
			node := net.JoinHostPort(v4.String(), port)
			if _, exists := seen[node]; exists {
				continue
			}
			seen[node] = struct{}{}
			resolved = append(resolved, node)
		}
	}
	return resolved
}

type trackedStreamReader struct {
	io.ReadSeekCloser
	closeFn func()
	once    sync.Once
}

func (reader *trackedStreamReader) Close() error {
	err := reader.ReadSeekCloser.Close()
	reader.once.Do(reader.closeFn)
	return err
}
