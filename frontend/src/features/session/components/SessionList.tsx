import type { SessionItem } from '../../../types/session'

interface SessionListProps {
  loading: boolean
  sessions: SessionItem[]
}

export function SessionList({ loading, sessions }: SessionListProps) {
  return (
    <section className="panel session-list">
      <div className="panel-header">
        <h2>会话列表</h2>
        <span>{loading ? '正在加载...' : `共 ${sessions.length} 个会话`}</span>
      </div>
      <div className="session-list-body">
        {sessions.length === 0 && !loading ? <p className="empty-state">当前还没有会话</p> : null}
        {sessions.map((session) => (
          <article key={session.id} className="session-card">
            <div className="session-card-header">
              <strong>{session.name}</strong>
              <span>{session.status}</span>
            </div>
            <p className="session-field">ID：{session.id}</p>
            <p className="session-field">Magnet：{session.magnetUri}</p>
            <p className="session-field">Stream：{session.streamUrl}</p>
          </article>
        ))}
      </div>
    </section>
  )
}
