import { useEffect, useMemo, useRef, useState } from 'react'

import { getObservabilityOverview } from '../../../services/observability/observabilityApi'
import { cleanupSession, createSession, listSessions, pauseSession, resumeSession, stopSession } from '../../../services/session/sessionApi'
import type { ObservabilityOverview } from '../../../types/observabilityOverview'
import type { SessionCreatePayload } from '../../../types/sessionCreatePayload'
import type { SessionItem } from '../../../types/session'

export function useSessionDashboard() {
  const [sessions, setSessions] = useState<SessionItem[]>([])
  const [overview, setOverview] = useState<ObservabilityOverview>({
    sessionCount: 0,
    statusCounts: {},
    totalDownloadSpeedBytesPerSecond: 0,
    activePeersTotal: 0,
    averageFirstFrameLatencyMs: 0,
    averageSeekRecoveryMs: 0,
    averageBufferHitRatio: 0,
    averageRecoverySuccessRate: 0,
    recoveryLatencyDistribution: { lt1s: 0, '1to3s': 0, '3to8s': 0, gt8s: 0 },
    recentStreamAccesses: [],
    trend5m: []
  })
  const [loading, setLoading] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [errorMessage, setErrorMessage] = useState('')
  const [selectedSessionId, setSelectedSessionId] = useState('')
  const loadingRef = useRef(false)

  useEffect(() => {
    void loadDashboardData()
    const timer = window.setInterval(() => {
      void loadDashboardData({ silent: true })
    }, 5000)
    return () => {
      window.clearInterval(timer)
    }
  }, [])

  const selectedSession = useMemo(
    () => sessions.find((session) => session.id === selectedSessionId) ?? sessions[0],
    [selectedSessionId, sessions]
  )

  async function loadDashboardData(options?: { silent?: boolean }) {
    if (loadingRef.current) {
      return
    }

    loadingRef.current = true
    const silent = options?.silent ?? false

    if (!silent) {
      setLoading(true)
      setErrorMessage('')
    }

    try {
      const [sessionsResponse, overviewResponse] = await Promise.all([listSessions(), getObservabilityOverview()])
      const nextSessions = enrichSessions(sessionsResponse.data)
      setSessions(nextSessions)
      setOverview(overviewResponse.data)
      setSelectedSessionId((current) => {
        if (!nextSessions.length) {
          return ''
        }
        if (current && nextSessions.some((session) => session.id === current)) {
          return current
        }
        return nextSessions[0].id
      })
    } catch (error) {
      if (!silent) {
        setErrorMessage(error instanceof Error ? error.message : '加载会话失败')
      }
    } finally {
      loadingRef.current = false
      if (!silent) {
        setLoading(false)
      }
    }
  }

  async function submitSession(payload: SessionCreatePayload) {
    setSubmitting(true)
    setErrorMessage('')
    try {
      await createSession(payload)
      await loadDashboardData()
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : '创建会话失败')
    } finally {
      setSubmitting(false)
    }
  }

  async function stopCurrentSession() {
    if (!selectedSession?.id) {
      return
    }
    setErrorMessage('')
    try {
      await stopSession(selectedSession.id)
      await loadDashboardData()
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : '停止会话失败')
    }
  }

  async function pauseCurrentSession() {
    if (!selectedSession?.id) {
      return
    }
    setErrorMessage('')
    try {
      await pauseSession(selectedSession.id)
      await loadDashboardData()
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : '暂停会话失败')
    }
  }

  async function resumeCurrentSession() {
    if (!selectedSession?.id) {
      return
    }
    setErrorMessage('')
    try {
      await resumeSession(selectedSession.id)
      await loadDashboardData()
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : '恢复会话失败')
    }
  }

  async function cleanupCurrentSession() {
    if (!selectedSession?.id) {
      return
    }
    setErrorMessage('')
    try {
      await cleanupSession(selectedSession.id)
      await loadDashboardData()
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : '清理会话失败')
    }
  }

  function selectSession(session: SessionItem) {
    setSelectedSessionId(session.id)
  }

  return {
    sessions,
    overview,
    selectedSession,
    loading,
    submitting,
    errorMessage,
    loadDashboardData,
    submitSession,
    stopCurrentSession,
    pauseCurrentSession,
    resumeCurrentSession,
    cleanupCurrentSession,
    selectSession
  }
}

function enrichSessions(sessions: SessionItem[]) {
  return sessions.map(enrichSession)
}

function enrichSession(session: SessionItem): SessionItem {
  return {
    ...session,
    healthReport: session.healthReport ?? {
      summary: '等待诊断',
      reason: '后端尚未返回健康报告'
    },
    healthDiagnosis: session.healthDiagnosis ?? {
      window: 'FAST',
      availabilityTier: 'WEAK_AVAILABLE',
      confidence: 50,
      evidences: [],
      updatedAt: ''
    },
    metrics: session.metrics ?? {
      trackerCount: 0,
      originalTrackerCount: 0,
      appendedTrackerCount: 0,
      torrentPublicity: 'unknown',
      listenPort: 0,
      dhtStatus: 'unknown',
      dhtNodes: 0,
      udpReachable: 'unknown',
      incomingConnections: 0,
      activePeers: 0,
      totalPeers: 0,
      downloadSpeedBytesPerSecond: 0,
      usefulBytesDelta: 0,
      downloadedBytes: 0,
      totalBytes: 0,
      downloadProgress: 0,
      lastRangeResponseDurationMs: 0,
      firstFrameLatencyMs: 0,
      seekRecoveryMs: 0,
      bufferHitRatio: 0,
      windowRecoveryCount: 0,
      lastWindowRecoveryAt: '',
      lastWindowRecoveryReason: '',
      peerRecoveryCount: 0,
      lastPeerRecoveryAt: '',
      lastPeerRecoveryReason: '',
      recoveryLatencyBuckets: { lt1s: 0, '1to3s': 0, '3to8s': 0, gt8s: 0 },
      recoverySuccessRate: 0,
      downloadSpeedText: '0 B/s',
      streamStateText: '等待流媒体状态',
      deadTorrentStateText: '等待死种检测'
    }
  }
}
