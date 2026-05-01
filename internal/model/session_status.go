package model

type SessionStatus string

const (
	SessionStatusInitializing    SessionStatus = "INITIALIZING"
	SessionStatusWaitingMetadata SessionStatus = "WAITING_METADATA"
	SessionStatusDownloading     SessionStatus = "DOWNLOADING"
	SessionStatusReady           SessionStatus = "READY"
	SessionStatusStreaming       SessionStatus = "STREAMING"
	SessionStatusStalled         SessionStatus = "STALLED"
	SessionStatusCompleted       SessionStatus = "COMPLETED"
	SessionStatusError           SessionStatus = "ERROR"
)
