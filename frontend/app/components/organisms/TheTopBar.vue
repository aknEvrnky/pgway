<script setup lang="ts">
const { isDark, toggle } = useDarkMode()
const { width } = useSidebar()
</script>

<template>
  <header
    class="glass-panel sticky top-0 z-40 flex justify-between items-center h-16 px-8 border-b border-white/5 text-sm font-medium transition-all duration-300"
    :style="{ marginLeft: width, width: `calc(100% - ${width})` }"
  >
    <!-- Breadcrumbs -->
    <div class="flex items-center gap-4">
      <span class="text-slate-500">Infrastructure</span>
      <svg class="w-3 h-3 text-slate-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <polyline points="9 18 15 12 9 6" />
      </svg>
      <span class="text-primary font-bold">Dashboard</span>
    </div>

    <!-- Right side -->
    <div class="flex items-center gap-6">
      <div class="flex items-center gap-4 text-slate-400">
        <!-- Dark mode toggle -->
        <button class="hover:text-white transition-colors active:opacity-80" @click="toggle">
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

        <!-- Notifications -->
        <button class="relative hover:text-white transition-colors active:opacity-80">
          <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M18 8A6 6 0 006 8c0 7-3 9-3 9h18s-3-2-3-9" />
            <path d="M13.73 21a2 2 0 01-3.46 0" />
          </svg>
          <span class="absolute -top-0.5 -right-0.5 w-2 h-2 bg-primary rounded-full border-2 border-surface" />
        </button>
      </div>

      <!-- Divider -->
      <div class="h-8 w-px bg-white/5" />

      <!-- User -->
      <div class="flex items-center gap-3 cursor-pointer active:opacity-80">
        <div class="text-right">
          <p class="text-white text-xs font-semibold leading-tight">Admin User</p>
          <p class="text-slate-500 text-[10px] leading-tight">Project Owner</p>
        </div>
        <div class="w-8 h-8 rounded-full bg-gradient-to-br from-primary to-primary-container flex items-center justify-center text-on-primary-container text-xs font-bold">
          A
        </div>
      </div>
    </div>
  </header>
</template>
