import type { SessionItem } from '../../../types/session'

interface SessionListItemProps {
  session: SessionItem
  active: boolean
  onSelect: (session: SessionItem) => void
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

export function SessionListItem({ session, active, onSelect }: SessionListItemProps) {
  const availabilityTier = formatAvailabilityTier(session.healthDiagnosis?.availabilityTier ?? '')

  return (
    <button
      type="button"
      className={active ? 'session-card active' : 'session-card'}
      onClick={() => onSelect(session)}
    >
      <div className="session-card-header">
        <strong>{session.name}</strong>
        <span>{session.status}</span>
      </div>
      <p className="session-field">ID：{session.id}</p>
      <p className="session-field">可用性：{availabilityTier}</p>
      <p className="session-field">置信度：{session.healthDiagnosis?.confidence ?? 0}%</p>
      <p className="session-field">速度：{session.metrics?.downloadSpeedText ?? '0 B/s'}</p>
      <p className="session-field">Peer：{session.metrics?.activePeers ?? 0} / {session.metrics?.totalPeers ?? 0}</p>
    </button>
  )
}
