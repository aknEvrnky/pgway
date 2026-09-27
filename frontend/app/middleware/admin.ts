export default defineNuxtRouteMiddleware(async () => {
  if (import.meta.server) {
    return
  }

  const auth = useAuth()
  await auth.ensureSession()

  if (!auth.isAuthenticated.value) {
    return navigateTo({
      path: '/login',
      query: { redirect: '/users' },
    })
  }

  if (!auth.isAdmin.value) {
    return navigateTo('/')
  }
})
