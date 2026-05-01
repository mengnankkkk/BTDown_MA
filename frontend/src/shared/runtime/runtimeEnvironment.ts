import type { RuntimeEnvironment } from '../../types/runtimeEnvironment'

declare global {
  interface Window {
    go?: unknown
    runtime?: unknown
  }
}

export function getRuntimeEnvironment(): RuntimeEnvironment {
  return {
    isWailsRuntime: Boolean(window.go || window.runtime),
    platformName: window.go || window.runtime ? 'Wails Desktop Runtime' : 'Browser Preview Mode'
  }
}
