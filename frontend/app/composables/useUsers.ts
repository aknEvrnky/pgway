import type {
  ChangeUserPasswordRequest,
  CreateUserRequest,
  CreateUserResponse,
  UserListQuery,
  UserListResponse,
} from '~/types'

const DEFAULT_PAGE_SIZE = 20

export function useUsers() {
  const { apiFetch } = useApi()

  const items = useState<UserListResponse['items']>('users-items', () => [])
  const totalCount = useState('users-total-count', () => 0)
  const nextCursor = useState<string | null>('users-next-cursor', () => null)
  const loading = useState('users-loading', () => false)
  const error = useState<string | null>('users-error', () => null)

  async function refresh(query: UserListQuery = {}) {
    loading.value = true
    error.value = null
    try {
      const params = new URLSearchParams()
      const search = query.search?.trim()
      if (search) params.set('search', search)
      if (query.role) params.set('role', query.role)
      params.set('page_size', String(query.page_size ?? DEFAULT_PAGE_SIZE))
      if (query.page_token) params.set('page_token', query.page_token)

      const qs = params.toString()
      const res = await apiFetch<UserListResponse>(`/api/v1/users${qs ? `?${qs}` : ''}`)
      items.value = res.items || []
      totalCount.value = res.total_count ?? 0
      nextCursor.value = res.next_cursor || null
      return res
    }
    catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'failed to load users'
      throw e
    }
    finally {
      loading.value = false
    }
  }

  async function create(body: CreateUserRequest) {
    return apiFetch<CreateUserResponse>('/api/v1/users', {
      method: 'POST',
      body,
    })
  }

  async function remove(username: string) {
    await apiFetch(`/api/v1/users/${encodeURIComponent(username)}`, {
      method: 'DELETE',
    })
  }

  async function changePassword(username: string, body: ChangeUserPasswordRequest) {
    await apiFetch(
      `/api/v1/users/${encodeURIComponent(username)}/password`,
      {
        method: 'POST',
        body,
      },
    )
  }

  return {
    items,
    totalCount,
    nextCursor,
    loading,
    error,
    pageSize: DEFAULT_PAGE_SIZE,
    refresh,
    create,
    remove,
    changePassword,
  }
}
