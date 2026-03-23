<script setup lang="ts">
const route = useRoute()
const { isCollapsed, toggle } = useSidebar()

const navItems = [
  { label: 'Dashboard', to: '/', icon: 'dashboard' },
  { label: 'Proxies', to: '/proxies', icon: 'proxy' },
  { label: 'Pools', to: '/pools', icon: 'pool' },
  { label: 'LB', to: '/load-balancers', icon: 'lb' },
  { label: 'Routers', to: '/routers', icon: 'router' },
  { label: 'Flows', to: '/flows', icon: 'flow' },
  { label: 'Entrypoints', to: '/entrypoints', icon: 'entrypoint' },
  { label: 'Agents', to: '/agents', icon: 'agent' },
  { label: 'Settings', to: '/settings', icon: 'settings' },
]

const footerItems = [
  { label: 'Docs', icon: 'docs' },
  { label: 'Support', icon: 'support' },
]

function isActive(to: string) {
  return route.path === to
}
</script>

<template>
  <aside
    class="bg-surface-container-lowest h-screen fixed left-0 top-0 overflow-y-auto overflow-x-hidden flex flex-col py-6 border-r border-white/5 z-50 font-sans antialiased tracking-tight transition-all duration-300"
    :class="isCollapsed ? 'w-16' : 'w-64'"
  >
    <!-- Logo -->
    <div class="mb-10 flex items-center gap-3" :class="isCollapsed ? 'px-4 justify-center' : 'px-6'">
      <div class="w-8 h-8 rounded bg-gradient-to-br from-primary to-primary-container flex items-center justify-center shrink-0">
        <svg class="w-4 h-4 text-on-primary-container" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <path d="M4 6h16M4 12h16M4 18h16" stroke-linecap="round" />
        </svg>
      </div>
      <div v-if="!isCollapsed" class="overflow-hidden">
        <h1 class="text-xl font-black tracking-tighter text-primary">pgway</h1>
        <p class="text-[10px] text-slate-500 font-mono">v0.1.0-alpha</p>
      </div>
    </div>

    <!-- Navigation -->
    <nav class="flex-1 space-y-1" :class="isCollapsed ? 'px-2' : 'px-3'">
      <NuxtLink
        v-for="item in navItems"
        :key="item.to"
        :to="item.to"
        :title="isCollapsed ? item.label : undefined"
        class="flex items-center gap-3 py-2 transition-all active:scale-[0.98]"
        :class="[
          isCollapsed ? 'px-0 justify-center rounded-lg' : 'px-3 border-l-2',
          isActive(item.to)
            ? isCollapsed
              ? 'text-primary bg-primary/5'
              : 'text-primary border-primary bg-primary/5'
            : isCollapsed
              ? 'text-slate-400 hover:text-white hover:bg-white/5'
              : 'text-slate-400 hover:text-white hover:bg-white/5 border-transparent',
        ]"
      >
        <!-- Dashboard -->
        <svg v-if="item.icon === 'dashboard'" class="w-5 h-5 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <rect x="3" y="3" width="7" height="7" rx="1" />
          <rect x="14" y="3" width="7" height="7" rx="1" />
          <rect x="3" y="14" width="7" height="7" rx="1" />
          <rect x="14" y="14" width="7" height="7" rx="1" />
        </svg>
        <!-- Proxies -->
        <svg v-else-if="item.icon === 'proxy'" class="w-5 h-5 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <rect x="2" y="6" width="20" height="12" rx="2" />
          <circle cx="7" cy="12" r="1.5" fill="currentColor" />
          <line x1="12" y1="9" x2="12" y2="15" />
          <circle cx="17" cy="12" r="1.5" fill="currentColor" />
        </svg>
        <!-- Pools -->
        <svg v-else-if="item.icon === 'pool'" class="w-5 h-5 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <ellipse cx="12" cy="6" rx="8" ry="3" />
          <path d="M4 6v6c0 1.66 3.58 3 8 3s8-1.34 8-3V6" />
          <path d="M4 12v6c0 1.66 3.58 3 8 3s8-1.34 8-3v-6" />
        </svg>
        <!-- Load Balancer -->
        <svg v-else-if="item.icon === 'lb'" class="w-5 h-5 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <circle cx="12" cy="5" r="3" />
          <circle cx="5" cy="19" r="3" />
          <circle cx="19" cy="19" r="3" />
          <line x1="12" y1="8" x2="5" y2="16" />
          <line x1="12" y1="8" x2="19" y2="16" />
        </svg>
        <!-- Routers -->
        <svg v-else-if="item.icon === 'router'" class="w-5 h-5 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M6 3v12" />
          <path d="M18 9a3 3 0 100-6 3 3 0 000 6z" />
          <path d="M6 21a3 3 0 100-6 3 3 0 000 6z" />
          <path d="M15 6H9" />
          <path d="M18 9v3a3 3 0 01-3 3H6" />
        </svg>
        <!-- Flows -->
        <svg v-else-if="item.icon === 'flow'" class="w-5 h-5 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M5 4h4a2 2 0 012 2v4a2 2 0 01-2 2H5a2 2 0 01-2-2V6a2 2 0 012-2z" />
          <path d="M15 12h4a2 2 0 012 2v4a2 2 0 01-2 2h-4a2 2 0 01-2-2v-4a2 2 0 012-2z" />
          <path d="M9 8h4" />
          <path d="M11 8v6h2" />
        </svg>
        <!-- Entrypoints -->
        <svg v-else-if="item.icon === 'entrypoint'" class="w-5 h-5 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M15 3h4a2 2 0 012 2v14a2 2 0 01-2 2h-4" />
          <polyline points="10 17 15 12 10 7" />
          <line x1="15" y1="12" x2="3" y2="12" />
        </svg>
        <!-- Agents -->
        <svg v-else-if="item.icon === 'agent'" class="w-5 h-5 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <rect x="2" y="3" width="20" height="14" rx="2" />
          <line x1="8" y1="21" x2="16" y2="21" />
          <line x1="12" y1="17" x2="12" y2="21" />
        </svg>
        <!-- Settings -->
        <svg v-else-if="item.icon === 'settings'" class="w-5 h-5 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <circle cx="12" cy="12" r="3" />
          <path d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 010 2.83 2 2 0 01-2.83 0l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-4 0v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83-2.83l.06-.06A1.65 1.65 0 004.68 15a1.65 1.65 0 00-1.51-1H3a2 2 0 010-4h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 012.83-2.83l.06.06A1.65 1.65 0 009 4.68a1.65 1.65 0 001-1.51V3a2 2 0 014 0v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 012.83 2.83l-.06.06A1.65 1.65 0 0019.4 9a1.65 1.65 0 001.51 1H21a2 2 0 010 4h-.09a1.65 1.65 0 00-1.51 1z" />
        </svg>
        <span v-if="!isCollapsed" class="text-sm font-medium">{{ item.label }}</span>
      </NuxtLink>
    </nav>

    <!-- Footer -->
    <div class="mt-auto border-t border-white/5 pt-6 space-y-1" :class="isCollapsed ? 'px-2' : 'px-3'">
      <a
        v-for="item in footerItems"
        :key="item.label"
        href="#"
        :title="isCollapsed ? item.label : undefined"
        class="flex items-center gap-3 py-2 text-slate-400 hover:text-white hover:bg-white/5 transition-all"
        :class="isCollapsed ? 'px-0 justify-center rounded-lg' : 'px-3'"
      >
        <svg v-if="item.icon === 'docs'" class="w-5 h-5 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z" />
          <polyline points="14 2 14 8 20 8" />
          <line x1="16" y1="13" x2="8" y2="13" />
          <line x1="16" y1="17" x2="8" y2="17" />
        </svg>
        <svg v-else class="w-5 h-5 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <circle cx="12" cy="12" r="10" />
          <path d="M9.09 9a3 3 0 015.83 1c0 2-3 3-3 3" />
          <line x1="12" y1="17" x2="12.01" y2="17" />
        </svg>
        <span v-if="!isCollapsed" class="text-sm font-medium">{{ item.label }}</span>
      </a>

      <!-- Collapse toggle -->
      <button
        class="flex items-center gap-3 py-2 text-slate-400 hover:text-white hover:bg-white/5 transition-all w-full"
        :class="isCollapsed ? 'px-0 justify-center rounded-lg' : 'px-3'"
        @click="toggle"
      >
        <svg
          class="w-5 h-5 shrink-0 transition-transform duration-300"
          :class="isCollapsed ? 'rotate-180' : ''"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.5"
        >
          <polyline points="11 17 6 12 11 7" />
          <polyline points="18 17 13 12 18 7" />
        </svg>
        <span v-if="!isCollapsed" class="text-sm font-medium">Collapse</span>
      </button>
    </div>
  </aside>
</template>
