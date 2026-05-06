package service

import (
	"BTDown_MA/internal/model"
	"BTDown_MA/internal/vo"
	"sort"
	"time"
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

	recentRecords := service.streamAccessBuffer.ListRecent()
	trend5m := buildTrend5m(sessions, recentRecords)

	return vo.ObservabilityOverviewResponse{
		SessionCount:                     len(sessions),
		StatusCounts:                     statusCounts,
		TotalDownloadSpeedBytesPerSecond: totalSpeed,
		ActivePeersTotal:                 activePeersTotal,
		AverageFirstFrameLatencyMs:       averageInt64(totalFirstFrameLatency, firstFrameCount),
		AverageSeekRecoveryMs:            averageInt64(totalSeekRecoveryMs, seekRecoveryCount),
		AverageBufferHitRatio:            averageFloat64(totalBufferHitRatio, bufferHitRatioCount),
		RecentStreamAccesses:             toStreamAccessResponseList(recentRecords),
		Trend5m:                          trend5m,
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

func buildTrend5m(sessions []model.Session, records []StreamAccessRecord) []vo.TrendPoint {
	const bucketSeconds = 30
	const bucketCount = 10

	if len(records) == 0 {
		return make([]vo.TrendPoint, 0)
	}

	now := time.Now()
	windowStart := now.Add(-5 * time.Minute)

	type bucketStat struct {
		at                  time.Time
		rangeRequestCount   int
		totalRangeDuration  int64
		totalSeekRecoveryMs int64
		seekRecoveryCount   int
		totalBufferHitRatio float64
		bufferHitRatioCount int
	}

	buckets := make(map[int64]*bucketStat)
	for _, record := range records {
		if record.At.Before(windowStart) {
			continue
		}
		bucketStartUnix := record.At.Unix() - (record.At.Unix() % bucketSeconds)
		bucket, ok := buckets[bucketStartUnix]
		if !ok {
			bucket = &bucketStat{at: time.Unix(bucketStartUnix, 0)}
			buckets[bucketStartUnix] = bucket
		}
		bucket.rangeRequestCount++
		bucket.totalRangeDuration += record.DurationMs

		if session := findSessionByID(sessions, record.SessionID); session != nil {
			if session.SeekRecoveryMs > 0 {
				bucket.totalSeekRecoveryMs += session.SeekRecoveryMs
				bucket.seekRecoveryCount++
			}
			if session.BufferHitRatio > 0 {
				bucket.totalBufferHitRatio += session.BufferHitRatio
				bucket.bufferHitRatioCount++
			}
		}
	}

	keys := make([]int64, 0, len(buckets))
	for key := range buckets {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	if len(keys) > bucketCount {
		keys = keys[len(keys)-bucketCount:]
	}

	result := make([]vo.TrendPoint, 0, len(keys))
	for _, key := range keys {
		bucket := buckets[key]
		result = append(result, vo.TrendPoint{
			At:                 bucket.at.Format("2006-01-02 15:04:05"),
			RangeRequestCount:  bucket.rangeRequestCount,
			AvgRangeDurationMs: averageInt64(bucket.totalRangeDuration, bucket.rangeRequestCount),
			AvgSeekRecoveryMs:  averageInt64(bucket.totalSeekRecoveryMs, bucket.seekRecoveryCount),
			AvgBufferHitRatio:  averageFloat64(bucket.totalBufferHitRatio, bucket.bufferHitRatioCount),
			ActiveSessions:     len(sessions),
		})
	}
	return result
}

func findSessionByID(sessions []model.Session, sessionID string) *model.Session {
	for index := range sessions {
		if sessions[index].ID == sessionID {
			return &sessions[index]
		}
	}
	return nil
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
