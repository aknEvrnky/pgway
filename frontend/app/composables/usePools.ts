import type { Pool, PoolApplyRequest, PoolListQuery, PoolListResponse } from '~/types'

const DEFAULT_PAGE_SIZE = 20

export function usePools() {
  const { apiFetch } = useApi()

  const items = useState<Pool[]>('pools-items', () => [])
  const totalCount = useState('pools-total-count', () => 0)
  const nextCursor = useState<string | null>('pools-next-cursor', () => null)
  const loading = useState('pools-loading', () => false)
  const error = useState<string | null>('pools-error', () => null)

  async function refresh(query: PoolListQuery = {}) {
    loading.value = true
    error.value = null
    try {
      const params = new URLSearchParams()
      const search = query.search?.trim()
      if (search) params.set('search', search)
      if (query.type) params.set('type', query.type)
      params.set('page_size', String(query.page_size ?? DEFAULT_PAGE_SIZE))
      if (query.page_token) params.set('page_token', query.page_token)

      const qs = params.toString()
      const res = await apiFetch<PoolListResponse>(`/api/v1/pools${qs ? `?${qs}` : ''}`)
      items.value = res.items || []
      totalCount.value = res.total_count ?? 0
      nextCursor.value = res.next_cursor || null
      return res
    }
    catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'failed to load pools'
      throw e
    }
    finally {
      loading.value = false
    }
  }

  async function get(name: string) {
    return apiFetch<Pool>(`/api/v1/pools/${encodeURIComponent(name)}`)
  }

  async function apply(body: PoolApplyRequest) {
    const pool = await apiFetch<Pool>('/api/v1/pools', {
      method: 'POST',
      body,
    })
    return pool
  }

  async function remove(name: string) {
    await apiFetch(`/api/v1/pools/${encodeURIComponent(name)}`, {
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
