import type { ChangeEvent } from 'react'

import type { DesktopSettingItem } from '../../../types/desktopSettingItem'

interface SettingsFormCardProps {
  title: string
  item: DesktopSettingItem
  onChange?: (event: ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>) => void
}

export function SettingsFormCard({ title, item, onChange }: SettingsFormCardProps) {
  return (
    <article className="setting-card">
      <strong>{title}</strong>
      <label className="settings-field">
        <span>{item.label}</span>
        {item.type === 'select' ? (
          <select value={item.value} onChange={onChange}>
            {item.options?.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        ) : item.type === 'textarea' ? (
          <textarea value={item.value} onChange={onChange} placeholder={item.label} rows={6} />
        ) : (
          <input value={item.value} onChange={onChange} placeholder={item.label} />
        )}
      </label>
      <span>{item.hint}</span>
    </article>
  )
}
