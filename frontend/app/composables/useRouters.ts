import type { Router, RouterApplyRequest, RouterListQuery, RouterListResponse } from '~/types'

const DEFAULT_PAGE_SIZE = 20

export function useRouters() {
  const { apiFetch } = useApi()

  const items = useState<Router[]>('routers-items', () => [])
  const totalCount = useState('routers-total-count', () => 0)
  const nextCursor = useState<string | null>('routers-next-cursor', () => null)
  const loading = useState('routers-loading', () => false)
  const error = useState<string | null>('routers-error', () => null)

  async function refresh(query: RouterListQuery = {}) {
    loading.value = true
    error.value = null
    try {
      const params = new URLSearchParams()
      const search = query.search?.trim()
      if (search) params.set('search', search)
      if (query.has_catch_all === true) params.set('has_catch_all', 'true')
      if (query.has_catch_all === false) params.set('has_catch_all', 'false')
      if (query.target) params.set('target', query.target)
      params.set('page_size', String(query.page_size ?? DEFAULT_PAGE_SIZE))
      if (query.page_token) params.set('page_token', query.page_token)

      const qs = params.toString()
      const res = await apiFetch<RouterListResponse>(`/api/v1/routers${qs ? `?${qs}` : ''}`)
      items.value = res.items || []
      totalCount.value = res.total_count ?? 0
      nextCursor.value = res.next_cursor || null
      return res
    }
    catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'failed to load routers'
      throw e
    }
    finally {
      loading.value = false
    }
  }

  async function get(name: string) {
    return apiFetch<Router>(`/api/v1/routers/${encodeURIComponent(name)}`)
  }

  async function apply(body: RouterApplyRequest) {
    return apiFetch<Router>('/api/v1/routers', {
      method: 'POST',
      body,
    })
  }

  async function remove(name: string) {
    await apiFetch(`/api/v1/routers/${encodeURIComponent(name)}`, {
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
