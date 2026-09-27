<script setup lang="ts">
import type { Agent, AgentStatus } from '~/types'

const { items, loading, error, totalCount, activeCount, passiveCount, disconnectedCount, refresh } = useAgents()

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
  <div class="space-y-6">
    <div class="flex flex-col sm:flex-row sm:items-end sm:justify-between gap-4">
      <div>
        <p class="text-xs font-bold uppercase tracking-widest text-primary mb-1">
          Cluster
        </p>
        <h2 class="text-3xl font-black tracking-tighter text-on-surface">
          Agents
        </h2>
        <p class="text-sm text-on-surface-variant mt-2 max-w-xl">
          Registered data-plane agents. Register and manage credentials with
          <span class="font-mono text-primary">pgctl</span>
          / gRPC; this page is read-only.
        </p>
      </div>
      <div class="flex flex-wrap gap-2">
        <span class="text-[10px] font-bold uppercase tracking-wider px-2 py-1 rounded-md border border-outline-variant/30 text-on-surface-variant">
          {{ totalCount }} total
        </span>
        <span class="text-[10px] font-bold uppercase tracking-wider px-2 py-1 rounded-md border border-emerald-500/30 text-emerald-400">
          {{ activeCount }} active
        </span>
        <span class="text-[10px] font-bold uppercase tracking-wider px-2 py-1 rounded-md border border-amber-500/30 text-amber-400">
          {{ disconnectedCount }} disconnected
        </span>
        <span class="text-[10px] font-bold uppercase tracking-wider px-2 py-1 rounded-md border border-slate-500/30 text-slate-400">
          {{ passiveCount }} passive
        </span>
      </div>
    </div>

    <div class="bg-surface-container rounded-xl ghost-border overflow-hidden">
      <div v-if="loading && !items.length" class="px-6 py-12 text-center text-sm text-on-surface-variant">
        Loading agents…
      </div>
      <div v-else-if="error" class="px-6 py-12 text-center text-sm text-red-400">
        {{ error }}
      </div>
      <div v-else-if="!items.length" class="px-6 py-12 text-center text-sm text-on-surface-variant">
        No agents yet.
      </div>
      <table v-else class="w-full text-left">
        <thead>
          <tr class="border-b border-white/5 text-[10px] font-bold uppercase tracking-widest text-on-surface-variant">
            <th class="px-6 py-3">ID</th>
            <th class="px-6 py-3">Status</th>
            <th class="px-6 py-3">Hostname</th>
            <th class="px-6 py-3">Version</th>
            <th class="px-6 py-3">Last heartbeat</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="agent in items"
            :key="agent.id"
            class="border-b border-white/5 last:border-0"
          >
            <td class="px-6 py-4 font-mono text-sm font-semibold text-on-surface">
              {{ agent.id }}
            </td>
            <td class="px-6 py-4">
              <span
                class="text-[10px] font-bold uppercase tracking-wider px-2 py-1 rounded-md border"
                :class="statusClass(agent.status)"
              >
                {{ agent.status }}
              </span>
            </td>
            <td class="px-6 py-4 text-sm text-on-surface-variant">
              {{ agent.hostname || '—' }}
            </td>
            <td class="px-6 py-4 text-sm text-on-surface-variant font-mono">
              {{ agent.version || '—' }}
            </td>
            <td class="px-6 py-4 text-sm text-on-surface-variant whitespace-nowrap">
              {{ formatHeartbeat(agent) }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
