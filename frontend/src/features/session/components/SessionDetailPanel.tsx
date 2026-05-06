import type { SessionItem } from '../../../types/session'

interface SessionDetailPanelProps {
  session?: SessionItem
  onStopSession?: () => void
  onPauseSession?: () => void
  onResumeSession?: () => void
  onCleanupSession?: () => void
}

function formatAvailabilityTier(tier: string): string {
  switch (tier) {
    case 'HIGH_AVAILABLE':
      return '高概率可用'
    case 'WEAK_AVAILABLE':
      return '弱可用'
    case 'NO_RESOURCE':
      return '暂未发现资源'
    case 'HIGH_UNAVAILABLE':
      return '高概率不可用'
    default:
      return '弱可用'
  }
}

function formatBytes(value: number): string {
  const units = ['B', 'KiB', 'MiB', 'GiB']
  let size = Math.max(0, value)
  let unitIndex = 0

  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024
    unitIndex += 1
  }

  if (unitIndex === 0) {
    return `${Math.round(size)} ${units[unitIndex]}`
  }
  return `${size.toFixed(2)} ${units[unitIndex]}`
}

function formatProgress(value: number): string {
  return `${(Math.max(0, value) * 100).toFixed(1)}%`
}

function formatEta(remainingBytes: number, speedBytesPerSecond: number): string {
  if (remainingBytes <= 0) {
    return '已完成'
  }
  if (speedBytesPerSecond <= 0) {
    return '计算中'
  }

  const totalSeconds = Math.ceil(remainingBytes / speedBytesPerSecond)
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60

  if (hours > 0) {
    return `${hours}小时${minutes}分${seconds}秒`
  }
  if (minutes > 0) {
    return `${minutes}分${seconds}秒`
  }
  return `${seconds}秒`
}

