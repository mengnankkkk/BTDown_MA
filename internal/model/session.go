package model

import "time"

type Session struct {
	ID                          string
	MagnetURI                   string
	Name                        string
	Status                      SessionStatus
	StreamURL                   string
	MetadataState               SessionMetadataState
	DownloadState               SessionDownloadState
	StreamState                 SessionStreamState
	DeadState                   SessionDeadState
	HealthReport                SessionHealthReport
	HealthDiagnosis             SessionHealthDiagnosis
	SelectedFileName            string
	TrackerCount                int
	OriginalTrackerCount        int
	AppendedTrackerCount        int
	TorrentPublicity            string
	ListenPort                  int
	DHTStatus                   string
	DHTNodes                    int
	UDPReachable                string
	IncomingConnections         int
	ActivePeers                 int
	TotalPeers                  int
	DownloadSpeedBytesPerSecond int64
	UsefulBytesDelta            int64
	DownloadedBytes             int64
	TotalBytes                  int64
	LastRangeRequestAt          time.Time
	LastRangeResponseAt         time.Time
	LastRangeResponseDurationMs int64
	FirstFrameLatencyMs         int64
	SeekRecoveryMs              int64
	BufferHitRatio              float64
	LastError                   string
	CreatedAt                   time.Time
	UpdatedAt                   time.Time
}

type SessionMetadataState string

const (
	SessionMetadataStatePending   SessionMetadataState = "PENDING"
	SessionMetadataStateResolving SessionMetadataState = "RESOLVING"
	SessionMetadataStateReady     SessionMetadataState = "READY"
	SessionMetadataStateError     SessionMetadataState = "ERROR"
)

type SessionDownloadState string

const (
	SessionDownloadStateQueued      SessionDownloadState = "QUEUED"
	SessionDownloadStateDownloading SessionDownloadState = "DOWNLOADING"
	SessionDownloadStatePaused      SessionDownloadState = "PAUSED"
	SessionDownloadStateCompleted   SessionDownloadState = "COMPLETED"
	SessionDownloadStateError       SessionDownloadState = "ERROR"
)

type SessionStreamState string

const (
	SessionStreamStateUnavailable SessionStreamState = "UNAVAILABLE"
	SessionStreamStateReady       SessionStreamState = "READY"
	SessionStreamStateStreaming   SessionStreamState = "STREAMING"
	SessionStreamStateError       SessionStreamState = "ERROR"
)

type SessionDeadState string

const (
	SessionDeadStateUnknown          SessionDeadState = "UNKNOWN"
	SessionDeadStateAlive            SessionDeadState = "ALIVE"
	SessionDeadStateNoPeers          SessionDeadState = "NO_PEERS"
	SessionDeadStateNoUsefulTransfer SessionDeadState = "NO_USEFUL_TRANSFER"
	SessionDeadStateWaitingRange     SessionDeadState = "WAITING_RANGE"
	SessionDeadStateMetadataStuck    SessionDeadState = "METADATA_STUCK"
)

type SessionHealthReport struct {
	Summary string
	Reason  string
}

type SessionHealthWindow string

const (
	SessionHealthWindowFast     SessionHealthWindow = "FAST"
	SessionHealthWindowStandard SessionHealthWindow = "STANDARD"
	SessionHealthWindowDeep     SessionHealthWindow = "DEEP"
)

type SessionAvailabilityTier string

const (
	SessionAvailabilityTierHighAvailable   SessionAvailabilityTier = "HIGH_AVAILABLE"
	SessionAvailabilityTierWeakAvailable   SessionAvailabilityTier = "WEAK_AVAILABLE"
	SessionAvailabilityTierNoResource      SessionAvailabilityTier = "NO_RESOURCE"
	SessionAvailabilityTierHighUnavailable SessionAvailabilityTier = "HIGH_UNAVAILABLE"
)

type SessionHealthEvidence struct {
	Type        string
	Code        string
	Severity    string
	Detail      string
	FirstSeenAt time.Time
	LastSeenAt  time.Time
	Count       int
}

type SessionHealthDiagnosis struct {
	Window           SessionHealthWindow
	AvailabilityTier SessionAvailabilityTier
	Confidence       int
	Evidences        []SessionHealthEvidence
	UpdatedAt        time.Time
}
