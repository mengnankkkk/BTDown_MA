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
	settings.AutoCleanupPolicy = strings.TrimSpace(settings.AutoCleanupPolicy)
	if settings.AutoCleanupPolicy == "" {
		settings.AutoCleanupPolicy = "manual"
	}
	settings.PublicTrackers = normalizeTrackerList(settings.PublicTrackers)
	if settings.UploadRateLimitKiBps <= 0 {
		settings.UploadRateLimitKiBps = minimumUploadRateLimitKiBps
	}
	if settings.BTListenPort < 0 {
		settings.BTListenPort = 0
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
	if settings.StreamBaseURL == "" {
		return fmt.Errorf("streamBaseUrl 不能为空")
	}
	if settings.BTListenPort < 0 || settings.BTListenPort > 65535 {
		return fmt.Errorf("btListenPort 必须在 0-65535 之间")
	}
	if settings.DownloadRateLimitKiBps < 0 {
		return fmt.Errorf("downloadRateLimitKiBps 不能小于 0")
	}
	if settings.UploadRateLimitKiBps < minimumUploadRateLimitKiBps {
		return fmt.Errorf("uploadRateLimitKiBps 不能小于 %d", minimumUploadRateLimitKiBps)
	}
	switch settings.AutoCleanupPolicy {
	case "manual", "onSessionDelete":
	default:
		return fmt.Errorf("autoCleanupPolicy 不支持: %s", settings.AutoCleanupPolicy)
	}
	return nil
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
