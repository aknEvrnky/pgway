import type { Entrypoint, EntrypointApplyRequest, EntrypointListQuery, EntrypointListResponse } from '~/types'

const DEFAULT_PAGE_SIZE = 20

export function useEntrypoints() {
  const { apiFetch } = useApi()

  const items = useState<Entrypoint[]>('entrypoints-items', () => [])
  const totalCount = useState('entrypoints-total-count', () => 0)
  const nextCursor = useState<string | null>('entrypoints-next-cursor', () => null)
  const loading = useState('entrypoints-loading', () => false)
  const error = useState<string | null>('entrypoints-error', () => null)

  async function list(query: EntrypointListQuery = {}) {
    const params = new URLSearchParams()
    const search = query.search?.trim()
    if (search) params.set('search', search)
    if (query.protocol) params.set('protocol', query.protocol)
    if (query.host) params.set('host', query.host)
    if (query.flow_id) params.set('flow_id', query.flow_id)
    params.set('page_size', String(query.page_size ?? DEFAULT_PAGE_SIZE))
    if (query.page_token) params.set('page_token', query.page_token)
    const qs = params.toString()
    return apiFetch<EntrypointListResponse>(`/api/v1/entrypoints${qs ? `?${qs}` : ''}`)
  }

  async function refresh(query: EntrypointListQuery = {}) {
    loading.value = true
    error.value = null
    try {
      const res = await list({ ...query, page_size: query.page_size ?? DEFAULT_PAGE_SIZE })
      items.value = res.items || []
      totalCount.value = res.total_count ?? 0
      nextCursor.value = res.next_cursor || null
      return res
    }
    catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'failed to load entrypoints'
      throw e
    }
    finally {
      loading.value = false
    }
  }

  async function get(name: string) {
    return apiFetch<Entrypoint>(`/api/v1/entrypoints/${encodeURIComponent(name)}`)
  }

  async function apply(body: EntrypointApplyRequest) {
    return apiFetch<Entrypoint>('/api/v1/entrypoints', {
      method: 'POST',
      body,
    })
  }

  async function remove(name: string) {
    await apiFetch(`/api/v1/entrypoints/${encodeURIComponent(name)}`, {
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
    list,
    refresh,
    get,
    apply,
    remove,
  }
}
