import type { BalancerApplyRequest, BalancerListQuery, BalancerListResponse, LoadBalancer } from '~/types'

const DEFAULT_PAGE_SIZE = 20

export function useBalancers() {
  const { apiFetch } = useApi()

  const items = useState<LoadBalancer[]>('balancers-items', () => [])
  const totalCount = useState('balancers-total-count', () => 0)
  const nextCursor = useState<string | null>('balancers-next-cursor', () => null)
  const loading = useState('balancers-loading', () => false)
  const error = useState<string | null>('balancers-error', () => null)

  async function refresh(query: BalancerListQuery = {}) {
    loading.value = true
    error.value = null
    try {
      const params = new URLSearchParams()
      const search = query.search?.trim()
      if (search) params.set('search', search)
      if (query.type) params.set('type', query.type)
      if (query.pool_id) params.set('pool_id', query.pool_id)
      params.set('page_size', String(query.page_size ?? DEFAULT_PAGE_SIZE))
      if (query.page_token) params.set('page_token', query.page_token)

      const qs = params.toString()
      const res = await apiFetch<BalancerListResponse>(`/api/v1/balancers${qs ? `?${qs}` : ''}`)
      items.value = res.items || []
      totalCount.value = res.total_count ?? 0
      nextCursor.value = res.next_cursor || null
      return res
    }
    catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'failed to load balancers'
      throw e
    }
    finally {
      loading.value = false
    }
  }

  async function get(name: string) {
    return apiFetch<LoadBalancer>(`/api/v1/balancers/${encodeURIComponent(name)}`)
  }

  async function apply(body: BalancerApplyRequest) {
    return apiFetch<LoadBalancer>('/api/v1/balancers', {
      method: 'POST',
      body,
    })
  }

  async function remove(name: string) {
    await apiFetch(`/api/v1/balancers/${encodeURIComponent(name)}`, {
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
