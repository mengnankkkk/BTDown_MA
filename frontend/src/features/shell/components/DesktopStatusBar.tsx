interface DesktopStatusBarProps {
  sessionCount: number
  runtimeLabel: string
  selectedSessionName?: string
}

export function DesktopStatusBar({ sessionCount, runtimeLabel, selectedSessionName }: DesktopStatusBarProps) {
  return (
    <footer className="desktop-status-bar">
      <span>运行环境：{runtimeLabel}</span>
      <span>会话数量：{sessionCount}</span>
      <span>当前焦点：{selectedSessionName ?? '未选择'}</span>
    </footer>
  )
}
