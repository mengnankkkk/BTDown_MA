import type { SessionCreatePayload } from '../../types/sessionCreatePayload'
import type { SessionItem } from '../../types/session'
import type { AppSettings } from '../../types/appSettings'
import type { ObservabilityOverview } from '../../types/observabilityOverview'

interface WailsSessionBridge {
  CreateSession?: (payload: SessionCreatePayload) => Promise<SessionItem>
  ListSessions?: () => Promise<SessionItem[]>
  GetSettings?: () => Promise<AppSettings>
  UpdateSettings?: (payload: AppSettings) => Promise<AppSettings>
  GetObservabilityOverview?: () => Promise<ObservabilityOverview>
  StopSession?: (sessionId: string) => Promise<void>
  PauseSession?: (sessionId: string) => Promise<void>
  ResumeSession?: (sessionId: string) => Promise<void>
  CleanupSession?: (sessionId: string) => Promise<void>
}

declare global {
  interface Window {
    backendBridge?: WailsSessionBridge
  }
}

export function getWailsSessionBridge() {
  return window.backendBridge
}
