import type { AppView } from '../../../types/appView'

export interface NavigationItem {
  key: AppView
  title: string
  description: string
}

export const navigationItems: NavigationItem[] = [
  {
    key: 'dashboard',
    title: '总览',
    description: '查看当前应用状态和会话入口'
  },
  {
    key: 'sessions',
    title: '会话',
    description: '查看实时进度、速度和预估时间'
  },
  {
    key: 'monitoring',
    title: '详细监控',
    description: '查看完整下载与诊断参数'
  },
  {
    key: 'settings',
    title: '设置',
    description: '维护播放器和目录配置'
  }
]
