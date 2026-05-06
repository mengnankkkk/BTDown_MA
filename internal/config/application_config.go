package config

import (
	"fmt"
	"os"
	"path/filepath"
)

type ApplicationSettings struct {
	PlayerPath                           string   `json:"playerPath"`
	TorrentDataDir                       string   `json:"torrentDataDir"`
	LogDir                               string   `json:"logDir"`
	AutoCleanupEnabled                   bool     `json:"autoCleanupEnabled"`
	AutoCleanupPolicy                    string   `json:"autoCleanupPolicy"`
	StreamBaseURL                        string   `json:"streamBaseUrl"`
	PublicTrackers                       []string `json:"publicTrackers"`
	AppendPublicTrackersForPublicTorrent bool     `json:"appendPublicTrackersForPublicTorrent"`
	BTListenPort                         int      `json:"btListenPort"`
	DownloadRateLimitKiBps               int      `json:"downloadRateLimitKiBps"`
	UploadRateLimitKiBps                 int      `json:"uploadRateLimitKiBps"`
	EnablePortForwarding                 bool     `json:"enablePortForwarding"`
	MetadataProxyEnabled                 bool     `json:"metadataProxyEnabled"`
	MetadataProxyURL                     string   `json:"metadataProxyUrl"`
	MetadataFallbackSources             []string `json:"metadataFallbackSources"`
	StreamDynamicReadaheadEnabled        bool     `json:"streamDynamicReadaheadEnabled"`
	StreamReadaheadMinBytes              int64    `json:"streamReadaheadMinBytes"`
	StreamReadaheadMaxBytes              int64    `json:"streamReadaheadMaxBytes"`
	StreamPreheatHeadPieces              int      `json:"streamPreheatHeadPieces"`
	StreamPreheatTailPieces              int      `json:"streamPreheatTailPieces"`
	StreamSeekGapFactor                  float64  `json:"streamSeekGapFactor"`
	StreamBoostWindowPieces              int      `json:"streamBoostWindowPieces"`
	StreamDeprioritizeOldWindow          bool     `json:"streamDeprioritizeOldWindow"`
}

type ApplicationConfig struct {
	ServerAddress string
	SettingsFile  string
	Settings      ApplicationSettings
}

func LoadApplicationConfig() ApplicationConfig {
	serverAddress := stringsTrimOrDefault(os.Getenv("BTDOWN_SERVER_ADDRESS"), ":18080")
	dataDir := stringsTrimOrDefault(os.Getenv("BTDOWN_TORRENT_DATA_DIR"), defaultTorrentDataDir())
	settingsFile := stringsTrimOrDefault(os.Getenv("BTDOWN_SETTINGS_FILE"), defaultSettingsFile())
	logDir := stringsTrimOrDefault(os.Getenv("BTDOWN_LOG_DIR"), defaultLogDir())
	streamBaseURL := stringsTrimOrDefault(os.Getenv("BTDOWN_STREAM_BASE_URL"), fmt.Sprintf("http://127.0.0.1%s", serverAddress))

	return ApplicationConfig{
		ServerAddress: serverAddress,
		SettingsFile:  settingsFile,
		Settings: ApplicationSettings{
			PlayerPath:                           "",
			TorrentDataDir:                       dataDir,
			LogDir:                               logDir,
			AutoCleanupEnabled:                   false,
			AutoCleanupPolicy:                    "manual",
			StreamBaseURL:                        streamBaseURL,
			PublicTrackers:                       defaultPublicTrackers(),
			AppendPublicTrackersForPublicTorrent: true,
			BTListenPort:                         51413,
			DownloadRateLimitKiBps:               0,
			UploadRateLimitKiBps:                 128,
			EnablePortForwarding:                 true,
			MetadataProxyEnabled:                 false,
			MetadataProxyURL:                     "",
			MetadataFallbackSources:              defaultMetadataFallbackSources(),
			StreamDynamicReadaheadEnabled:        true,
			StreamReadaheadMinBytes:              2 << 20,
			StreamReadaheadMaxBytes:              16 << 20,
			StreamPreheatHeadPieces:              8,
			StreamPreheatTailPieces:              8,
			StreamSeekGapFactor:                  1.0,
			StreamBoostWindowPieces:              12,
			StreamDeprioritizeOldWindow:          true,
		},
	}
}

func defaultPublicTrackers() []string {
	return []string{
		"udp://tracker.opentrackr.org:1337/announce",
		"udp://open.stealth.si:80/announce",
		"udp://tracker.torrent.eu.org:451/announce",
		"https://tracker.opentrackr.org/announce",
	}
}

func defaultMetadataFallbackSources() []string {
	return []string{
		"https://itorrents.org/torrent/{INFO_HASH_UPPER}.torrent",
		"https://itorrents.org/torrent/{INFO_HASH_LOWER}.torrent",
		"https://torrage.info/torrent.php?h={INFO_HASH_UPPER}",
	}
}

func defaultTorrentDataDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", "data", "downloads")
	}
	return filepath.Join(homeDir, "Downloads", "BTDown_MA")
}

func defaultLogDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", "data", "logs")
	}
	return filepath.Join(homeDir, "AppData", "Local", "BTDown_MA", "logs")
}

func defaultSettingsFile() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", "data", "settings.json")
	}
	return filepath.Join(homeDir, "AppData", "Local", "BTDown_MA", "settings.json")
}

func stringsTrimOrDefault(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
