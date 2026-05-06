package vo

type ObservabilityOverviewResponse struct {
	SessionCount                     int                  `json:"sessionCount"`
	StatusCounts                     map[string]int       `json:"statusCounts"`
	TotalDownloadSpeedBytesPerSecond int64                `json:"totalDownloadSpeedBytesPerSecond"`
	ActivePeersTotal                 int                  `json:"activePeersTotal"`
	AverageFirstFrameLatencyMs       int64                `json:"averageFirstFrameLatencyMs"`
	AverageSeekRecoveryMs            int64                `json:"averageSeekRecoveryMs"`
	AverageBufferHitRatio            float64              `json:"averageBufferHitRatio"`
	AverageRecoverySuccessRate       float64              `json:"averageRecoverySuccessRate"`
	RecoveryLatencyDistribution      map[string]int64     `json:"recoveryLatencyDistribution"`
	RecentStreamAccesses             []StreamAccessRecord `json:"recentStreamAccesses"`
	Trend5m                          []TrendPoint         `json:"trend5m"`
}

type TrendPoint struct {
	At                 string  `json:"at"`
	RangeRequestCount  int     `json:"rangeRequestCount"`
	AvgRangeDurationMs int64   `json:"avgRangeDurationMs"`
	AvgSeekRecoveryMs  int64   `json:"avgSeekRecoveryMs"`
	AvgBufferHitRatio  float64 `json:"avgBufferHitRatio"`
	ActiveSessions     int     `json:"activeSessions"`
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
