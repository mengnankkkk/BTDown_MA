import { getApiBaseUrl } from '../../shared/config/apiConfig'
import { requestJSON } from '../../shared/http/requestJSON'
import { getWailsSessionBridge } from '../../shared/runtime/wailsSessionBridge'
import type { ApiResponse } from '../../types/apiResponse'
import type { ObservabilityOverview } from '../../types/observabilityOverview'

export async function getObservabilityOverview() {
  const bridge = getWailsSessionBridge()
  if (bridge?.GetObservabilityOverview) {
    const overview = await bridge.GetObservabilityOverview()
    return {
      code: 0,
      message: 'success',
      data: overview
    } satisfies ApiResponse<ObservabilityOverview>
  }

  return requestJSON<ApiResponse<ObservabilityOverview>>(`${getApiBaseUrl()}/api/v1/observability/overview`)
}
