package mapper

import (
	"fmt"
	"strings"
	"time"

	"BTDown_MA/internal/model"
	"BTDown_MA/internal/vo"
)

func ToSessionResponse(session model.Session) vo.SessionResponse {
	return vo.SessionResponse{
		ID:            session.ID,
		Name:          session.Name,
		MagnetURI:     session.MagnetURI,
		Status:        string(session.Status),
		StreamURL:     session.StreamURL,
		MetadataState: string(session.MetadataState),
		DownloadState: string(session.DownloadState),
		StreamState:   string(session.StreamState),
		DeadState:     string(session.DeadState),
		HealthReport: vo.SessionHealthReportResponse{
			Summary: session.HealthReport.Summary,
			Reason:  session.HealthReport.Reason,
		},
		HealthDiagnosis:  toHealthDiagnosisResponse(session.HealthDiagnosis),
		SelectedFileName: session.SelectedFileName,
		LastError:        session.LastError,
		Metrics: vo.SessionMetricsResponse{
			TrackerCount:                session.TrackerCount,
			OriginalTrackerCount:        session.OriginalTrackerCount,
			AppendedTrackerCount:        session.AppendedTrackerCount,
			TorrentPublicity:            session.TorrentPublicity,
			ListenPort:                  session.ListenPort,
			DHTStatus:                   session.DHTStatus,
			DHTNodes:                    session.DHTNodes,
			UDPReachable:                session.UDPReachable,
			IncomingConnections:         session.IncomingConnections,
			ActivePeers:                 session.ActivePeers,
			TotalPeers:                  session.TotalPeers,
			DownloadSpeedBytesPerSecond: session.DownloadSpeedBytesPerSecond,
			UsefulBytesDelta:            session.UsefulBytesDelta,
			DownloadedBytes:             session.DownloadedBytes,
			TotalBytes:                  session.TotalBytes,
			DownloadProgress:            calculateDownloadProgress(session.DownloadedBytes, session.TotalBytes),
			LastRangeResponseDurationMs: session.LastRangeResponseDurationMs,
			FirstFrameLatencyMs:         session.FirstFrameLatencyMs,
			SeekRecoveryMs:              session.SeekRecoveryMs,
			BufferHitRatio:              session.BufferHitRatio,
			DownloadSpeedText:           formatBytesPerSecond(session.DownloadSpeedBytesPerSecond),
			StreamStateText:             formatStreamState(session),
			DeadTorrentStateText:        formatDeadState(session),
		},
	}
}

func ToSessionResponseList(sessions []model.Session) []vo.SessionResponse {
	result := make([]vo.SessionResponse, 0, len(sessions))
	for _, session := range sessions {
		result = append(result, ToSessionResponse(session))
	}

	return result
}

func toHealthDiagnosisResponse(diagnosis model.SessionHealthDiagnosis) vo.SessionHealthDiagnosisResponse {
	evidences := make([]vo.SessionHealthEvidenceResponse, 0, len(diagnosis.Evidences))
	for _, evidence := range diagnosis.Evidences {
		evidences = append(evidences, vo.SessionHealthEvidenceResponse{
			Type:        evidence.Type,
			Code:        evidence.Code,
			Severity:    evidence.Severity,
			Detail:      evidence.Detail,
			FirstSeenAt: formatTimestamp(evidence.FirstSeenAt),
			LastSeenAt:  formatTimestamp(evidence.LastSeenAt),
			Count:       evidence.Count,
		})
	}

	return vo.SessionHealthDiagnosisResponse{
		Window:           string(diagnosis.Window),
		AvailabilityTier: string(diagnosis.AvailabilityTier),
		Confidence:       diagnosis.Confidence,
		Evidences:        evidences,
		UpdatedAt:        formatTimestamp(diagnosis.UpdatedAt),
	}
}

func formatTimestamp(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339)
}

func calculateDownloadProgress(downloadedBytes, totalBytes int64) float64 {
	if totalBytes <= 0 {
		return 0
	}
	return float64(downloadedBytes) / float64(totalBytes)
}

func formatBytesPerSecond(bytesPerSecond int64) string {
	if bytesPerSecond <= 0 {
		return "0 B/s"
	}

	units := []string{"B/s", "KB/s", "MB/s", "GB/s", "TB/s"}
	value := float64(bytesPerSecond)
	unitIndex := 0
	for value >= 1024 && unitIndex < len(units)-1 {
		value /= 1024
		unitIndex++
	}

	if unitIndex == 0 {
		return fmt.Sprintf("%d %s", bytesPerSecond, units[unitIndex])
	}
	return fmt.Sprintf("%.2f %s", value, units[unitIndex])
}

func formatStreamState(session model.Session) string {
	switch session.StreamState {
	case model.SessionStreamStateStreaming:
		return "外部播放器正在拉流"
	case model.SessionStreamStateReady:
		if session.SelectedFileName != "" {
			return "可播放: " + session.SelectedFileName
		}
		return "已就绪"
	case model.SessionStreamStateUnavailable:
		return "等待 metadata 与主文件识别"
	case model.SessionStreamStateError:
		if strings.TrimSpace(session.LastError) != "" {
			return "流媒体错误: " + session.LastError
		}
		return "流媒体错误"
	default:
		return string(session.StreamState)
	}
}

func formatDeadState(session model.Session) string {
	switch session.HealthDiagnosis.AvailabilityTier {
	case model.SessionAvailabilityTierHighAvailable:
		return "高概率可用：已发现可用 peer，传输链路稳定"
	case model.SessionAvailabilityTierWeakAvailable:
		return "弱可用：存在可用信号，但仍需继续观察"
	case model.SessionAvailabilityTierNoResource:
		return "暂未发现资源：metadata/tracker/dht 尚未形成稳定发现链路"
	case model.SessionAvailabilityTierHighUnavailable:
		return "高概率不可用：多项负向证据持续成立"
	}

	switch session.DeadState {
	case model.SessionDeadStateAlive:
		return "已发现可用 peer，传输链路正常"
	case model.SessionDeadStateNoPeers:
		return "当前未发现可用 peer"
	case model.SessionDeadStateNoUsefulTransfer:
		return "peer 已连接，但最近没有 useful bytes 增量"
	case model.SessionDeadStateWaitingRange:
		return "播放器暂未继续发起 Range 请求"
	case model.SessionDeadStateMetadataStuck:
		return "metadata 获取耗时异常"
	case model.SessionDeadStateUnknown:
		return "等待 metadata/peer 探测"
	default:
		return string(session.DeadState)
	}
}
