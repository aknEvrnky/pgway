<script setup lang="ts">
const { throughputData, selectedRange } = useDashboardData()
</script>

<template>
  <div class="bg-surface-container rounded-xl ghost-border p-6 flex flex-col">
    <!-- Header -->
    <div class="flex justify-between items-center mb-8">
      <div>
        <h3 class="text-sm font-bold text-white">Throughput Trends</h3>
        <p class="text-[10px] text-slate-500 uppercase tracking-wider">Historical Request Volume</p>
      </div>
      <MoleculesTimeRangeToggle v-model="selectedRange" />
    </div>

    <!-- Chart -->
    <div class="flex-1 min-h-[240px] relative">
      <!-- Y-Axis -->
      <div class="absolute left-0 top-0 h-full flex flex-col justify-between text-[10px] text-slate-600 font-mono pr-4 border-r border-white/5">
        <span>3.0k</span>
        <span>2.0k</span>
        <span>1.0k</span>
        <span>0</span>
      </div>

      <!-- Bars -->
      <div class="absolute inset-0 pl-12 flex items-end justify-between gap-1 px-2">
        <div
          v-for="(bar, i) in throughputData"
          :key="i"
          class="w-full bg-gradient-to-t from-primary/20 to-transparent border-t-2 border-primary/50 rounded-t-sm transition-all duration-300"
          :style="{ height: `${bar.value}%` }"
        />
      </div>
    </div>
  </div>
</template>
