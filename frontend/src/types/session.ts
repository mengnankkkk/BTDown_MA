import type { SessionMetrics } from './sessionMetrics'

export interface SessionHealthReport {
  summary: string
  reason: string
}

export interface SessionHealthEvidence {
  type: string
  code: string
  severity: string
  detail: string
  firstSeenAt: string
  lastSeenAt: string
  count: number
}

export interface SessionHealthDiagnosis {
  window: string
  availabilityTier: string
  confidence: number
  evidences: SessionHealthEvidence[]
  updatedAt: string
}

export interface SessionItem {
  id: string
  name: string
  magnetUri: string
  status: string
  streamUrl: string
  metadataState: string
  downloadState: string
  streamState: string
  deadState: string
  healthReport: SessionHealthReport
  healthDiagnosis: SessionHealthDiagnosis
  selectedFileName?: string
  lastError?: string
  metrics?: SessionMetrics
}
