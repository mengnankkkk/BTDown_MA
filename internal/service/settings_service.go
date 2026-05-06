package service

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"BTDown_MA/internal/config"
)

const minimumUploadRateLimitKiBps = 16

var fallbackPublicTrackers = []string{
	"udp://tracker.opentrackr.org:1337/announce",
	"udp://open.stealth.si:80/announce",
	"udp://tracker.torrent.eu.org:451/announce",
	"https://tracker.opentrackr.org/announce",
}

type SettingsService struct {
	settingsFile string

	mutex    sync.RWMutex
	settings config.ApplicationSettings
}

func NewSettingsService(settingsFile string, defaults config.ApplicationSettings) *SettingsService {
	service := &SettingsService{
		settingsFile: settingsFile,
		settings:     normalizeSettings(defaults),
	}
	if err := service.load(); err != nil {
		log.Printf("load settings failed: %v", err)
	}
	return service
}

func (service *SettingsService) GetSettings() config.ApplicationSettings {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	return service.settings
}

func (service *SettingsService) UpdateSettings(next config.ApplicationSettings) (config.ApplicationSettings, error) {
	normalized := normalizeSettings(next)
	if err := validateSettings(normalized); err != nil {
		return config.ApplicationSettings{}, err
	}
	if err := service.persist(normalized); err != nil {
		return config.ApplicationSettings{}, err
	}

	service.mutex.Lock()
	service.settings = normalized
	service.mutex.Unlock()
	return normalized, nil
}

func (service *SettingsService) load() error {
	payload, err := os.ReadFile(service.settingsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("读取设置文件失败: %w", err)
	}

	var loaded config.ApplicationSettings
	if err := json.Unmarshal(payload, &loaded); err != nil {
		return fmt.Errorf("解析设置文件失败: %w", err)
	}

	normalized := normalizeSettings(loaded)
	if err := validateSettings(normalized); err != nil {
		return fmt.Errorf("设置文件内容不合法: %w", err)
	}

	service.mutex.Lock()
	service.settings = normalized
	service.mutex.Unlock()
	return nil
}

func (service *SettingsService) persist(next config.ApplicationSettings) error {
	if err := os.MkdirAll(filepath.Dir(service.settingsFile), 0o755); err != nil {
		return fmt.Errorf("创建设置目录失败: %w", err)
	}

	payload, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return fmt.Errorf("编码设置失败: %w", err)
	}
	if err := os.WriteFile(service.settingsFile, payload, 0o644); err != nil {
		return fmt.Errorf("写入设置文件失败: %w", err)
	}
	return nil
}

func normalizeSettings(settings config.ApplicationSettings) config.ApplicationSettings {
	settings.PlayerPath = strings.TrimSpace(settings.PlayerPath)
	settings.TorrentDataDir = strings.TrimSpace(settings.TorrentDataDir)
	settings.LogDir = strings.TrimSpace(settings.LogDir)
	settings.StreamBaseURL = strings.TrimSpace(settings.StreamBaseURL)
	settings.MetadataProxyURL = strings.TrimSpace(settings.MetadataProxyURL)
	settings.MetadataFallbackSources = normalizeMetadataFallbackSourceList(settings.MetadataFallbackSources)
	settings.AutoCleanupPolicy = strings.TrimSpace(settings.AutoCleanupPolicy)
	if settings.AutoCleanupPolicy == "" {
		settings.AutoCleanupPolicy = "manual"
	}
	settings.PublicTrackers = normalizeTrackerList(settings.PublicTrackers)
	if len(settings.PublicTrackers) == 0 {
		settings.PublicTrackers = append([]string(nil), fallbackPublicTrackers...)
	}
	if settings.UploadRateLimitKiBps <= 0 {
		settings.UploadRateLimitKiBps = minimumUploadRateLimitKiBps
	}
	if settings.StreamReadaheadMinBytes <= 0 {
		settings.StreamReadaheadMinBytes = 2 << 20
	}
	if settings.StreamReadaheadMaxBytes <= 0 {
		settings.StreamReadaheadMaxBytes = 16 << 20
	}
	if settings.StreamReadaheadMaxBytes < settings.StreamReadaheadMinBytes {
		settings.StreamReadaheadMaxBytes = settings.StreamReadaheadMinBytes
	}
	if settings.StreamPreheatHeadPieces <= 0 {
		settings.StreamPreheatHeadPieces = 8
	}
	if settings.StreamPreheatTailPieces <= 0 {
		settings.StreamPreheatTailPieces = 8
	}
	if settings.StreamSeekGapFactor <= 0 {
		settings.StreamSeekGapFactor = 1.0
	}
	if settings.StreamBoostWindowPieces <= 0 {
		settings.StreamBoostWindowPieces = 12
	}
	if settings.BTListenPort <= 0 {
		settings.BTListenPort = 51413
	}
	return settings
}

