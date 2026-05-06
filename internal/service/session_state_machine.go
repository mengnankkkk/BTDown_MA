package service

import (
	"strings"
	"time"

	"BTDown_MA/internal/model"
)

const (
	healthWindowFastMaxAge     = 30 * time.Second
	healthWindowStandardMaxAge = 2 * time.Minute
	metadataSoftTimeout        = 30 * time.Second
	inactiveRangeThreshold     = 10 * time.Second
)

func refreshSessionDerivedFields(session *model.Session, activeStreamRequests int) {
	session.UpdatedAt = time.Now()
	refreshSessionHealthReport(session, activeStreamRequests)

	switch {
	case session.LastError != "":
		session.Status = model.SessionStatusError
	case session.MetadataState != model.SessionMetadataStateReady:
		session.Status = model.SessionStatusWaitingMetadata
	case session.StreamState == model.SessionStreamStateStreaming || activeStreamRequests > 0:
		session.Status = model.SessionStatusStreaming
	case session.DeadState == model.SessionDeadStateNoPeers && session.ActivePeers == 0:
		session.Status = model.SessionStatusStalled
	case session.DeadState == model.SessionDeadStateNoUsefulTransfer:
		session.Status = model.SessionStatusStalled
	case session.TotalBytes > 0 && session.DownloadedBytes >= session.TotalBytes:
		session.Status = model.SessionStatusCompleted
	case session.DownloadState == model.SessionDownloadStatePaused:
		session.Status = model.SessionStatusReady
	case session.DownloadState == model.SessionDownloadStateDownloading:
		session.Status = model.SessionStatusDownloading
	default:
		session.Status = model.SessionStatusReady
	}

	if activeStreamRequests > 0 {
		session.StreamState = model.SessionStreamStateStreaming
	} else if session.StreamState == model.SessionStreamStateStreaming {
		if session.MetadataState == model.SessionMetadataStateReady {
			session.StreamState = model.SessionStreamStateReady
		} else {
			session.StreamState = model.SessionStreamStateUnavailable
		}
	}
}

func refreshSessionHealthReport(session *model.Session, activeStreamRequests int) {
	now := time.Now()
	window := detectHealthWindow(session.CreatedAt, now)
	evidences := buildHealthEvidences(session, now)
	confidence := calculateHealthConfidence(session, window, evidences, activeStreamRequests, now)
	tier := classifyAvailabilityTier(session, window, evidences, confidence)

	session.HealthDiagnosis = model.SessionHealthDiagnosis{
		Window:           window,
		AvailabilityTier: tier,
		Confidence:       confidence,
		Evidences:        evidences,
		UpdatedAt:        now,
	}

	applyLegacyHealthState(session, tier, evidences, activeStreamRequests, now)
}

func detectHealthWindow(createdAt, now time.Time) model.SessionHealthWindow {
	age := now.Sub(createdAt)
	if age <= healthWindowFastMaxAge {
		return model.SessionHealthWindowFast
	}
	if age <= healthWindowStandardMaxAge {
		return model.SessionHealthWindowStandard
	}
	return model.SessionHealthWindowDeep
}

func buildHealthEvidences(session *model.Session, now time.Time) []model.SessionHealthEvidence {
	evidences := make([]model.SessionHealthEvidence, 0, 6)
	age := now.Sub(session.CreatedAt)

	if session.MetadataState != model.SessionMetadataStateReady && age > metadataSoftTimeout {
		evidences = append(evidences, model.SessionHealthEvidence{
			Type:        "metadata",
			Code:        "metadata_timeout",
			Severity:    "critical",
			Detail:      "metadata 获取超时，已超过 30 秒",
			FirstSeenAt: session.CreatedAt.Add(metadataSoftTimeout),
			LastSeenAt:  now,
			Count:       positiveCount(age.Seconds() - metadataSoftTimeout.Seconds()),
		})
	}

	if session.TrackerCount == 0 || (session.AppendedTrackerCount > 0 && session.TotalPeers == 0 && age > healthWindowFastMaxAge) {
		evidences = append(evidences, model.SessionHealthEvidence{
			Type:        "tracker",
			Code:        "tracker_failed",
			Severity:    "warning",
			Detail:      "tracker 未提供有效 peer 发现结果",
			FirstSeenAt: session.CreatedAt,
			LastSeenAt:  now,
			Count:       positiveCount(age.Seconds()),
		})
	}

	if session.TotalPeers == 0 && (session.DHTNodes == 0 || strings.EqualFold(session.UDPReachable, "blocked")) {
		evidences = append(evidences, model.SessionHealthEvidence{
			Type:        "dht",
			Code:        "dht_no_peers",
			Severity:    "warning",
			Detail:      "DHT/UDP 未提供可用 peers",
			FirstSeenAt: session.CreatedAt,
			LastSeenAt:  now,
			Count:       positiveCount(age.Seconds()),
		})
	}

	if session.ActivePeers > 0 && session.UsefulBytesDelta == 0 {
		evidences = append(evidences, model.SessionHealthEvidence{
			Type:        "transfer",
			Code:        "no_useful_data",
			Severity:    "warning",
			Detail:      "存在活跃 peer，但最近没有 useful bytes 增量",
			FirstSeenAt: session.UpdatedAt,
			LastSeenAt:  now,
			Count:       1,
		})
	}

	if session.LastError != "" {
		evidences = append(evidences, model.SessionHealthEvidence{
			Type:        "runtime",
			Code:        "runtime_error",
			Severity:    "critical",
			Detail:      session.LastError,
			FirstSeenAt: now,
			LastSeenAt:  now,
			Count:       1,
		})
	}

	return evidences
}

