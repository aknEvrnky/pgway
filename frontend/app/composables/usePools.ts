import type { Pool, PoolApplyRequest } from '~/types'

export function usePools() {
  const { apiFetch } = useApi()

  const items = useState<Pool[]>('pools-items', () => [])
  const loading = useState('pools-loading', () => false)
  const error = useState<string | null>('pools-error', () => null)

  async function refresh() {
    loading.value = true
    error.value = null
    try {
      items.value = await apiFetch<Pool[]>('/api/v1/pools')
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
    await refresh()
    return pool
  }

  async function remove(name: string) {
    await apiFetch(`/api/v1/pools/${encodeURIComponent(name)}`, {
      method: 'DELETE',
    })
    await refresh()
  }

  return {
    items,
    loading,
    error,
    refresh,
    get,
    apply,
    remove,
  }
}
