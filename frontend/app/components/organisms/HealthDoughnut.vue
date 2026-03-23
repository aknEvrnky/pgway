<script setup lang="ts">
const { healthDistribution } = useDashboardData()

const total = computed(() =>
  healthDistribution.healthy + healthDistribution.degraded + healthDistribution.down
)

const circumference = 2 * Math.PI * 70 // ~440

const healthyOffset = computed(() => {
  const ratio = healthDistribution.healthy / total.value
  return circumference - circumference * ratio
})

const degradedOffset = computed(() => {
  const ratio = healthDistribution.degraded / total.value
  return circumference - circumference * ratio
})

const degradedRotation = computed(() => {
  return (healthDistribution.healthy / total.value) * 360
})

const downOffset = computed(() => {
  const ratio = healthDistribution.down / total.value
  return circumference - circumference * ratio
})

const downRotation = computed(() => {
  return ((healthDistribution.healthy + healthDistribution.degraded) / total.value) * 360
})

const legend = computed(() => [
  { label: 'Healthy', value: healthDistribution.healthy, color: 'bg-primary' },
  { label: 'Degraded', value: healthDistribution.degraded, color: 'bg-tertiary' },
  { label: 'Down', value: healthDistribution.down, color: 'bg-error' },
])
</script>

<template>
  <div class="bg-surface-container rounded-xl ghost-border p-6">
    <h3 class="text-sm font-bold text-white mb-8">System Health</h3>

    <!-- Doughnut -->
    <div class="relative flex justify-center items-center py-6">
      <svg class="w-40 h-40" viewBox="0 0 160 160">
        <!-- Background ring -->
        <circle
          cx="80" cy="80" r="70"
          fill="transparent"
          stroke="#2d3449"
          stroke-width="12"
        />
        <!-- Healthy -->
        <circle
          cx="80" cy="80" r="70"
          fill="transparent"
          stroke="#8ed5ff"
          stroke-width="12"
          :stroke-dasharray="circumference"
          :stroke-dashoffset="healthyOffset"
          transform="rotate(-90 80 80)"
        />
        <!-- Degraded -->
        <circle
          cx="80" cy="80" r="70"
          fill="transparent"
          stroke="#ffc176"
          stroke-width="12"
          :stroke-dasharray="circumference"
          :stroke-dashoffset="degradedOffset"
          :transform="`rotate(${degradedRotation - 90} 80 80)`"
        />
        <!-- Down -->
        <circle
          cx="80" cy="80" r="70"
          fill="transparent"
          stroke="#ffb4ab"
          stroke-width="12"
          :stroke-dasharray="circumference"
          :stroke-dashoffset="downOffset"
          :transform="`rotate(${downRotation - 90} 80 80)`"
        />
      </svg>
      <!-- Center text -->
      <div class="absolute inset-0 flex flex-col items-center justify-center">
        <p class="text-3xl font-black text-white">{{ total }}</p>
        <p class="text-[9px] text-slate-500 uppercase font-bold">Total Nodes</p>
      </div>
    </div>

    <!-- Legend -->
    <div class="mt-6 space-y-3">
      <div v-for="item in legend" :key="item.label" class="flex justify-between items-center text-xs">
        <div class="flex items-center gap-2">
          <span class="w-2 h-2 rounded-full" :class="item.color" />
          <span class="text-slate-400">{{ item.label }}</span>
        </div>
        <span class="font-mono text-white">{{ item.value }}</span>
      </div>
    </div>
  </div>
</template>
