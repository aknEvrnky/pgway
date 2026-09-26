import type { Proxy, ProxyApplyRequest, ProxyListQuery, ProxyListResponse } from '~/types'

const DEFAULT_PAGE_SIZE = 20

export function useProxies() {
  const { apiFetch } = useApi()

  const items = useState<Proxy[]>('proxies-items', () => [])
  const totalCount = useState('proxies-total-count', () => 0)
  const nextCursor = useState<string | null>('proxies-next-cursor', () => null)
  const loading = useState('proxies-loading', () => false)
  const error = useState<string | null>('proxies-error', () => null)

  async function refresh(query: ProxyListQuery = {}) {
    loading.value = true
    error.value = null
    try {
      const params = new URLSearchParams()
      const search = query.search?.trim()
      if (search) params.set('search', search)
      if (query.protocol) params.set('protocol', query.protocol)
      params.set('page_size', String(query.page_size ?? DEFAULT_PAGE_SIZE))
      if (query.page_token) params.set('page_token', query.page_token)

      const qs = params.toString()
      const res = await apiFetch<ProxyListResponse>(`/api/v1/proxies${qs ? `?${qs}` : ''}`)
      items.value = res.items || []
      totalCount.value = res.total_count ?? 0
      nextCursor.value = res.next_cursor || null
      return res
    }
    catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'failed to load proxies'
      throw e
    }
    finally {
      loading.value = false
    }
  }

  async function get(name: string) {
    return apiFetch<Proxy>(`/api/v1/proxies/${encodeURIComponent(name)}`)
  }

  async function apply(body: ProxyApplyRequest) {
    return apiFetch<Proxy>('/api/v1/proxies', {
      method: 'POST',
      body,
    })
  }

  async function remove(name: string) {
    await apiFetch(`/api/v1/proxies/${encodeURIComponent(name)}`, {
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
