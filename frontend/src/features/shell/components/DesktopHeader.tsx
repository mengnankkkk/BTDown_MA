interface DesktopHeaderProps {
  title: string
  subtitle: string
  runtimeLabel: string
}

export function DesktopHeader({ title, subtitle, runtimeLabel }: DesktopHeaderProps) {
  return (
    <header className="desktop-header">
      <div>
        <h2>{title}</h2>
        <p>{subtitle}</p>
      </div>
      <div className="runtime-badge">{runtimeLabel}</div>
    </header>
  )
}
