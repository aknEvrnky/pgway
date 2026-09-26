<script setup lang="ts">
const { isDark, toggle } = useDarkMode()
const { width } = useSidebar()
const auth = useAuth()
const toast = useAppToast()
const loggingOut = ref(false)
const menuOpen = ref(false)
const rootRef = ref<HTMLElement | null>(null)

const displayName = computed(() => auth.user.value?.id || 'User')
const roleLabel = computed(() => auth.user.value?.role === 'admin' ? 'Admin' : 'Member')
const initial = computed(() => (displayName.value[0] || '?').toUpperCase())

function toggleMenu() {
  menuOpen.value = !menuOpen.value
}

function closeMenu() {
  menuOpen.value = false
}

async function onLogout() {
  if (loggingOut.value) return
  loggingOut.value = true
  closeMenu()
  try {
    await auth.logout()
    toast.info('Signed out')
    await navigateTo('/login')
  }
  catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : 'Logout failed')
  }
  finally {
    loggingOut.value = false
  }
}

function onDocClick(e: MouseEvent) {
  if (!menuOpen.value || !rootRef.value) return
  if (!rootRef.value.contains(e.target as Node)) {
    closeMenu()
  }
}

function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape') closeMenu()
}

onMounted(() => {
  document.addEventListener('click', onDocClick)
  document.addEventListener('keydown', onKey)
})

onBeforeUnmount(() => {
  document.removeEventListener('click', onDocClick)
  document.removeEventListener('keydown', onKey)
})
</script>

<template>
  <header
    class="glass-panel sticky top-0 z-40 flex justify-between items-center h-16 px-8 border-b border-white/5 text-sm font-medium transition-all duration-300"
    :style="{ marginLeft: width, width: `calc(100% - ${width})` }"
  >
    <div class="flex items-center gap-4">
      <span class="text-slate-500">Infrastructure</span>
      <svg class="w-3 h-3 text-slate-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <polyline points="9 18 15 12 9 6" />
      </svg>
      <span class="text-primary font-bold">Dashboard</span>
    </div>

    <div class="flex items-center gap-6">
      <div class="flex items-center gap-4 text-slate-400">
        <button type="button" class="hover:text-white transition-colors active:opacity-80" aria-label="Toggle theme" @click="toggle">
          <svg v-if="isDark" class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z" />
          </svg>
          <svg v-else class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <circle cx="12" cy="12" r="5" />
            <line x1="12" y1="1" x2="12" y2="3" />
            <line x1="12" y1="21" x2="12" y2="23" />
            <line x1="4.22" y1="4.22" x2="5.64" y2="5.64" />
            <line x1="18.36" y1="18.36" x2="19.78" y2="19.78" />
            <line x1="1" y1="12" x2="3" y2="12" />
            <line x1="21" y1="12" x2="23" y2="12" />
            <line x1="4.22" y1="19.78" x2="5.64" y2="18.36" />
            <line x1="18.36" y1="5.64" x2="19.78" y2="4.22" />
          </svg>
        </button>

        <!-- Notifications (hidden until a real feed exists)
        <button type="button" class="relative hover:text-white transition-colors active:opacity-80" aria-label="Notifications">
          <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M18 8A6 6 0 006 8c0 7-3 9-3 9h18s-3-2-3-9" />
            <path d="M13.73 21a2 2 0 01-3.46 0" />
          </svg>
          <span class="absolute -top-0.5 -right-0.5 w-2 h-2 bg-primary rounded-full border-2 border-surface" />
        </button>
        -->
      </div>

      <div class="h-8 w-px bg-white/5" />

      <div ref="rootRef" class="relative flex items-center gap-3">
        <div class="text-right hidden sm:block">
          <p class="text-white text-xs font-semibold leading-tight">{{ displayName }}</p>
          <p class="text-slate-500 text-[10px] leading-tight">{{ roleLabel }}</p>
        </div>
        <button
          type="button"
          class="w-8 h-8 rounded-full bg-gradient-to-br from-primary to-primary-container flex items-center justify-center text-on-primary-container text-xs font-bold ring-offset-2 ring-offset-surface focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/50"
          :aria-expanded="menuOpen"
          aria-haspopup="menu"
          aria-label="Account menu"
          @click.stop="toggleMenu"
        >
          {{ initial }}
        </button>

        <div
          v-if="menuOpen"
          class="absolute right-0 top-[calc(100%+0.5rem)] w-48 glass-panel rounded-xl border border-outline-variant/20 shadow-2xl py-1.5 z-50"
          role="menu"
        >
          <div class="px-3 py-2 border-b border-outline-variant/15 sm:hidden">
            <p class="text-xs font-semibold text-on-surface">{{ displayName }}</p>
            <p class="text-[10px] text-on-surface-variant">{{ roleLabel }}</p>
          </div>
          <button
            type="button"
            class="w-full text-left px-3 py-2.5 text-sm text-on-surface-variant hover:text-error hover:bg-error/10 transition-colors disabled:opacity-50"
            role="menuitem"
            :disabled="loggingOut"
            @click="onLogout"
          >
            {{ loggingOut ? 'Signing out…' : 'Logout' }}
          </button>
        </div>
      </div>
    </div>
  </header>
</template>
