import { SessionCreateForm } from '../../features/session/components/SessionCreateForm'
import { SessionList } from '../../features/session/components/SessionList'
import { useSessionDashboard } from '../../features/session/hooks/useSessionDashboard'
import { StatusBanner } from '../../shared/components/feedback/StatusBanner'

export function DashboardPage() {
  const { sessions, loading, submitting, errorMessage, submitSession } = useSessionDashboard()

  return (
    <main className="dashboard-page">
      <header className="dashboard-header">
        <div>
          <h1>BTDown_MA</h1>
          <p>Wails + React + Go 的边下边播桌面应用骨架</p>
        </div>
      </header>
      <StatusBanner message={errorMessage} />
      <section className="dashboard-grid">
        <SessionCreateForm submitting={submitting} onSubmit={submitSession} />
        <SessionList loading={loading} sessions={sessions} />
      </section>
    </main>
  )
}
