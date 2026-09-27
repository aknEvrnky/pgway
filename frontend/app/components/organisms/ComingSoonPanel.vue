<script setup lang="ts">
defineProps<{
  title: string
  subtitle?: string
}>()
</script>

<template>
  <div class="bg-surface-container rounded-xl ghost-border p-6 flex flex-col min-h-[240px]">
    <div class="mb-6">
      <h3 class="text-sm font-bold text-white">{{ title }}</h3>
      <p v-if="subtitle" class="text-[10px] text-slate-500 uppercase tracking-wider mt-0.5">
        {{ subtitle }}
      </p>
    </div>
    <div class="flex-1 flex items-center justify-center text-center px-4">
      <div>
        <p class="text-sm font-semibold text-on-surface-variant">Coming soon</p>
        <p class="text-[11px] text-slate-500 mt-1 leading-relaxed">
          Live metrics will appear here once telemetry aggregations ship.
        </p>
      </div>
    </div>
  </div>
</template>
