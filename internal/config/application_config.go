package config

import (
	"os"
	"path/filepath"
)

type ApplicationSettings struct {
	PlayerPath                          string   `json:"playerPath"`
	TorrentDataDir                      string   `json:"torrentDataDir"`
	LogDir                              string   `json:"logDir"`
	AutoCleanupEnabled                  bool     `json:"autoCleanupEnabled"`
	AutoCleanupPolicy                   string   `json:"autoCleanupPolicy"`
	StreamBaseURL                       string   `json:"streamBaseUrl"`
	PublicTrackers                      []string `json:"publicTrackers"`
	AppendPublicTrackersForPublicTorrent bool    `json:"appendPublicTrackersForPublicTorrent"`
	BTListenPort                        int      `json:"btListenPort"`
	DownloadRateLimitKiBps              int      `json:"downloadRateLimitKiBps"`
	UploadRateLimitKiBps                int      `json:"uploadRateLimitKiBps"`
	EnablePortForwarding                bool     `json:"enablePortForwarding"`
}

type ApplicationConfig struct {
	ServerAddress string
	SettingsFile  string
	Settings      ApplicationSettings
}

func LoadApplicationConfig() ApplicationConfig {
	serverAddress := ":18080"
	dataDir := defaultTorrentDataDir()
	streamBaseURL := "http://127.0.0.1:18080"

	return ApplicationConfig{
		ServerAddress: serverAddress,
		SettingsFile:  defaultSettingsFile(),
		Settings: ApplicationSettings{
			PlayerPath:                          "",
			TorrentDataDir:                      dataDir,
			LogDir:                              defaultLogDir(),
			AutoCleanupEnabled:                  false,
			AutoCleanupPolicy:                   "manual",
			StreamBaseURL:                       streamBaseURL,
			PublicTrackers:                      defaultPublicTrackers(),
			AppendPublicTrackersForPublicTorrent: true,
			BTListenPort:                        51413,
			DownloadRateLimitKiBps:              0,
			UploadRateLimitKiBps:                128,
			EnablePortForwarding:                true,
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
