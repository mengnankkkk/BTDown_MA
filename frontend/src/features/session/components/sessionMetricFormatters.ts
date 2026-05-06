import type { SessionItem } from '../../../types/session'

export function formatAvailabilityTier(tier: string): string {
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

export function formatBytes(value: number): string {
  const units = ['B', 'KiB', 'MiB', 'GiB']
  let size = Math.max(0, value)
  let unitIndex = 0

  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024
    unitIndex += 1
  }

  if (unitIndex === 0) {
    return `${Math.round(size)} ${units[unitIndex]}`
  }
  return `${size.toFixed(2)} ${units[unitIndex]}`
}

export function formatProgress(value: number): string {
  return `${(Math.max(0, value) * 100).toFixed(1)}%`
}

export function formatEta(remainingBytes: number, speedBytesPerSecond: number): string {
  if (remainingBytes <= 0) {
    return '已完成'
  }
  if (speedBytesPerSecond <= 0) {
    return '计算中'
  }

  const totalSeconds = Math.ceil(remainingBytes / speedBytesPerSecond)
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60

  if (hours > 0) {
    return `${hours}小时${minutes}分${seconds}秒`
  }
  if (minutes > 0) {
    return `${minutes}分${seconds}秒`
  }
  return `${seconds}秒`
}

export function getSessionDownloadSummary(session: SessionItem) {
  const downloadProgress = session.metrics?.downloadProgress ?? 0
  const downloadedBytes = session.metrics?.downloadedBytes ?? 0
  const totalBytes = session.metrics?.totalBytes ?? 0
  const remainingBytes = Math.max(0, totalBytes - downloadedBytes)
  const downloadSpeedBytesPerSecond = session.metrics?.downloadSpeedBytesPerSecond ?? 0

  return {
    downloadProgress,
    downloadedBytes,
    totalBytes,
    etaText: formatEta(remainingBytes, downloadSpeedBytesPerSecond)
  }
}
