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
}
