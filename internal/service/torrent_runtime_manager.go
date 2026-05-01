package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"BTDown_MA/internal/config"
	"BTDown_MA/internal/model"
	"BTDown_MA/internal/repository"

	"github.com/anacrolix/dht/v2"
	"github.com/anacrolix/torrent"
	"golang.org/x/time/rate"
)

const (
	metadataResolveTimeout = 2 * time.Minute
	metricsRefreshInterval = time.Second
	streamReaderReadahead  = 8 << 20
	streamPreheatPieces    = 8
	bufferHitThreshold     = 200 * time.Millisecond
)

type TorrentRuntimeManager = torrentRuntimeManager

type torrentRuntimeManager struct {
	sessionRepository repository.SessionRepository
	client            *torrent.Client
	settings          config.ApplicationSettings

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
	lastUsefulDataBytes      int64
	lastUsefulBytesDelta     int64
	lastStatsAt              time.Time
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
	torrentPublicity         string
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

	return &torrentRuntimeManager{
		sessionRepository: sessionRepository,
		client:            client,
		settings:          settings,
		runtimes:          make(map[string]*sessionRuntime),
	}, nil
}

func (manager *torrentRuntimeManager) startSession(session model.Session) {
	go manager.bootstrapSession(session)
}

func (manager *torrentRuntimeManager) bootstrapSession(session model.Session) {
	torrentHandle, err := manager.client.AddMagnet(session.MagnetURI)
	if err != nil {
		manager.failSession(session.ID, fmt.Sprintf("添加 magnet 失败: %v", err))
		return
	}

	runtimeCtx, runtimeCancel := context.WithCancel(context.Background())
	runtime := &sessionRuntime{
		sessionID:            session.ID,
		torrent:              torrentHandle,
		ctx:                  runtimeCtx,
		cancel:               runtimeCancel,
		lastStatsAt:          time.Now(),
		originalTrackerCount: trackerCount(torrentHandle),
		torrentPublicity:     "unknown",
	}

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
		current.TrackerCount = runtime.originalTrackerCount
		current.OriginalTrackerCount = runtime.originalTrackerCount
		current.AppendedTrackerCount = runtime.appendedTrackerCount
		current.TorrentPublicity = runtime.torrentPublicity
		current.ListenPort = manager.settings.BTListenPort
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
	ctx, cancel := context.WithTimeout(context.Background(), metadataResolveTimeout)
	defer cancel()

	select {
	case <-runtime.torrent.GotInfo():
	case <-runtime.ctx.Done():
		return
	case <-ctx.Done():
		manager.failSession(runtime.sessionID, "等待 metadata 超时")
		return
	}

	descriptor := selectPrimarySessionFile(runtime.torrent.Files())
	if descriptor == nil || descriptor.file == nil {
		manager.failSession(runtime.sessionID, "未找到可播放的文件")
		return
	}

	descriptor.file.SetPriority(torrent.PiecePriorityHigh)
	descriptor.file.Download()
	preheatFilePieces(runtime.torrent, descriptor.file)

	runtime.mutex.Lock()
	runtime.torrentPublicity = torrentPublicity(runtime.torrent)
	if manager.settings.AppendPublicTrackersForPublicTorrent && runtime.torrentPublicity == "public" {
		runtime.appendedTrackerCount = appendPublicTrackers(runtime.torrent, manager.settings.PublicTrackers)
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
		current.ListenPort = manager.settings.BTListenPort
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

		runtime.mutex.Lock()
		now := time.Now()
		elapsed := now.Sub(runtime.lastStatsAt).Seconds()
		if elapsed <= 0 {
			elapsed = 1
		}
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
			current.ListenPort = manager.settings.BTListenPort
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
	reader.SetReadahead(streamReaderReadahead)

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

func (manager *torrentRuntimeManager) RecordRangeActivity(sessionID, rangeHeader string, statusCode int, duration time.Duration) {
	manager.recordRangeActivity(sessionID, rangeHeader, statusCode, duration)
}

func (manager *torrentRuntimeManager) recordRangeActivity(sessionID, rangeHeader string, statusCode int, duration time.Duration) {
	runtime, ok := manager.getRuntime(sessionID)
	if !ok {
		return
	}

	now := time.Now()
	runtime.mutex.Lock()
	runtime.lastRangeRequestAt = now
	runtime.lastRangeResponseAt = now.Add(duration)
	runtime.lastRangeDurationMs = duration.Milliseconds()
	if statusCode >= 200 && statusCode < 300 {
		runtime.successfulStreamRequests++
		if duration <= bufferHitThreshold {
			runtime.bufferHitRequests++
		}
		if runtime.firstFrameLatencyMs == 0 {
			runtime.firstFrameLatencyMs = duration.Milliseconds()
		}
		rangeStart := parseRangeStart(rangeHeader)
		if runtime.hasLastRangeStart {
			gap := absInt64(rangeStart - runtime.lastRangeStart)
			if gap > streamReaderReadahead {
				runtime.seekRecoveryMs = duration.Milliseconds()
			}
		}
		runtime.lastRangeStart = rangeStart
		runtime.hasLastRangeStart = true
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
		switch stats := server.Stats().(type) {
		case dht.ServerStats:
			if stats.Nodes > 0 {
				count += stats.Nodes
			}
		case *dht.ServerStats:
			if stats != nil && stats.Nodes > 0 {
				count += stats.Nodes
			}
		}
	}
	return count
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

func preheatFilePieces(torrentHandle *torrent.Torrent, file *torrent.File) {
	begin := file.BeginPieceIndex()
	end := file.EndPieceIndex()
	if begin >= end {
		return
	}

	headEnd := minInt(begin+streamPreheatPieces, end)
	torrentHandle.DownloadPieces(begin, headEnd)

	tailStart := maxInt(begin, end-streamPreheatPieces)
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

func absInt64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
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
