export default defineNuxtRouteMiddleware(async () => {
  if (import.meta.server) {
    return
  }

  const auth = useAuth()
  await auth.ensureSession()

  if (auth.isAuthenticated.value) {
    return navigateTo('/')
  }
})
