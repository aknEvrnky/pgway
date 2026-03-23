<script setup lang="ts">
import type { MetricCardData } from '~/types'

defineProps<{
  data: MetricCardData
}>()
</script>

<template>
  <div class="bg-surface-container p-6 rounded-xl ghost-border ghost-border-hover transition-all relative overflow-hidden">
    <!-- Badge (top right) -->
    <div v-if="data.badge?.variant === 'warning'" class="absolute top-0 right-0 p-4">
      <AtomsStatusBadge :text="data.badge.text" :variant="data.badge.variant" />
    </div>

    <!-- Icon + Trend -->
    <div class="flex justify-between items-start mb-4">
      <slot name="icon" />
      <AtomsTrendIndicator
        v-if="data.trend"
        :direction="data.trend.direction"
        :percentage="data.trend.percentage"
      />
      <div v-else-if="data.badge && data.badge.variant !== 'warning'" class="flex items-center gap-1.5">
        <AtomsPulseDot status="active" />
        <span class="text-[10px] font-bold text-primary">{{ data.badge.text }}</span>
      </div>
    </div>

    <!-- Label -->
    <p class="text-slate-400 text-xs font-medium mb-1">{{ data.label }}</p>

    <!-- Value -->
    <p class="text-2xl font-black text-white tracking-tight">{{ data.value }}</p>

    <!-- Progress bar -->
    <div v-if="data.progress" class="mt-4 h-1 w-full bg-surface-container-highest rounded-full overflow-hidden">
      <div class="h-full bg-primary rounded-full" :style="{ width: `${data.progress}%` }" />
    </div>

    <!-- Subtitle -->
    <p v-if="data.subtitle" class="text-[10px] text-slate-500 mt-2 font-mono uppercase">
      {{ data.subtitle }}
    </p>
  </div>
</template>
