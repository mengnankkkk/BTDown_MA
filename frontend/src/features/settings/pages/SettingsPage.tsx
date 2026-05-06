import { useEffect, useMemo, useState } from 'react'

import { SettingsFormCard } from '../components/SettingsFormCard'
import { getSettings, updateSettings } from '../../../services/settings/settingsApi'
import { StatusBanner } from '../../../shared/components/feedback/StatusBanner'
import type { AppSettings } from '../../../types/appSettings'

const defaultSettings: AppSettings = {
  playerPath: '',
  torrentDataDir: '',
  logDir: '',
  autoCleanupEnabled: false,
  autoCleanupPolicy: 'manual',
  streamBaseUrl: '',
  publicTrackers: [],
  appendPublicTrackersForPublicTorrent: true,
  btListenPort: 51413,
  downloadRateLimitKiBps: 0,
  uploadRateLimitKiBps: 128,
  enablePortForwarding: true,
  streamDynamicReadaheadEnabled: true,
  streamReadaheadMinBytes: 2 << 20,
  streamReadaheadMaxBytes: 16 << 20,
  streamPreheatHeadPieces: 8,
  streamPreheatTailPieces: 8,
  streamSeekGapFactor: 1,
  streamBoostWindowPieces: 12,
  streamDeprioritizeOldWindow: true
}