func calculateHealthConfidence(
	session *model.Session,
	window model.SessionHealthWindow,
	evidences []model.SessionHealthEvidence,
	activeStreamRequests int,
	now time.Time,
) int {
	score := 50

	if session.MetadataState == model.SessionMetadataStateReady {
		score += 20
	}
	if session.ActivePeers > 0 {
		score += 15
	}
	if session.UsefulBytesDelta > 0 {
		score += 25
	}
	if activeStreamRequests > 0 || (!session.LastRangeRequestAt.IsZero() && now.Sub(session.LastRangeRequestAt) <= inactiveRangeThreshold) {
		score += 10
	}

	for _, evidence := range evidences {
		score -= penaltyByEvidence(evidence)
	}

	if window == model.SessionHealthWindowFast && score < 30 {
		score = 30
	}
	if window == model.SessionHealthWindowStandard && score < 20 {
		score = 20
	}
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return score
}

func penaltyByEvidence(evidence model.SessionHealthEvidence) int {
	switch evidence.Code {
	case "metadata_timeout":
		return 25
	case "tracker_failed":
		return 15
	case "dht_no_peers":
		return 20
	case "no_useful_data":
		return 20
	case "runtime_error":
		return 30
	default:
		if evidence.Severity == "critical" {
			return 20
		}
		return 10
	}
}

func classifyAvailabilityTier(
	session *model.Session,
	window model.SessionHealthWindow,
	evidences []model.SessionHealthEvidence,
	confidence int,
) model.SessionAvailabilityTier {
	if session.LastError != "" {
		return model.SessionAvailabilityTierHighUnavailable
	}
	if session.UsefulBytesDelta > 0 || session.ActivePeers > 0 {
		if confidence >= 70 {
			return model.SessionAvailabilityTierHighAvailable
		}
		return model.SessionAvailabilityTierWeakAvailable
	}

	metadataTimeout := hasEvidence(evidences, "metadata_timeout")
	trackerFailed := hasEvidence(evidences, "tracker_failed")
	dhtNoPeers := hasEvidence(evidences, "dht_no_peers")
	noUsefulData := hasEvidence(evidences, "no_useful_data")

	if window == model.SessionHealthWindowDeep && confidence <= 35 && (metadataTimeout || (dhtNoPeers && (trackerFailed || noUsefulData))) {
		return model.SessionAvailabilityTierHighUnavailable
	}
	if metadataTimeout || dhtNoPeers || trackerFailed {
		return model.SessionAvailabilityTierNoResource
	}
	return model.SessionAvailabilityTierWeakAvailable
}

func applyLegacyHealthState(
	session *model.Session,
	tier model.SessionAvailabilityTier,
	evidences []model.SessionHealthEvidence,
	activeStreamRequests int,
	now time.Time,
) {
	if session.LastError != "" {
		session.DeadState = model.SessionDeadStateMetadataStuck
		session.HealthReport = model.SessionHealthReport{Summary: "高概率不可用", Reason: session.LastError}
		return
	}

	switch tier {
	case model.SessionAvailabilityTierHighAvailable:
		session.DeadState = model.SessionDeadStateAlive
		session.HealthReport = model.SessionHealthReport{Summary: "高概率可用", Reason: "已发现可用 peer，且传输链路稳定"}
	case model.SessionAvailabilityTierWeakAvailable:
		if activeStreamRequests == 0 && !session.LastRangeRequestAt.IsZero() && now.Sub(session.LastRangeRequestAt) > inactiveRangeThreshold {
			session.DeadState = model.SessionDeadStateWaitingRange
			session.HealthReport = model.SessionHealthReport{Summary: "弱可用", Reason: "当前无新的 Range 请求，等待播放器继续拉流"}
			return
		}
		session.DeadState = model.SessionDeadStateAlive
		session.HealthReport = model.SessionHealthReport{Summary: "弱可用", Reason: "当前链路可用但信号偏弱，建议继续观察"}
	case model.SessionAvailabilityTierNoResource:
		if hasEvidence(evidences, "metadata_timeout") {
			session.DeadState = model.SessionDeadStateMetadataStuck
			session.HealthReport = model.SessionHealthReport{Summary: "暂未发现资源", Reason: "metadata/tracker/dht 仍未建立稳定发现链路"}
			return
		}
		session.DeadState = model.SessionDeadStateNoPeers
		session.HealthReport = model.SessionHealthReport{Summary: "暂未发现资源", Reason: "当前未发现可用 peer，建议继续等待标准窗口完成"}
	case model.SessionAvailabilityTierHighUnavailable:
		if hasEvidence(evidences, "no_useful_data") {
			session.DeadState = model.SessionDeadStateNoUsefulTransfer
			session.HealthReport = model.SessionHealthReport{Summary: "高概率不可用", Reason: "长时间无 useful 数据增量，疑似死链或弱资源"}
			return
		}
		session.DeadState = model.SessionDeadStateNoPeers
		session.HealthReport = model.SessionHealthReport{Summary: "高概率不可用", Reason: "多项负向证据持续成立，资源可用性较低"}
	default:
		session.DeadState = model.SessionDeadStateUnknown
		session.HealthReport = model.SessionHealthReport{Summary: "弱可用", Reason: "等待更多观测数据"}
	}
}

func hasEvidence(evidences []model.SessionHealthEvidence, code string) bool {
	for _, evidence := range evidences {
		if evidence.Code == code {
			return true
		}
	}
	return false
}

func positiveCount(value float64) int {
	if value <= 0 {
		return 1
	}
	count := int(value)
	if count < 1 {
		return 1
	}
	return count
}
