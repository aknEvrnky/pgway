<script setup lang="ts">
import type { Flow } from '~/types'
import type { FlowGraphKind } from '~/types'
import { flowNodeInline, FLOW_NODE_COLORS } from '~/utils/flowNodeTheme'

const props = defineProps<{
  flow: Flow
}>()

type PreviewNode = {
  key: string
  kind: Extract<FlowGraphKind, 'entrypoint' | 'flow' | 'router' | 'balancer'>
  label: string
  active: boolean
}

const nodes = computed((): PreviewNode[] => {
  const f = props.flow
  const list: PreviewNode[] = [
    { key: 'ep', kind: 'entrypoint', label: 'EP', active: true },
    { key: 'flow', kind: 'flow', label: f.id, active: true },
  ]
  if (f.router_id) {
    list.push({ key: 'router', kind: 'router', label: f.router_id, active: true })
  }
  if (f.balancer_id) {
    list.push({ key: 'balancer', kind: 'balancer', label: f.balancer_id, active: true })
  }
  else if (!f.router_id) {
    list.push({ key: 'balancer-missing', kind: 'balancer', label: 'LB?', active: false })
  }
  return list
})

function shortLabel(s: string, max = 14): string {
  if (s.length <= max) return s
  return `${s.slice(0, max - 1)}…`
}

function chipStyle(kind: PreviewNode['kind'], active: boolean) {
  const s = flowNodeInline(kind)
  return {
    borderColor: active ? s.borderColor : `${FLOW_NODE_COLORS[kind]}40`,
    backgroundColor: active ? s.backgroundColor : 'transparent',
    color: s.color,
  }
}
</script>

<template>
  <div
    class="inline-flex items-center gap-0 max-w-full overflow-x-auto py-0.5"
    :title="nodes.filter(n => n.active).map(n => `${n.kind}: ${n.label}`).join(' → ')"
    role="img"
    :aria-label="`Pipeline: ${nodes.map(n => n.label).join(' to ')}`"
  >
    <template v-for="(node, i) in nodes" :key="node.key">
      <div
        class="shrink-0 flex flex-col items-center gap-0.5 px-2 py-1 rounded-md border text-[10px] font-mono font-semibold leading-tight max-w-[7.5rem]"
        :class="node.active ? '' : 'opacity-45 border-dashed'"
        :style="chipStyle(node.kind, node.active)"
      >
        <span class="text-[8px] font-black uppercase tracking-wider opacity-80 font-sans">
          {{ node.kind === 'entrypoint' ? 'EP' : node.kind === 'balancer' ? 'LB' : node.kind }}
        </span>
        <span class="truncate w-full text-center text-on-surface">{{ shortLabel(node.label) }}</span>
      </div>
      <svg
        v-if="i < nodes.length - 1"
        class="w-4 h-3 shrink-0 text-outline/50 mx-0.5"
        viewBox="0 0 16 12"
        fill="none"
        aria-hidden="true"
      >
        <path d="M1 6h12M10 2l4 4-4 4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
      </svg>
    </template>
  </div>
</template>
