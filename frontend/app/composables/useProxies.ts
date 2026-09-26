import type { Proxy, ProxyApplyRequest } from '~/types'

export function useProxies() {
  const { apiFetch } = useApi()

  const items = useState<Proxy[]>('proxies-items', () => [])
  const loading = useState('proxies-loading', () => false)
  const error = useState<string | null>('proxies-error', () => null)

  async function refresh() {
    loading.value = true
    error.value = null
    try {
      items.value = await apiFetch<Proxy[]>('/api/v1/proxies')
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
    const proxy = await apiFetch<Proxy>('/api/v1/proxies', {
      method: 'POST',
      body,
    })
    await refresh()
    return proxy
  }

  async function remove(name: string) {
    await apiFetch(`/api/v1/proxies/${encodeURIComponent(name)}`, {
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
