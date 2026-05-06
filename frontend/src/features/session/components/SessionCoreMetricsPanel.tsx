import type { SessionItem } from '../../../types/session'
import { formatBytes, formatProgress, getSessionDownloadSummary } from './sessionMetricFormatters'

interface SessionCoreMetricsPanelProps {
  session?: SessionItem
  onStopSession?: () => void
  onPauseSession?: () => void
  onResumeSession?: () => void
  onCleanupSession?: () => void
}

export function SessionCoreMetricsPanel({
  session,
  onStopSession,
  onPauseSession,
  onResumeSession,
  onCleanupSession
}: SessionCoreMetricsPanelProps) {
  if (!session) {
    return (
      <section className="panel session-detail-panel">
        <div className="panel-header">
          <h2>实时进度</h2>
          <span>选择一个会话后，在这里查看核心下载状态</span>
        </div>
        <p className="empty-state">当前未选择会话</p>
      </section>
    )
  }

  const { downloadProgress, downloadedBytes, totalBytes, etaText } = getSessionDownloadSummary(session)
  const progressPercent = Math.min(100, Math.max(0, downloadProgress * 100))

  return (
    <section className="panel session-detail-panel">
      <div className="panel-header">
        <h2>{session.name}</h2>
        <span>实时下载进度、速度与预计完成时间</span>
      </div>
      <div className="session-progress-card">
        <div className="session-progress-header">
          <strong>实时进度</strong>
          <span>{formatProgress(downloadProgress)}</span>
        </div>
        <div className="session-progress-track" aria-label="实时下载进度">
          <div className="session-progress-fill" style={{ width: `${progressPercent}%` }} />
        </div>
        <p className="session-field">{formatBytes(downloadedBytes)} / {formatBytes(totalBytes)}</p>
      </div>
      <div className="detail-grid session-core-grid">
        <div className="detail-item">
          <strong>实时下载速度</strong>
          <p>{session.metrics?.downloadSpeedText ?? '0 B/s'}</p>
        </div>
        <div className="detail-item">
          <strong>预估下载时间</strong>
          <p>{etaText}</p>
        </div>
        <div className="detail-item">
          <strong>会话状态</strong>
          <p>{session.status}</p>
        </div>
      </div>
      <div className="panel-subsection session-actions">
        <button type="button" className="session-action-button" onClick={onStopSession}>
          停止会话
        </button>
        <button
          type="button"
          className="session-action-button"
          onClick={onPauseSession}
          disabled={session.downloadState !== 'DOWNLOADING'}
        >
          暂停下载
        </button>
        <button
          type="button"
          className="session-action-button"
          onClick={onResumeSession}
          disabled={session.downloadState !== 'PAUSED'}
        >
          继续下载
        </button>
        <button type="button" className="session-action-button session-action-button-danger" onClick={onCleanupSession}>
          清理会话
        </button>
      </div>
    </section>
  )
}
