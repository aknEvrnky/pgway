<script setup lang="ts">
definePageMeta({
  layout: 'auth',
  middleware: ['guest'],
})

const auth = useAuth()
const route = useRoute()
const username = ref('')
const password = ref('')
const showPassword = ref(false)
const error = ref('')
const loading = ref(false)

async function onSubmit() {
  error.value = ''
  loading.value = true
  try {
    await auth.login(username.value.trim(), password.value)
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/'
    await navigateTo(redirect || '/')
  }
  catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Login failed'
  }
  finally {
    loading.value = false
  }
}
</script>

<template>
  <main class="relative z-10 w-full max-w-[420px] px-6 mx-auto min-h-screen flex flex-col items-center justify-center">
    <header class="flex flex-col items-center mb-12 text-center">
      <div class="relative mb-6">
        <div class="w-16 h-16 bg-surface-container rounded-xl flex items-center justify-center aura-glow border border-white/5">
          <svg class="w-9 h-9 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" d="M9 6v6m0 0l-3 4m3-4l3 4M15 6v12" />
            <circle cx="9" cy="4" r="1.5" fill="currentColor" stroke="none" />
            <circle cx="6" cy="18" r="1.5" fill="currentColor" stroke="none" />
            <circle cx="12" cy="18" r="1.5" fill="currentColor" stroke="none" />
            <circle cx="15" cy="4" r="1.5" fill="currentColor" stroke="none" />
            <circle cx="15" cy="20" r="1.5" fill="currentColor" stroke="none" />
          </svg>
        </div>
        <div class="absolute -bottom-1 -right-1 w-6 h-6 bg-primary-container rounded-full flex items-center justify-center shadow-lg">
          <svg class="w-3.5 h-3.5 text-on-primary-container" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
            <path d="M11 21h-1l1-7H7.5c-.88 0-.33-.75-.31-.78C8.48 10.94 10.42 7.54 13.01 3h1l-1 7h3.51c.4 0 .62.19.4.66C12.97 17.55 11 21 11 21z" />
          </svg>
        </div>
      </div>
      <h1 class="text-3xl font-black tracking-tighter text-on-surface mb-2">
        pgway
      </h1>
      <p class="text-sm uppercase tracking-[0.2em] text-outline">
        Proxy Gateway, Simplified.
      </p>
    </header>

    <div class="w-full glass-panel p-8 rounded-xl border border-outline-variant/15 shadow-2xl">
      <form class="space-y-6" @submit.prevent="onSubmit">
        <div class="space-y-2">
          <label class="block text-[11px] font-bold uppercase tracking-wider text-outline ml-1" for="username">
            Username
          </label>
          <div class="relative group">
            <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none text-outline group-focus-within:text-primary transition-colors">
              <svg class="w-[18px] h-[18px]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" aria-hidden="true">
                <path stroke-linecap="round" stroke-linejoin="round" d="M20 21v-2a4 4 0 00-4-4H8a4 4 0 00-4 4v2" />
                <circle cx="12" cy="7" r="4" />
              </svg>
            </div>
            <input
              id="username"
              v-model="username"
              name="username"
              type="text"
              autocomplete="username"
              placeholder="admin"
              required
              class="w-full bg-surface-container-lowest border border-outline-variant/20 focus:border-primary focus:ring-4 focus:ring-primary/20 text-on-surface placeholder:text-outline/40 rounded-xl py-3.5 pl-11 pr-4 transition-all duration-200 outline-none font-mono text-sm"
            >
          </div>
        </div>

        <div class="space-y-2">
          <label class="block text-[11px] font-bold uppercase tracking-wider text-outline px-1" for="password">
            Password
          </label>
          <div class="relative group">
            <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none text-outline group-focus-within:text-primary transition-colors">
              <svg class="w-[18px] h-[18px]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" aria-hidden="true">
                <rect x="3" y="11" width="18" height="11" rx="2" />
                <path stroke-linecap="round" d="M7 11V7a5 5 0 0110 0v4" />
              </svg>
            </div>
            <input
              id="password"
              v-model="password"
              name="password"
              :type="showPassword ? 'text' : 'password'"
              autocomplete="current-password"
              placeholder="••••••••••••"
              required
              class="w-full bg-surface-container-lowest border border-outline-variant/20 focus:border-primary focus:ring-4 focus:ring-primary/20 text-on-surface placeholder:text-outline/40 rounded-xl py-3.5 pl-11 pr-12 transition-all duration-200 outline-none font-mono text-sm"
            >
            <button
              type="button"
              class="absolute inset-y-0 right-0 pr-4 flex items-center text-outline hover:text-on-surface-variant transition-colors"
              :aria-label="showPassword ? 'Hide password' : 'Show password'"
              @click="showPassword = !showPassword"
            >
              <svg v-if="!showPassword" class="w-[18px] h-[18px]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" aria-hidden="true">
                <path stroke-linecap="round" stroke-linejoin="round" d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
                <circle cx="12" cy="12" r="3" />
              </svg>
              <svg v-else class="w-[18px] h-[18px]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" aria-hidden="true">
                <path stroke-linecap="round" stroke-linejoin="round" d="M17.94 17.94A10.07 10.07 0 0112 20c-7 0-11-8-11-8a18.45 18.45 0 015.06-5.94M9.9 4.24A9.12 9.12 0 0112 4c7 0 11 8 11 8a18.5 18.5 0 01-2.16 3.19m-6.72-1.07a3 3 0 11-4.24-4.24" />
                <line x1="1" y1="1" x2="23" y2="23" stroke-linecap="round" />
              </svg>
            </button>
          </div>
        </div>

        <p v-if="error" class="text-error text-sm" role="alert">
          {{ error }}
        </p>

        <button
          type="submit"
          :disabled="loading"
          class="w-full bg-gradient-to-br from-primary to-primary-container text-on-primary-container font-bold py-4 rounded-xl shadow-lg shadow-primary/10 hover:shadow-primary/20 active:scale-[0.98] transition-all duration-150 flex items-center justify-center gap-2 group disabled:opacity-60 disabled:pointer-events-none"
        >
          {{ loading ? 'Signing in…' : 'Login' }}
          <svg
            v-if="!loading"
            class="w-[18px] h-[18px] group-hover:translate-x-1 transition-transform"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            aria-hidden="true"
          >
            <line x1="5" y1="12" x2="19" y2="12" stroke-linecap="round" />
            <polyline points="12 5 19 12 12 19" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
        </button>
      </form>
    </div>
  </main>
</template>
