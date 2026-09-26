type FetchOptions = {
  method?: string
  body?: unknown
  headers?: Record<string, string>
}

type ApiErrorBody = {
  error?: string
}

export function useApi() {
  const config = useRuntimeConfig()
  const baseURL = (config.public.apiBase as string) || 'http://localhost:8081'
  // Shared with useAuth — prefer cookie; bearer is same-tab fallback after login.
  const bearer = useState<string | null>('auth-bearer', () => null)

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
      const msg = e?.data?.error || e?.message || 'request failed'
      const wrapped = new Error(msg) as Error & { statusCode?: number }
      wrapped.statusCode = e?.statusCode
      throw wrapped
    }
  }

  return { apiFetch, baseURL }
}