func validateSettings(settings config.ApplicationSettings) error {
	if settings.TorrentDataDir == "" {
		return fmt.Errorf("torrentDataDir 不能为空")
	}
	if settings.LogDir == "" {
		return fmt.Errorf("logDir 不能为空")
	}
	if settings.MetadataProxyEnabled && settings.MetadataProxyURL == "" {
		return fmt.Errorf("metadataProxyEnabled=true 时 metadataProxyUrl 不能为空")
	}
	if settings.MetadataProxyEnabled {
		if !strings.HasPrefix(settings.MetadataProxyURL, "http://") && !strings.HasPrefix(settings.MetadataProxyURL, "https://") {
			return fmt.Errorf("metadataProxyUrl 必须以 http:// 或 https:// 开头")
		}
	}
	for _, source := range settings.MetadataFallbackSources {
		if !strings.HasPrefix(source, "http://") && !strings.HasPrefix(source, "https://") {
			return fmt.Errorf("metadataFallbackSources 仅支持 http/https: %s", source)
		}
	}
	if settings.DownloadRateLimitKiBps < 0 {
		return fmt.Errorf("downloadRateLimitKiBps 不能小于 0")
	}
	if settings.UploadRateLimitKiBps < minimumUploadRateLimitKiBps {
		return fmt.Errorf("uploadRateLimitKiBps 不能小于 %d", minimumUploadRateLimitKiBps)
	}
	if settings.StreamReadaheadMinBytes <= 0 {
		return fmt.Errorf("streamReadaheadMinBytes 必须大于 0")
	}
	if settings.StreamReadaheadMaxBytes < settings.StreamReadaheadMinBytes {
		return fmt.Errorf("streamReadaheadMaxBytes 不能小于 streamReadaheadMinBytes")
	}
	if settings.StreamPreheatHeadPieces <= 0 {
		return fmt.Errorf("streamPreheatHeadPieces 必须大于 0")
	}
	if settings.StreamPreheatTailPieces <= 0 {
		return fmt.Errorf("streamPreheatTailPieces 必须大于 0")
	}
	if settings.StreamSeekGapFactor <= 0 {
		return fmt.Errorf("streamSeekGapFactor 必须大于 0")
	}
	if settings.StreamBoostWindowPieces <= 0 {
		return fmt.Errorf("streamBoostWindowPieces 必须大于 0")
	}
	return nil
}

func normalizeMetadataFallbackSourceList(sources []string) []string {
	seen := make(map[string]struct{})
	normalized := make([]string, 0, len(sources))
	for _, source := range sources {
		candidate := strings.TrimSpace(source)
		if candidate == "" {
			continue
		}
		if !strings.HasPrefix(candidate, "http://") && !strings.HasPrefix(candidate, "https://") {
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

func normalizeTrackerList(trackers []string) []string {
	seen := make(map[string]struct{})
	normalized := make([]string, 0, len(trackers))
	for _, tracker := range trackers {
		candidate := strings.TrimSpace(tracker)
		if candidate == "" {
			continue
		}
		if !strings.HasPrefix(candidate, "udp://") && !strings.HasPrefix(candidate, "http://") && !strings.HasPrefix(candidate, "https://") {
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
