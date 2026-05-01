package vo

type ObservabilityOverviewResponse struct {
	SessionCount                     int                  `json:"sessionCount"`
	StatusCounts                     map[string]int       `json:"statusCounts"`
	TotalDownloadSpeedBytesPerSecond int64                `json:"totalDownloadSpeedBytesPerSecond"`
	ActivePeersTotal                 int                  `json:"activePeersTotal"`
	AverageFirstFrameLatencyMs       int64                `json:"averageFirstFrameLatencyMs"`
	AverageSeekRecoveryMs            int64                `json:"averageSeekRecoveryMs"`
	AverageBufferHitRatio            float64              `json:"averageBufferHitRatio"`
	RecentStreamAccesses             []StreamAccessRecord `json:"recentStreamAccesses"`
}

type StreamAccessRecord struct {
	At           string `json:"at"`
	SessionID    string `json:"sessionId"`
	Method       string `json:"method"`
	Range        string `json:"range"`
	Status       int    `json:"status"`
	DurationMs   int64  `json:"durationMs"`
	ContentRange string `json:"contentRange"`
}
