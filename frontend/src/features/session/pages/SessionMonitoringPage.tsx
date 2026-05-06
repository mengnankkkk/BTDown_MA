import { SessionDetailPanel } from '../components/SessionDetailPanel'
import { SessionWorkspace } from '../components/SessionWorkspace'
import { useSessionDashboard } from '../hooks/useSessionDashboard'
import { StatusBanner } from '../../../shared/components/feedback/StatusBanner'

export function SessionMonitoringPage() {
  const {
    sessions,
    selectedSession,
    loading,
    errorMessage,
    selectSession
  } = useSessionDashboard()

  return (
    <section className="page-section">
      <div className="panel-header">
        <h2>详细监控</h2>
        <span>集中查看下载、流媒体、网络与慢因诊断的完整参数</span>
      </div>
      <StatusBanner message={errorMessage} />
      <div className="dashboard-grid">
        <SessionWorkspace
          loading={loading}
          sessions={sessions}
          selectedSession={selectedSession}
          onSelectSession={selectSession}
        />
        <SessionDetailPanel session={selectedSession} />
      </div>
    </section>
  )
}
