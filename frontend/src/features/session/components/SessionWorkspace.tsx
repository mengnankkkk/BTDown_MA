import type { SessionItem } from '../../../types/session'
import { SessionListItem } from './SessionListItem'

interface SessionWorkspaceProps {
  loading: boolean
  sessions: SessionItem[]
  selectedSession?: SessionItem
  onSelectSession: (session: SessionItem) => void
}

export function SessionWorkspace({
  loading,
  sessions,
  selectedSession,
  onSelectSession
}: SessionWorkspaceProps) {
  return (
    <section className="panel session-list">
      <div className="panel-header">
        <h2>会话列表</h2>
        <span>{loading ? '正在加载...' : `共 ${sessions.length} 个会话`}</span>
      </div>
      <div className="session-list-body">
        {sessions.length === 0 && !loading ? <p className="empty-state">当前还没有会话</p> : null}
        {sessions.map((session) => (
          <SessionListItem
            key={session.id}
            session={session}
            active={selectedSession?.id === session.id}
            onSelect={onSelectSession}
          />
        ))}
      </div>
    </section>
  )
}
