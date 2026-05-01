import { useMemo, useState } from 'react'

import type { AppView } from '../../types/appView'
import { getRuntimeEnvironment } from '../../shared/runtime/runtimeEnvironment'
import { DashboardOverviewPage } from '../../features/dashboard/pages/DashboardOverviewPage'
import { SessionsPage } from '../../features/session/pages/SessionsPage'
import { SettingsPage } from '../../features/settings/pages/SettingsPage'
import { DesktopHeader } from '../../features/shell/components/DesktopHeader'
import { SideNavigation } from '../../features/shell/components/SideNavigation'
import { DesktopStatusBar } from '../../features/shell/components/DesktopStatusBar'

export function DesktopHomePage() {
  const [activeView, setActiveView] = useState<AppView>('dashboard')
  const runtimeEnvironment = useMemo(() => getRuntimeEnvironment(), [])
  const currentView = resolveCurrentView(activeView)

  return (
    <main className="desktop-shell">
      <SideNavigation activeView={activeView} onChange={setActiveView} />
      <section className="desktop-content">
        <DesktopHeader
          title={currentView.title}
          subtitle={currentView.subtitle}
          runtimeLabel={runtimeEnvironment.platformName}
        />
        <div className="desktop-workspace">{currentView.content}</div>
        <DesktopStatusBar
          runtimeLabel={runtimeEnvironment.platformName}
          sessionCount={currentView.sessionCount}
          selectedSessionName={currentView.selectedSessionName}
        />
      </section>
    </main>
  )
}

function resolveCurrentView(activeView: AppView) {
  switch (activeView) {
    case 'sessions':
      return {
        title: '会话管理',
        subtitle: '聚焦当前会话状态与播放入口',
        content: <SessionsPage />,
        sessionCount: 0,
        selectedSessionName: ''
      }
    case 'settings':
      return {
        title: '桌面设置',
        subtitle: '维护播放器、目录和运行时配置',
        content: <SettingsPage />,
        sessionCount: 0,
        selectedSessionName: ''
      }
    case 'dashboard':
    default:
      return {
        title: '应用总览',
        subtitle: '先保留桌面应用主工作区，再逐步填充能力',
        content: <DashboardOverviewPage />,
        sessionCount: 0,
        selectedSessionName: ''
      }
  }
}
