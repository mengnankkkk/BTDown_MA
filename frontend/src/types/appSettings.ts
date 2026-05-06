export interface AppSettings {
  playerPath: string
  torrentDataDir: string
  logDir: string
  autoCleanupEnabled: boolean
  autoCleanupPolicy: 'manual' | 'onSessionDelete'
  streamBaseUrl: string
  publicTrackers: string[]
  appendPublicTrackersForPublicTorrent: boolean
  btListenPort: number
  downloadRateLimitKiBps: number
  uploadRateLimitKiBps: number
  enablePortForwarding: boolean
  streamDynamicReadaheadEnabled: boolean
  streamReadaheadMinBytes: number
  streamReadaheadMaxBytes: number
  streamPreheatHeadPieces: number
  streamPreheatTailPieces: number
  streamSeekGapFactor: number
  streamBoostWindowPieces: number
  streamDeprioritizeOldWindow: boolean
}