export function SessionDetailPanel({ session, onStopSession, onPauseSession, onResumeSession, onCleanupSession }: SessionDetailPanelProps) {
  if (!session) {
    return (
      <section className="panel session-detail-panel">
        <div className="panel-header">
          <h2>会话详情</h2>
          <span>选择一个会话后，在这里查看后端重点指标</span>
        </div>
        <p className="empty-state">当前未选择会话</p>
      </section>
    )
  }

  const downloadProgress = session.metrics?.downloadProgress ?? 0
  const downloadedBytes = session.metrics?.downloadedBytes ?? 0
  const totalBytes = session.metrics?.totalBytes ?? 0
  const remainingBytes = Math.max(0, totalBytes - downloadedBytes)
  const downloadSpeedBytesPerSecond = session.metrics?.downloadSpeedBytesPerSecond ?? 0
  const etaText = formatEta(remainingBytes, downloadSpeedBytesPerSecond)

  return (
    <section className="panel session-detail-panel">
      <div className="panel-header">
        <h2>{session.name}</h2>
        <span>聚合查看下载、流媒体与慢因诊断，便于调参与实验</span>
      </div>
      <div className="detail-grid">
        <div className="detail-item">
          <strong>会话状态</strong>
          <p>{session.status}</p>
        </div>
        <div className="detail-item">
          <strong>可用性分档</strong>
          <p>{formatAvailabilityTier(session.healthDiagnosis?.availabilityTier ?? '')}</p>
        </div>
        <div className="detail-item">
          <strong>检测窗口 / 置信度</strong>
          <p>{session.healthDiagnosis?.window ?? 'FAST'} / {session.healthDiagnosis?.confidence ?? 0}%</p>
        </div>
        <div className="detail-item">
          <strong>健康状态</strong>
          <p>{session.healthReport.summary}</p>
        </div>
        <div className="detail-item">
          <strong>Metadata / 下载状态</strong>
          <p>{session.metadataState} / {session.downloadState}</p>
        </div>
        <div className="detail-item">
          <strong>下载速度</strong>
          <p>{session.metrics?.downloadSpeedText ?? '0 B/s'}</p>
        </div>
        <div className="detail-item">
          <strong>Peer 数</strong>
          <p>{session.metrics?.activePeers ?? 0} / {session.metrics?.totalPeers ?? 0}</p>
        </div>
        <div className="detail-item">
          <strong>Tracker 数</strong>
          <p>{session.metrics?.trackerCount ?? 0}</p>
        </div>
        <div className="detail-item">
          <strong>原始 / 追加 Tracker</strong>
          <p>{session.metrics?.originalTrackerCount ?? 0} / {session.metrics?.appendedTrackerCount ?? 0}</p>
        </div>
        <div className="detail-item">
          <strong>Torrent 属性</strong>
          <p>{session.metrics?.torrentPublicity ?? 'unknown'}</p>
        </div>
        <div className="detail-item">
          <strong>BT 监听端口</strong>
          <p>{session.metrics?.listenPort ?? 0}</p>
        </div>
        <div className="detail-item">
          <strong>DHT 状态 / 节点数</strong>
          <p>{session.metrics?.dhtStatus ?? 'unknown'} / {session.metrics?.dhtNodes ?? 0}</p>
        </div>
        <div className="detail-item">
          <strong>UDP 状态 / 入站连接</strong>
          <p>{session.metrics?.udpReachable ?? 'unknown'} / {session.metrics?.incomingConnections ?? 0}</p>
        </div>
        <div className="detail-item">
          <strong>最近 useful bytes 增量</strong>
          <p>{session.metrics?.usefulBytesDelta ?? 0}</p>
        </div>
        <div className="detail-item">
          <strong>最近 Range 响应耗时</strong>
          <p>{session.metrics?.lastRangeResponseDurationMs ?? 0} ms</p>
        </div>
        <div className="detail-item">
          <strong>第一帧时间</strong>
          <p>{session.metrics?.firstFrameLatencyMs ?? 0} ms</p>
        </div>
        <div className="detail-item">
          <strong>Seek 恢复时间</strong>
          <p>{session.metrics?.seekRecoveryMs ?? 0} ms</p>
        </div>
        <div className="detail-item">
          <strong>缓冲命中率</strong>
          <p>{Math.round((session.metrics?.bufferHitRatio ?? 0) * 100)}%</p>
        </div>
        <div className="detail-item">
          <strong>流媒体状态</strong>
          <p>{session.metrics?.streamStateText ?? session.streamState}</p>
        </div>
        <div className="detail-item">
          <strong>慢因判定</strong>
          <p>{session.metrics?.deadTorrentStateText ?? session.deadState}</p>
        </div>
        <div className="detail-item">
          <strong>下载进度</strong>
          <p>{formatProgress(downloadProgress)}</p>
        </div>
        <div className="detail-item">
          <strong>已下载 / 总大小</strong>
          <p>{formatBytes(downloadedBytes)} / {formatBytes(totalBytes)}</p>
        </div>
        <div className="detail-item">
          <strong>预计剩余时间</strong>
          <p>{etaText}</p>
        </div>
      </div>
      <div className="panel-subsection">
        <strong>健康说明</strong>
        <p className="session-field">{session.healthReport.reason}</p>
      </div>
      <div className="panel-subsection">
        <strong>诊断证据</strong>
        {session.healthDiagnosis?.evidences?.length ? (
          <ul className="session-field">
            {session.healthDiagnosis.evidences.slice(0, 6).map((evidence, index) => (
              <li key={`${evidence.code}-${index}`}>
                [{evidence.severity}] {evidence.code}：{evidence.detail}
              </li>
            ))}
          </ul>
        ) : (
          <p className="session-field">暂无诊断证据</p>
        )}
      </div>
      <div className="panel-subsection">
        <strong>主文件</strong>
        <p className="session-field">{session.selectedFileName || '等待主文件识别'}</p>
      </div>
      <div className="panel-subsection">
        <strong>最近错误</strong>
        <p className="session-field">{session.lastError || '暂无错误'}</p>
      </div>
      <div className="panel-subsection">
        <strong>Magnet</strong>
        <p className="session-field">{session.magnetUri}</p>
      </div>
      <div className="panel-subsection">
        <strong>Stream URL</strong>
        <p className="session-field">{session.streamUrl}</p>
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
