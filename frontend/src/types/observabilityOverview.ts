export interface StreamAccessRecord {
  at: string
  sessionId: string
  method: string
  range: string
  status: number
  durationMs: number
  contentRange: string
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
}
