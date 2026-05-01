import { getApiBaseUrl } from '../../shared/config/apiConfig'
import { requestJSON } from '../../shared/http/requestJSON'
import { getWailsSessionBridge } from '../../shared/runtime/wailsSessionBridge'
import type { ApiResponse } from '../../types/apiResponse'
import type { AppSettings } from '../../types/appSettings'

export async function getSettings() {
  const bridge = getWailsSessionBridge()
  if (bridge?.GetSettings) {
    const settings = await bridge.GetSettings()
    return {
      code: 0,
      message: 'success',
      data: settings
    } satisfies ApiResponse<AppSettings>
  }

  return requestJSON<ApiResponse<AppSettings>>(`${getApiBaseUrl()}/api/v1/settings`)
}

export async function updateSettings(payload: AppSettings) {
  const bridge = getWailsSessionBridge()
  if (bridge?.UpdateSettings) {
    const settings = await bridge.UpdateSettings(payload)
    return {
      code: 0,
      message: 'success',
      data: settings
    } satisfies ApiResponse<AppSettings>
  }

  return requestJSON<ApiResponse<AppSettings>>(`${getApiBaseUrl()}/api/v1/settings`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(payload)
  })
}
