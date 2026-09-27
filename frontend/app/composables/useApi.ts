type FetchOptions = {
  method?: string
  body?: unknown
  headers?: Record<string, string>
  /** Skip 401 → clear session + /login redirect (login itself, etc.). */
  skipAuthRedirect?: boolean
}

type ApiErrorBody = {
  error?: string
}

function isAuthPublicPath(path: string) {
  return path === '/api/v1/auth/login' || path.startsWith('/api/v1/auth/login?')
}

export function useApi() {
  const config = useRuntimeConfig()
  // Empty string = same-origin (embedded UI). Do not coalesce "" with ||.
  const baseURL = (config.public.apiBase as string | undefined) ?? 'http://localhost:8081'
  // Shared with useAuth — prefer cookie; bearer is same-tab fallback after login.
  const bearer = useState<string | null>('auth-bearer', () => null)
  const user = useState<{ id: string, role: string } | null>('auth-user', () => null)
  const ready = useState('auth-ready', () => false)

  async function apiFetch<T>(path: string, opts: FetchOptions = {}): Promise<T> {
    const headers: Record<string, string> = {
      ...(opts.headers || {}),
    }

    if (opts.body !== undefined) {
      headers['Content-Type'] = 'application/json'
    }

    if (bearer.value && !headers.Authorization) {
      headers.Authorization = `Bearer ${bearer.value}`
    }

    try {
      return await $fetch<T>(path, {
        baseURL,
        method: (opts.method || 'GET') as 'GET' | 'POST' | 'PUT' | 'DELETE',
        body: opts.body as BodyInit | Record<string, unknown> | null | undefined,
        headers,
        credentials: 'include',
      })
    }
    catch (err: unknown) {
      const e = err as { data?: ApiErrorBody, statusCode?: number, message?: string }
      const status = e?.statusCode
      const msg = e?.data?.error || e?.message || 'request failed'
      const wrapped = new Error(msg) as Error & { statusCode?: number }
      wrapped.statusCode = status

      if (status === 401 && !opts.skipAuthRedirect && !isAuthPublicPath(path)) {
        bearer.value = null
        user.value = null
        ready.value = true
        if (import.meta.client) {
          const route = useRoute()
          if (route.path !== '/login') {
            await navigateTo({
              path: '/login',
              query: { redirect: route.fullPath },
            })
          }
        }
      }

      throw wrapped
    }
  }

  return { apiFetch, baseURL }
}
