export interface DesktopSettingOption {
  label: string
  value: string
}

export interface DesktopSettingItem {
  label: string
  value: string
  hint: string
  type?: 'text' | 'select' | 'textarea'
  options?: DesktopSettingOption[]
}
