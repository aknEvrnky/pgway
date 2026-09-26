export default defineNuxtRouteMiddleware(async (to) => {
  if (import.meta.server) {
    return
  }
  if (to.path === '/login') {
    return
  }

  const auth = useAuth()
  await auth.ensureSession()

  if (!auth.isAuthenticated.value) {
    return navigateTo({
      path: '/login',
      query: { redirect: to.fullPath },
    })
  }
})
