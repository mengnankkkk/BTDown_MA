interface StatusBannerProps {
  message: string
}

export function StatusBanner({ message }: StatusBannerProps) {
  if (!message) {
    return null
  }

  return <div className="status-banner">{message}</div>
}
