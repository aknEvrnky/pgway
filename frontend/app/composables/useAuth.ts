export type AuthUser = {
  id: string
  role: string
}

type LoginResponse = {
  token: string
  user: AuthUser
}

export function useAuth() {
  const user = useState<AuthUser | null>('auth-user', () => null)
  const bearer = useState<string | null>('auth-bearer', () => null)
  const ready = useState('auth-ready', () => false)

  const { apiFetch } = useApi()

  const isAdmin = computed(() => user.value?.role === 'admin')
  const isAuthenticated = computed(() => !!user.value)

  async function login(username: string, password: string) {
    const res = await apiFetch<LoginResponse>('/api/v1/auth/login', {
      method: 'POST',
      body: { username, password },
      skipAuthRedirect: true,
    })
    bearer.value = res.token
    user.value = res.user
    ready.value = true
  }

  async function logout() {
    try {
      await apiFetch('/api/v1/auth/logout', {
        method: 'POST',
        skipAuthRedirect: true,
      })
    }
    catch {
      // always clear local session
    }
    bearer.value = null
    user.value = null
  }

  async function ensureSession() {
    if (ready.value) {
      return
    }
    try {
      // 401 here is expected when logged out — do not bounce to /login from apiFetch;
      // auth middleware decides navigation.
      const me = await apiFetch<AuthUser>('/api/v1/auth/me', { skipAuthRedirect: true })
      user.value = me
    }
    catch {
      user.value = null
      bearer.value = null
    }
    finally {
      ready.value = true
    }
  }

  return {
    user,
    bearer,
    ready,
    isAdmin,
    isAuthenticated,
    login,
    logout,
    ensureSession,
  }
}
