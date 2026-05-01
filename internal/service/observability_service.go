package service

import (
	"BTDown_MA/internal/model"
	"BTDown_MA/internal/vo"
)

type ObservabilityService struct {
	sessionService     *SessionService
	streamAccessBuffer *StreamAccessBuffer
}

func NewObservabilityService(sessionService *SessionService, streamAccessBuffer *StreamAccessBuffer) *ObservabilityService {
	return &ObservabilityService{
		sessionService:     sessionService,
		streamAccessBuffer: streamAccessBuffer,
	}
}

func (service *ObservabilityService) GetOverview() vo.ObservabilityOverviewResponse {
	sessions := service.sessionService.ListSessions()
	statusCounts := make(map[string]int)
	var totalSpeed int64
	activePeersTotal := 0
	var totalFirstFrameLatency int64
	firstFrameCount := 0
	var totalSeekRecoveryMs int64
	seekRecoveryCount := 0
	totalBufferHitRatio := 0.0
	bufferHitRatioCount := 0

	for _, session := range sessions {
		statusCounts[string(session.Status)]++
		totalSpeed += session.DownloadSpeedBytesPerSecond
		activePeersTotal += session.ActivePeers
		if session.FirstFrameLatencyMs > 0 {
			totalFirstFrameLatency += session.FirstFrameLatencyMs
			firstFrameCount++
		}
		if session.SeekRecoveryMs > 0 {
			totalSeekRecoveryMs += session.SeekRecoveryMs
			seekRecoveryCount++
		}
		if session.BufferHitRatio > 0 {
			totalBufferHitRatio += session.BufferHitRatio
			bufferHitRatioCount++
		}
	}

	return vo.ObservabilityOverviewResponse{
		SessionCount:                     len(sessions),
		StatusCounts:                     statusCounts,
		TotalDownloadSpeedBytesPerSecond: totalSpeed,
		ActivePeersTotal:                 activePeersTotal,
		AverageFirstFrameLatencyMs:       averageInt64(totalFirstFrameLatency, firstFrameCount),
		AverageSeekRecoveryMs:            averageInt64(totalSeekRecoveryMs, seekRecoveryCount),
		AverageBufferHitRatio:            averageFloat64(totalBufferHitRatio, bufferHitRatioCount),
		RecentStreamAccesses:             toStreamAccessResponseList(service.streamAccessBuffer.ListRecent()),
	}
}

func averageInt64(total int64, count int) int64 {
	if count <= 0 {
		return 0
	}
	return total / int64(count)
}

func averageFloat64(total float64, count int) float64 {
	if count <= 0 {
		return 0
	}
	return total / float64(count)
}

func toStreamAccessResponseList(records []StreamAccessRecord) []vo.StreamAccessRecord {
	result := make([]vo.StreamAccessRecord, 0, len(records))
	for _, record := range records {
		result = append(result, vo.StreamAccessRecord{
			At:           record.At.Format("2006-01-02 15:04:05"),
			SessionID:    record.SessionID,
			Method:       record.Method,
			Range:        record.Range,
			Status:       record.Status,
			DurationMs:   record.DurationMs,
			ContentRange: record.ContentRange,
		})
	}
	return result
}

func shouldAutoCleanup(settingsPolicy string, enabled bool, _ model.Session) bool {
	return enabled && settingsPolicy == "onSessionDelete"
}