export function SettingsPage() {
  const [settings, setSettings] = useState<AppSettings>(defaultSettings)
  const [saving, setSaving] = useState(false)
  const [errorMessage, setErrorMessage] = useState('')
  const [successMessage, setSuccessMessage] = useState('')

  useEffect(() => {
    void loadSettings()
  }, [])

  const cards = useMemo(
    () => [
      {
        title: '播放器设置',
        key: 'playerPath',
        item: {
          label: '播放器路径',
          value: settings.playerPath,
          hint: '填写播放器可执行文件路径'
        }
      },
      {
        title: '下载目录设置',
        key: 'torrentDataDir',
        item: {
          label: '下载目录',
          value: settings.torrentDataDir,
          hint: '用于存放 torrent 下载数据'
        }
      },
      {
        title: '日志目录设置',
        key: 'logDir',
        item: {
          label: '日志目录',
          value: settings.logDir,
          hint: '用于导出诊断与实验日志'
        }
      },
      {
        title: '自动清理开关',
        key: 'autoCleanupEnabled',
        item: {
          label: '自动清理',
          value: settings.autoCleanupEnabled ? 'enabled' : 'disabled',
          hint: '控制是否在删除会话时自动执行清理策略',
          type: 'select' as const,
          options: [
            { label: '关闭', value: 'disabled' },
            { label: '开启', value: 'enabled' }
          ]
        }
      },
      {
        title: '自动清理策略',
        key: 'autoCleanupPolicy',
        item: {
          label: '清理策略',
          value: settings.autoCleanupPolicy,
          hint: 'manual 为仅手动清理，onSessionDelete 为删除会话时自动清理',
          type: 'select' as const,
          options: [
            { label: '仅手动清理', value: 'manual' },
            { label: '删除会话时自动清理', value: 'onSessionDelete' }
          ]
        }
      },
      {
        title: 'BT 公共 Tracker 列表',
        key: 'publicTrackers',
        item: {
          label: '公共 Tracker',
          value: settings.publicTrackers.join('\n'),
          hint: '每行一个 tracker，仅对公开 torrent 的 peer 发现增强生效',
          type: 'textarea' as const
        }
      },
      {
        title: '公开 Torrent Tracker 追加',
        key: 'appendPublicTrackersForPublicTorrent',
        item: {
          label: '自动追加公共 Tracker',
          value: settings.appendPublicTrackersForPublicTorrent ? 'enabled' : 'disabled',
          hint: '仅对公开 magnet / 非私有 torrent 追加，帮助 tracker 较少的资源发现更多 peer',
          type: 'select' as const,
          options: [
            { label: '关闭', value: 'disabled' },
            { label: '开启', value: 'enabled' }
          ]
        }
      },
      {
        title: 'BT 监听端口',
        key: 'btListenPort',
        item: {
          label: '监听端口',
          value: String(settings.btListenPort),
          hint: '建议固定端口并在系统防火墙放行 TCP/UDP，同步提升入站连接能力'
        }
      },
      {
        title: '下载限速',
        key: 'downloadRateLimitKiBps',
        item: {
          label: '下载限速 KiB/s',
          value: String(settings.downloadRateLimitKiBps),
          hint: '0 表示不限速，通常保持不限速更利于实验吞吐'
        }
      },
      {
        title: '上传限速',
        key: 'uploadRateLimitKiBps',
        item: {
          label: '上传限速 KiB/s',
          value: String(settings.uploadRateLimitKiBps),
          hint: '不要设为 0，保留最小上传能力有利于互惠和持续下载'
        }
      },
      {
        title: '端口映射',
        key: 'enablePortForwarding',
        item: {
          label: '端口映射',
          value: settings.enablePortForwarding ? 'enabled' : 'disabled',
          hint: '开启后可尝试通过 NAT-PMP / UPnP 提升入站连接机会',
          type: 'select' as const,
          options: [
            { label: '关闭', value: 'disabled' },
            { label: '开启', value: 'enabled' }
          ]
        }
      },
      {
        title: '动态 Readahead',
        key: 'streamDynamicReadaheadEnabled',
        item: {
          label: '动态 Readahead',
          value: settings.streamDynamicReadaheadEnabled ? 'enabled' : 'disabled',
          hint: '根据 Range 响应耗时自动调节预读窗口',
          type: 'select' as const,
          options: [
            { label: '关闭', value: 'disabled' },
            { label: '开启', value: 'enabled' }
          ]
        }
      },
      {
        title: '最小 Readahead 字节数',
        key: 'streamReadaheadMinBytes',
        item: {
          label: '最小 Readahead Bytes',
          value: String(settings.streamReadaheadMinBytes),
          hint: '动态预读下限（字节）'
        }
      },
      {
        title: '最大 Readahead 字节数',
        key: 'streamReadaheadMaxBytes',
        item: {
          label: '最大 Readahead Bytes',
          value: String(settings.streamReadaheadMaxBytes),
          hint: '动态预读上限（字节）'
        }
      },
      {
        title: '首段预热 Piece 数',
        key: 'streamPreheatHeadPieces',
        item: {
          label: '首段预热 Piece',
          value: String(settings.streamPreheatHeadPieces),
          hint: 'metadata 就绪后优先预热文件开头的 piece 数'
        }
      },
      {
        title: '尾段预热 Piece 数',
        key: 'streamPreheatTailPieces',
        item: {
          label: '尾段预热 Piece',
          value: String(settings.streamPreheatTailPieces),
          hint: 'metadata 就绪后优先预热文件尾部的 piece 数'
        }
      },
      {
        title: 'Seek 判定系数',
        key: 'streamSeekGapFactor',
        item: {
          label: 'Seek Gap Factor',
          value: String(settings.streamSeekGapFactor),
          hint: 'gap 超过当前 readahead * 系数时判定为 seek'
        }
      },
      {
        title: 'Seek 提权窗口 Piece 数',
        key: 'streamBoostWindowPieces',
        item: {
          label: 'Boost Window Pieces',
          value: String(settings.streamBoostWindowPieces),
          hint: 'seek 后围绕目标区域提权下载的窗口大小'
        }
      },
      {
        title: '旧窗口降级',
        key: 'streamDeprioritizeOldWindow',
        item: {
          label: '旧窗口降级',
          value: settings.streamDeprioritizeOldWindow ? 'enabled' : 'disabled',
          hint: 'seek 到新位置后是否将旧提权窗口降级',
          type: 'select' as const,
          options: [
            { label: '关闭', value: 'disabled' },
            { label: '开启', value: 'enabled' }
          ]
        }
      },
    ],
    [settings]
  )

  async function loadSettings() {
    setErrorMessage('')
    try {
      const response = await getSettings()
      setSettings(response.data)
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : '加载设置失败')
    }
  }

  async function saveSettings() {
    setSaving(true)
    setErrorMessage('')
    setSuccessMessage('')
    try {
      const response = await updateSettings(settings)
      setSettings(response.data)
      setSuccessMessage('设置已保存')
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : '保存设置失败')
    } finally {
      setSaving(false)
    }
  }

  function handleCardChange(key: string, value: string) {
    setSettings((current) => {
      if (key === 'autoCleanupEnabled') {
        return {
          ...current,
          autoCleanupEnabled: value === 'enabled'
        }
      }
      if (key === 'autoCleanupPolicy') {
        return {
          ...current,
          autoCleanupPolicy: value as AppSettings['autoCleanupPolicy']
        }
      }
      if (key === 'appendPublicTrackersForPublicTorrent') {
        return {
          ...current,
          appendPublicTrackersForPublicTorrent: value === 'enabled'
        }
      }
      if (key === 'enablePortForwarding') {
        return {
          ...current,
          enablePortForwarding: value === 'enabled'
        }
      }
      if (key === 'streamDynamicReadaheadEnabled') {
        return {
          ...current,
          streamDynamicReadaheadEnabled: value === 'enabled'
        }
      }
      if (key === 'streamDeprioritizeOldWindow') {
        return {
          ...current,
          streamDeprioritizeOldWindow: value === 'enabled'
        }
      }
      if (key === 'publicTrackers') {
        return {
          ...current,
          publicTrackers: value
            .split('\n')
            .map((item) => item.trim())
            .filter(Boolean)
        }
      }
      if (key === 'streamSeekGapFactor') {
        const floatValue = Number.parseFloat(value)
        return {
          ...current,
          streamSeekGapFactor: Number.isNaN(floatValue) ? 1 : floatValue
        }
      }
      if (
        key === 'btListenPort' ||
        key === 'downloadRateLimitKiBps' ||
        key === 'uploadRateLimitKiBps' ||
        key === 'streamReadaheadMinBytes' ||
        key === 'streamReadaheadMaxBytes' ||
        key === 'streamPreheatHeadPieces' ||
        key === 'streamPreheatTailPieces' ||
        key === 'streamBoostWindowPieces'
      ) {
        const numericValue = Number.parseInt(value, 10)
        return {
          ...current,
          [key]: Number.isNaN(numericValue) ? 0 : numericValue
        }
      }
      return {
        ...current,
        [key]: value
      }
    })
  }

  return (
    <section className="page-section">
      <div className="panel-header">
        <h2>桌面设置</h2>
        <span>维护播放器、目录与清理策略，便于实验切换与回放</span>
      </div>
      <StatusBanner message={errorMessage || successMessage} />
      <div className="settings-list">
        {cards.map((card) => (
          <SettingsFormCard
            key={card.key}
            title={card.title}
            item={card.item}
            onChange={(event) => handleCardChange(card.key, event.target.value)}
          />
        ))}
      </div>
      <button type="button" className="session-action-button" onClick={saveSettings} disabled={saving}>
        {saving ? '保存中...' : '保存设置'}
      </button>
    </section>
  )
}
