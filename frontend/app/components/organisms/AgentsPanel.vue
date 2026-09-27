<script setup lang="ts">
import type { Agent, AgentStatus } from '~/types'

const { items, loading, error, activeCount, passiveCount, disconnectedCount, refresh } = useAgents()

onMounted(() => {
  refresh().catch(() => {})
})

function statusClass(status: AgentStatus) {
  switch (status) {
    case 'active':
      return 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30'
    case 'disconnected':
      return 'bg-amber-500/15 text-amber-400 border-amber-500/30'
    default:
      return 'bg-slate-500/15 text-slate-400 border-slate-500/30'
  }
}

function formatHeartbeat(agent: Agent) {
  if (!agent.last_heartbeat) return '—'
  try {
    return new Date(agent.last_heartbeat).toLocaleString()
  }
  catch {
    return agent.last_heartbeat
  }
}
</script>

<template>
  <div class="bg-surface-container rounded-xl ghost-border p-6 flex flex-col min-h-[280px]">
    <div class="flex justify-between items-start mb-6 gap-3">
      <div>
        <h3 class="text-sm font-bold text-white">Agents</h3>
        <p class="text-[10px] text-slate-500 uppercase tracking-wider mt-0.5">
          Data-plane registry
        </p>
      </div>
      <NuxtLink
        to="/agents"
        class="text-[10px] font-bold text-primary uppercase hover:underline shrink-0"
      >
        View all
      </NuxtLink>
    </div>

    <div class="flex flex-wrap gap-2 mb-4">
      <span class="text-[10px] font-bold uppercase tracking-wider px-2 py-1 rounded-md border border-emerald-500/30 text-emerald-400 bg-emerald-500/10">
        {{ activeCount }} active
      </span>
      <span class="text-[10px] font-bold uppercase tracking-wider px-2 py-1 rounded-md border border-amber-500/30 text-amber-400 bg-amber-500/10">
        {{ disconnectedCount }} disconnected
      </span>
      <span class="text-[10px] font-bold uppercase tracking-wider px-2 py-1 rounded-md border border-slate-500/30 text-slate-400 bg-slate-500/10">
        {{ passiveCount }} passive
      </span>
    </div>

    <div v-if="loading && !items.length" class="text-sm text-on-surface-variant py-8 text-center">
      Loading agents…
    </div>
    <div v-else-if="error" class="text-sm text-red-400 py-8 text-center">
      {{ error }}
    </div>
    <div v-else-if="!items.length" class="text-sm text-on-surface-variant py-8 text-center">
      No agents registered yet. Use
      <span class="font-mono text-primary">pgctl</span>
      to register a data-plane agent.
    </div>
    <ul v-else class="space-y-3 flex-1 overflow-auto">
      <li
        v-for="agent in items"
        :key="agent.id"
        class="flex items-start justify-between gap-3 py-2 border-b border-white/5 last:border-0"
      >
        <div class="min-w-0">
          <p class="font-mono text-sm font-semibold text-on-surface truncate">
            {{ agent.id }}
          </p>
          <p class="text-[11px] text-on-surface-variant truncate mt-0.5">
            <span v-if="agent.hostname">{{ agent.hostname }}</span>
            <span v-if="agent.hostname && agent.version"> · </span>
            <span v-if="agent.version">{{ agent.version }}</span>
            <span v-if="!agent.hostname && !agent.version">No hostname / version</span>
          </p>
          <p class="text-[10px] text-slate-500 mt-1">
            Last heartbeat: {{ formatHeartbeat(agent) }}
          </p>
        </div>
        <span
          class="shrink-0 text-[10px] font-bold uppercase tracking-wider px-2 py-1 rounded-md border"
          :class="statusClass(agent.status)"
        >
          {{ agent.status }}
        </span>
      </li>
    </ul>
  </div>
</template>
