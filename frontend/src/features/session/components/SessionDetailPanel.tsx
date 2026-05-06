import type { SessionItem } from '../../../types/session'
import { formatAvailabilityTier, formatBytes, formatProgress, getSessionDownloadSummary } from './sessionMetricFormatters'

interface SessionDetailPanelProps {
  session?: SessionItem
}

export function SessionDetailPanel({ session }: SessionDetailPanelProps) {
  if (!session) {
    return (
      <section className="panel session-detail-panel">
        <div className="panel-header">
          <h2>详细监控</h2>
          <span>选择一个会话后，在这里查看完整诊断参数</span>
        </div>
        <p className="empty-state">当前未选择会话</p>
      </section>
    )
  }

  const { downloadProgress, downloadedBytes, totalBytes, etaText } = getSessionDownloadSummary(session)

  return (
    <section className="panel session-detail-panel">
      <div className="panel-header">
        <h2>{session.name}</h2>
        <span>完整下载、流媒体、网络与慢因诊断参数</span>
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
          <strong>关键块补拉次数</strong>
          <p>{session.metrics?.windowRecoveryCount ?? 0}</p>
        </div>
        <div className="detail-item">
          <strong>最近补拉时间</strong>
          <p>{session.metrics?.lastWindowRecoveryAt || '暂无'}</p>
        </div>
        <div className="detail-item">
          <strong>最近补拉原因</strong>
          <p>{session.metrics?.lastWindowRecoveryReason || '暂无'}</p>
        </div>
        <div className="detail-item">
          <strong>慢 Peer 补救次数</strong>
          <p>{session.metrics?.peerRecoveryCount ?? 0}</p>
        </div>
        <div className="detail-item">
          <strong>最近慢 Peer 补救时间</strong>
          <p>{session.metrics?.lastPeerRecoveryAt || '暂无'}</p>
        </div>
        <div className="detail-item">
          <strong>最近慢 Peer 补救原因</strong>
          <p>{session.metrics?.lastPeerRecoveryReason || '暂无'}</p>
        </div>
        <div className="detail-item">
          <strong>补拉恢复成功率</strong>
          <p>{Math.round((session.metrics?.recoverySuccessRate ?? 0) * 100)}%</p>
        </div>
        <div className="detail-item">
          <strong>恢复耗时分布</strong>
          <p>
            {'<'}1s:{session.metrics?.recoveryLatencyBuckets?.lt1s ?? 0} / 1-3s:{session.metrics?.recoveryLatencyBuckets?.['1to3s'] ?? 0} / 3-8s:{session.metrics?.recoveryLatencyBuckets?.['3to8s'] ?? 0} / {'>'}8s:{session.metrics?.recoveryLatencyBuckets?.gt8s ?? 0}
          </p>
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
    </section>
  )
}
