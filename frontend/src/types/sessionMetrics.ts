export interface SessionMetrics {
  trackerCount: number
  originalTrackerCount: number
  appendedTrackerCount: number
  torrentPublicity: string
  listenPort: number
  dhtStatus: string
  dhtNodes: number
  udpReachable: string
  incomingConnections: number
  activePeers: number
  totalPeers: number
  downloadSpeedBytesPerSecond: number
  usefulBytesDelta: number
  downloadedBytes: number
  totalBytes: number
  downloadProgress: number
  lastRangeResponseDurationMs: number
  firstFrameLatencyMs: number
  seekRecoveryMs: number
  bufferHitRatio: number
  windowRecoveryCount: number
  lastWindowRecoveryAt: string
  lastWindowRecoveryReason: string
  peerRecoveryCount: number
  lastPeerRecoveryAt: string
  lastPeerRecoveryReason: string
  recoveryLatencyBuckets: Record<string, number>
  recoverySuccessRate: number
  downloadSpeedText: string
  streamStateText: string
  deadTorrentStateText: string
}
