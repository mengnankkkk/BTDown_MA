import { getApiBaseUrl } from '../../shared/config/apiConfig'
import { requestJSON } from '../../shared/http/requestJSON'
import { getWailsSessionBridge } from '../../shared/runtime/wailsSessionBridge'
import type { ApiResponse } from '../../types/apiResponse'
import type { SessionCreatePayload } from '../../types/sessionCreatePayload'
import type { SessionItem } from '../../types/session'

export async function listSessions() {
  const bridge = getWailsSessionBridge()
  if (bridge?.ListSessions) {
    const sessions = await bridge.ListSessions()
    return {
      code: 0,
      message: 'success',
      data: sessions
    } satisfies ApiResponse<SessionItem[]>
  }

  return requestJSON<ApiResponse<SessionItem[]>>(`${getApiBaseUrl()}/api/v1/sessions`)
}

export async function createSession(payload: SessionCreatePayload) {
  const bridge = getWailsSessionBridge()
  if (bridge?.CreateSession) {
    const session = await bridge.CreateSession(payload)
    return {
      code: 0,
      message: 'success',
      data: session
    } satisfies ApiResponse<SessionItem>
  }

  return requestJSON<ApiResponse<SessionItem>>(`${getApiBaseUrl()}/api/v1/sessions`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(payload)
  })
}

export async function stopSession(sessionId: string) {
  const bridge = getWailsSessionBridge()
  if (bridge?.StopSession) {
    await bridge.StopSession(sessionId)
    return
  }

  await requestJSON<ApiResponse<{ status: string }>>(`${getApiBaseUrl()}/api/v1/sessions/${sessionId}/stop`, {
    method: 'POST'
  })
}

export async function cleanupSession(sessionId: string) {
  const bridge = getWailsSessionBridge()
  if (bridge?.CleanupSession) {
    await bridge.CleanupSession(sessionId)
    return
  }

  await requestJSON<ApiResponse<{ status: string }>>(`${getApiBaseUrl()}/api/v1/sessions/${sessionId}`, {
    method: 'DELETE'
  })
}
