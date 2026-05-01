import { SessionDetailPanel } from '../components/SessionDetailPanel'
import { SessionWorkspace } from '../components/SessionWorkspace'
import { useSessionDashboard } from '../hooks/useSessionDashboard'
import { StatusBanner } from '../../../shared/components/feedback/StatusBanner'

export function SessionsPage() {
  const { sessions, selectedSession, loading, errorMessage, stopCurrentSession, cleanupCurrentSession, selectSession } = useSessionDashboard()

  return (
    <section className="page-section">
      <div className="panel-header">
        <h2>会话管理</h2>
        <span>集中查看后端会话状态、速度与流媒体核心指标</span>
      </div>
      <StatusBanner message={errorMessage} />
      <div className="dashboard-grid dashboard-grid-large">
        <SessionWorkspace
          loading={loading}
          sessions={sessions}
          selectedSession={selectedSession}
          onSelectSession={selectSession}
        />
        <SessionDetailPanel
          session={selectedSession}
          onStopSession={stopCurrentSession}
          onCleanupSession={cleanupCurrentSession}
        />
      </div>
    </section>
  )
}
