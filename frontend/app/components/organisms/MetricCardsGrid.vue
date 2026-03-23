<script setup lang="ts">
const { metrics } = useDashboardData()
</script>

<template>
  <section class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
    <MoleculesMetricCard v-for="(m, i) in metrics" :key="i" :data="m">
      <template #icon>
        <!-- Analytics -->
        <span v-if="m.icon === 'analytics'" class="text-primary bg-primary/10 p-2 rounded-lg inline-flex">
          <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M18 20V10M12 20V4M6 20v-6" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
        </span>
        <!-- Speed -->
        <span v-else-if="m.icon === 'speed'" class="text-primary bg-primary/10 p-2 rounded-lg inline-flex">
          <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
        </span>
        <!-- Hub -->
        <span v-else-if="m.icon === 'hub'" class="text-primary bg-primary/10 p-2 rounded-lg inline-flex">
          <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <circle cx="12" cy="12" r="3" />
            <circle cx="4" cy="6" r="2" />
            <circle cx="20" cy="6" r="2" />
            <circle cx="4" cy="18" r="2" />
            <circle cx="20" cy="18" r="2" />
            <line x1="9.5" y1="10" x2="5.5" y2="7.5" />
            <line x1="14.5" y1="10" x2="18.5" y2="7.5" />
            <line x1="9.5" y1="14" x2="5.5" y2="16.5" />
            <line x1="14.5" y1="14" x2="18.5" y2="16.5" />
          </svg>
        </span>
        <!-- DNS/Agents -->
        <span v-else class="text-primary bg-primary/10 p-2 rounded-lg inline-flex">
          <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <rect x="2" y="3" width="20" height="6" rx="1" />
            <rect x="2" y="15" width="20" height="6" rx="1" />
            <circle cx="6" cy="6" r="1" fill="currentColor" />
            <circle cx="6" cy="18" r="1" fill="currentColor" />
          </svg>
        </span>
      </template>
    </MoleculesMetricCard>
  </section>
</template>
