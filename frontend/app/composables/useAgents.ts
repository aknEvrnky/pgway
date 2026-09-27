import type { Agent, AgentListQuery, AgentListResponse } from '~/types'

const DEFAULT_PAGE_SIZE = 50

export function useAgents() {
  const { apiFetch } = useApi()

  const items = useState<Agent[]>('agents-items', () => [])
  const totalCount = useState('agents-total-count', () => 0)
  const nextCursor = useState<string | null>('agents-next-cursor', () => null)
  const loading = useState('agents-loading', () => false)
  const error = useState<string | null>('agents-error', () => null)

  const activeCount = computed(() => items.value.filter(a => a.status === 'active').length)
  const passiveCount = computed(() => items.value.filter(a => a.status === 'passive').length)
  const disconnectedCount = computed(() => items.value.filter(a => a.status === 'disconnected').length)

  async function refresh(query: AgentListQuery = {}) {
    loading.value = true
    error.value = null
    try {
      const params = new URLSearchParams()
      const search = query.search?.trim()
      if (search) params.set('search', search)
      params.set('page_size', String(query.page_size ?? DEFAULT_PAGE_SIZE))
      if (query.page_token) params.set('page_token', query.page_token)

      const qs = params.toString()
      const res = await apiFetch<AgentListResponse>(`/api/v1/agents${qs ? `?${qs}` : ''}`)
      items.value = res.items || []
      totalCount.value = res.total_count ?? 0
      nextCursor.value = res.next_cursor || null
      return res
    }
    catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'failed to load agents'
      throw e
    }
    finally {
      loading.value = false
    }
  }

  return {
    items,
    totalCount,
    nextCursor,
    loading,
    error,
    activeCount,
    passiveCount,
    disconnectedCount,
    pageSize: DEFAULT_PAGE_SIZE,
    refresh,
  }
}
