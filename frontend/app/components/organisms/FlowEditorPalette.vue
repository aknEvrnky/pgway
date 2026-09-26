<script setup lang="ts">
import type { FlowGraphKind } from '~/types'
import { FLOW_NODE_COLORS } from '~/utils/flowNodeTheme'

const props = defineProps<{
  disabled?: boolean
}>()

const emit = defineEmits<{
  add: [kind: Exclude<FlowGraphKind, 'flow'>]
  create: []
}>()

/** Pipeline order; colors from shared theme (green / orange / purple / teal / blue). */
const items: { kind: Exclude<FlowGraphKind, 'flow'>, label: string }[] = [
  { kind: 'entrypoint', label: 'Entrypoint' },
  { kind: 'router', label: 'Router' },
  { kind: 'balancer', label: 'Balancer' },
  { kind: 'pool', label: 'Pool' },
  { kind: 'proxy', label: 'Proxy' },
]

function color(kind: FlowGraphKind) {
  return FLOW_NODE_COLORS[kind]
}
</script>

<template>
  <div class="absolute bottom-8 right-8 z-40 flex items-center gap-1.5 p-2 rounded-2xl bg-[#1a1f2e]/95 backdrop-blur-xl border border-white/10 shadow-2xl">
    <button
      v-for="item in items"
      :key="item.kind"
      type="button"
      class="group flex items-center justify-center w-11 h-11 rounded-xl bg-[#252b3b] hover:bg-[#2f3648] transition-colors disabled:opacity-40"
      :disabled="props.disabled"
      :title="`Create or wire ${item.label}`"
      :style="{ color: color(item.kind) }"
      @click="emit('add', item.kind)"
    >
      <!-- Entrypoint: arrow into bracket -->
      <svg
        v-if="item.kind === 'entrypoint'"
        class="w-5 h-5"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
        aria-hidden="true"
      >
        <path d="M3 12h11" />
        <path d="M10 8l4 4-4 4" />
        <path d="M17 4v16" />
        <path d="M17 4h3" />
        <path d="M17 20h3" />
      </svg>
      <!-- Router: Y-fork -->
      <svg
        v-else-if="item.kind === 'router'"
        class="w-5 h-5"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
        aria-hidden="true"
      >
        <path d="M12 20V10" />
        <path d="M12 10L5 4" />
        <path d="M12 10l7-6" />
        <path d="M5 4l-1.5 1.5M5 4L3.5 2.5" />
        <path d="M19 4l1.5 1.5M19 4l1.5-1.5" />
      </svg>
      <!-- Balancer: path + nodes -->
      <svg
        v-else-if="item.kind === 'balancer'"
        class="w-5 h-5"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
        aria-hidden="true"
      >
        <circle cx="6" cy="7" r="2.5" />
        <circle cx="18" cy="5" r="2.5" />
        <rect x="13.5" y="14" width="8" height="5" rx="2.5" />
        <path d="M8.2 8.5c1.8 3.5 4 5.5 7.3 5.5" />
      </svg>
      <!-- Pool: server stack -->
      <svg
        v-else-if="item.kind === 'pool'"
        class="w-5 h-5"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
        aria-hidden="true"
      >
        <rect x="4" y="3" width="16" height="5" rx="1" />
        <rect x="4" y="9.5" width="16" height="5" rx="1" />
        <rect x="4" y="16" width="16" height="5" rx="1" />
        <path d="M7 5.5h.01M7 12h.01M7 18.5h.01" />
      </svg>
      <!-- Proxy: globe -->
      <svg
        v-else
        class="w-5 h-5"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
        aria-hidden="true"
      >
        <circle cx="12" cy="12" r="9" />
        <path d="M3 12h18" />
        <path d="M12 3a14 14 0 014 9 14 14 0 01-4 9 14 14 0 01-4-9 14 14 0 014-9z" />
      </svg>
    </button>

    <div class="w-px h-7 bg-white/15 mx-0.5" aria-hidden="true" />

    <button
      type="button"
      class="flex items-center justify-center w-11 h-11 rounded-xl bg-[#252b3b] text-white/90 hover:bg-[#2f3648] hover:text-white transition-colors disabled:opacity-40"
      :disabled="props.disabled"
      title="Create new resource"
      aria-label="Create new resource"
      @click="emit('create')"
    >
      <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.25" aria-hidden="true">
        <line x1="12" y1="5" x2="12" y2="19" stroke-linecap="round" />
        <line x1="5" y1="12" x2="19" y2="12" stroke-linecap="round" />
      </svg>
    </button>
  </div>
</template>
