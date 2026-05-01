package vo

type SessionHealthReportResponse struct {
	Summary string `json:"summary"`
	Reason  string `json:"reason"`
}

type SessionHealthEvidenceResponse struct {
	Type        string `json:"type"`
	Code        string `json:"code"`
	Severity    string `json:"severity"`
	Detail      string `json:"detail"`
	FirstSeenAt string `json:"firstSeenAt"`
	LastSeenAt  string `json:"lastSeenAt"`
	Count       int    `json:"count"`
}

type SessionHealthDiagnosisResponse struct {
	Window           string                          `json:"window"`
	AvailabilityTier string                          `json:"availabilityTier"`
	Confidence       int                             `json:"confidence"`
	Evidences        []SessionHealthEvidenceResponse `json:"evidences"`
	UpdatedAt        string                          `json:"updatedAt"`
}

type SessionMetricsResponse struct {
	TrackerCount                int     `json:"trackerCount"`
	OriginalTrackerCount        int     `json:"originalTrackerCount"`
	AppendedTrackerCount        int     `json:"appendedTrackerCount"`
	TorrentPublicity            string  `json:"torrentPublicity"`
	ListenPort                  int     `json:"listenPort"`
	DHTStatus                   string  `json:"dhtStatus"`
	DHTNodes                    int     `json:"dhtNodes"`
	UDPReachable                string  `json:"udpReachable"`
	IncomingConnections         int     `json:"incomingConnections"`
	ActivePeers                 int     `json:"activePeers"`
	TotalPeers                  int     `json:"totalPeers"`
	DownloadSpeedBytesPerSecond int64   `json:"downloadSpeedBytesPerSecond"`
	UsefulBytesDelta            int64   `json:"usefulBytesDelta"`
	DownloadedBytes             int64   `json:"downloadedBytes"`
	TotalBytes                  int64   `json:"totalBytes"`
	DownloadProgress            float64 `json:"downloadProgress"`
	LastRangeResponseDurationMs int64   `json:"lastRangeResponseDurationMs"`
	FirstFrameLatencyMs         int64   `json:"firstFrameLatencyMs"`
	SeekRecoveryMs              int64   `json:"seekRecoveryMs"`
	BufferHitRatio              float64 `json:"bufferHitRatio"`
	DownloadSpeedText           string  `json:"downloadSpeedText"`
	StreamStateText             string  `json:"streamStateText"`
	DeadTorrentStateText        string  `json:"deadTorrentStateText"`
}

type SessionResponse struct {
	ID               string                         `json:"id"`
	Name             string                         `json:"name"`
	MagnetURI        string                         `json:"magnetUri"`
	Status           string                         `json:"status"`
	StreamURL        string                         `json:"streamUrl"`
	MetadataState    string                         `json:"metadataState"`
	DownloadState    string                         `json:"downloadState"`
	StreamState      string                         `json:"streamState"`
	DeadState        string                         `json:"deadState"`
	HealthReport     SessionHealthReportResponse    `json:"healthReport"`
	HealthDiagnosis  SessionHealthDiagnosisResponse `json:"healthDiagnosis"`
	SelectedFileName string                         `json:"selectedFileName,omitempty"`
	LastError        string                         `json:"lastError,omitempty"`
	Metrics          SessionMetricsResponse         `json:"metrics"`
}
