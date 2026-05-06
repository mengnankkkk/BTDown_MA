export interface StreamAccessRecord {
  at: string
  sessionId: string
  method: string
  range: string
  status: number
  durationMs: number
  contentRange: string
}

export interface TrendPoint {
  at: string
  rangeRequestCount: number
  avgRangeDurationMs: number
  avgSeekRecoveryMs: number
  avgBufferHitRatio: number
  activeSessions: number
}

export interface ObservabilityOverview {
  sessionCount: number
  statusCounts: Record<string, number>
  totalDownloadSpeedBytesPerSecond: number
  activePeersTotal: number
  averageFirstFrameLatencyMs: number
  averageSeekRecoveryMs: number
  averageBufferHitRatio: number
  recentStreamAccesses: StreamAccessRecord[]
  trend5m: TrendPoint[]
}
