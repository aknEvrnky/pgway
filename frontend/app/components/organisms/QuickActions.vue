<script setup lang="ts">
const { logLines } = useDashboardData()
</script>

<template>
  <div class="space-y-6">
    <!-- Quick Actions -->
    <div class="bg-surface-container rounded-xl ghost-border p-6">
      <h3 class="text-sm font-bold text-white mb-6">Quick Actions</h3>
      <div class="space-y-3">
        <MoleculesQuickActionButton label="Apply YAML" variant="primary">
          <template #icon>
            <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="16 18 22 12 16 6" />
              <polyline points="8 6 2 12 8 18" />
            </svg>
          </template>
        </MoleculesQuickActionButton>

        <MoleculesQuickActionButton label="Add Proxy">
          <template #icon>
            <svg class="w-4 h-4 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="3" y="3" width="18" height="18" rx="2" />
              <line x1="12" y1="8" x2="12" y2="16" />
              <line x1="8" y1="12" x2="16" y2="12" />
            </svg>
          </template>
        </MoleculesQuickActionButton>

        <MoleculesQuickActionButton label="Add Pool">
          <template #icon>
            <svg class="w-4 h-4 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <ellipse cx="12" cy="6" rx="8" ry="3" />
              <path d="M4 6v6c0 1.66 3.58 3 8 3s8-1.34 8-3V6" />
              <path d="M4 12v6c0 1.66 3.58 3 8 3s8-1.34 8-3v-6" />
            </svg>
          </template>
        </MoleculesQuickActionButton>
      </div>
    </div>

    <!-- Live Logs -->
    <div class="bg-surface-container-lowest rounded-xl p-4 font-mono text-[10px] text-primary/70 ghost-border">
      <div class="flex items-center justify-between mb-2">
        <span class="uppercase font-bold text-slate-600">Live Logs</span>
        <div class="flex gap-1">
          <span class="w-1.5 h-1.5 rounded-full bg-primary/40 animate-pulse" />
          <span class="w-1.5 h-1.5 rounded-full bg-primary/40 animate-pulse delay-75" />
        </div>
      </div>
      <div class="space-y-1">
        <MoleculesLogEntry v-for="(log, i) in logLines" :key="i" :log="log" />
      </div>
    </div>
  </div>
</template>
