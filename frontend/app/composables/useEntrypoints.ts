import type { Entrypoint, EntrypointApplyRequest, EntrypointListQuery, EntrypointListResponse } from '~/types'

const DEFAULT_PAGE_SIZE = 100

export function useEntrypoints() {
  const { apiFetch } = useApi()

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

  async function get(name: string) {
    return apiFetch<Entrypoint>(`/api/v1/entrypoints/${encodeURIComponent(name)}`)
  }

  async function apply(body: EntrypointApplyRequest) {
    return apiFetch<Entrypoint>('/api/v1/entrypoints', {
      method: 'POST',
      body,
    })
  }

  return { list, get, apply }
}
