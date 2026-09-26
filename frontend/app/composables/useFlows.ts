import type { Flow, FlowApplyRequest, FlowListQuery, FlowListResponse } from '~/types'

const DEFAULT_PAGE_SIZE = 20

export function useFlows() {
  const { apiFetch } = useApi()

  const items = useState<Flow[]>('flows-items', () => [])
  const totalCount = useState('flows-total-count', () => 0)
  const nextCursor = useState<string | null>('flows-next-cursor', () => null)
  const loading = useState('flows-loading', () => false)
  const error = useState<string | null>('flows-error', () => null)

  async function refresh(query: FlowListQuery = {}) {
    loading.value = true
    error.value = null
    try {
      const params = new URLSearchParams()
      const search = query.search?.trim()
      if (search) params.set('search', search)
      if (query.mode) params.set('mode', query.mode)
      if (query.router_id) params.set('router_id', query.router_id)
      if (query.balancer_id) params.set('balancer_id', query.balancer_id)
      params.set('page_size', String(query.page_size ?? DEFAULT_PAGE_SIZE))
      if (query.page_token) params.set('page_token', query.page_token)

      const qs = params.toString()
      const res = await apiFetch<FlowListResponse>(`/api/v1/flows${qs ? `?${qs}` : ''}`)
      items.value = res.items || []
      totalCount.value = res.total_count ?? 0
      nextCursor.value = res.next_cursor || null
      return res
    }
    catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'failed to load flows'
      throw e
    }
    finally {
      loading.value = false
    }
  }

  async function get(name: string) {
    return apiFetch<Flow>(`/api/v1/flows/${encodeURIComponent(name)}`)
  }

  async function apply(body: FlowApplyRequest) {
    return apiFetch<Flow>('/api/v1/flows', {
      method: 'POST',
      body,
    })
  }

  async function remove(name: string) {
    await apiFetch(`/api/v1/flows/${encodeURIComponent(name)}`, {
      method: 'DELETE',
    })
  }

  return {
    items,
    totalCount,
    nextCursor,
    loading,
    error,
    pageSize: DEFAULT_PAGE_SIZE,
    refresh,
    get,
    apply,
    remove,
  }
}
