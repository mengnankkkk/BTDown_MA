import { SessionCreateForm } from '../../session/components/SessionCreateForm'
import { SessionDetailPanel } from '../../session/components/SessionDetailPanel'
import { SessionWorkspace } from '../../session/components/SessionWorkspace'
import { useSessionDashboard } from '../../session/hooks/useSessionDashboard'
import { StatusBanner } from '../../../shared/components/feedback/StatusBanner'

export function DashboardOverviewPage() {
  const {
    sessions,
    overview,
    selectedSession,
    loading,
    submitting,
    errorMessage,
    submitSession,
    stopCurrentSession,
    pauseCurrentSession,
    resumeCurrentSession,
    cleanupCurrentSession,
    selectSession
  } = useSessionDashboard()

  return (
    <section className="page-section dashboard-overview">
      <StatusBanner message={errorMessage} />
      <section className="panel">
        <div className="panel-header">
          <h2>可观测总览</h2>
          <span>汇总会话状态、吞吐与最近流访问，便于测试播放器与调度行为</span>
        </div>
        <div className="detail-grid">
          <div className="detail-item">
            <strong>会话总数</strong>
            <p>{overview.sessionCount}</p>
          </div>
          <div className="detail-item">
            <strong>总下载速度</strong>
            <p>{overview.totalDownloadSpeedBytesPerSecond} B/s</p>
          </div>
          <div className="detail-item">
            <strong>总活跃 Peer</strong>
            <p>{overview.activePeersTotal}</p>
          </div>
          <div className="detail-item">
            <strong>状态分布</strong>
            <p>{Object.entries(overview.statusCounts).map(([key, value]) => `${key}:${value}`).join(' / ') || '暂无数据'}</p>
          </div>
          <div className="detail-item">
            <strong>平均第一帧时间</strong>
            <p>{overview.averageFirstFrameLatencyMs} ms</p>
          </div>
          <div className="detail-item">
            <strong>平均 Seek 恢复</strong>
            <p>{overview.averageSeekRecoveryMs} ms</p>
          </div>
          <div className="detail-item">
            <strong>平均缓冲命中率</strong>
            <p>{Math.round(overview.averageBufferHitRatio * 100)}%</p>
          </div>
        </div>
        <div className="panel-subsection">
          <strong>最近 5 分钟趋势</strong>
          {overview.trend5m.length === 0 ? (
            <p className="session-field">暂无趋势数据</p>
          ) : (
            overview.trend5m.map((point) => (
              <p key={point.at} className="session-field">
                {point.at} | 请求:{point.rangeRequestCount} | 平均耗时:{point.avgRangeDurationMs}ms | 平均 Seek 恢复:{point.avgSeekRecoveryMs}ms | 平均缓冲命中:{Math.round(point.avgBufferHitRatio * 100)}% | 活跃会话:{point.activeSessions}
              </p>
            ))
          )}
        </div>
      </section>
      <div className="dashboard-grid dashboard-grid-large">
        <SessionCreateForm submitting={submitting} onSubmit={submitSession} />
        <SessionWorkspace
          loading={loading}
          sessions={sessions}
          selectedSession={selectedSession}
          onSelectSession={selectSession}
        />
        <SessionDetailPanel
          session={selectedSession}
          onStopSession={stopCurrentSession}
          onPauseSession={pauseCurrentSession}
          onResumeSession={resumeCurrentSession}
          onCleanupSession={cleanupCurrentSession}
        />
      </div>
    </section>
  )
}
