<script setup lang="ts">
import { Handle, Position } from '@vue-flow/core'
import type { FlowGraphKind } from '~/types'
import { flowNodeInline, FLOW_NODE_COLORS } from '~/utils/flowNodeTheme'

const props = defineProps<{
  data: {
    kind: FlowGraphKind
    label: string
    subtitle?: string
    resourceId: string
    dirty?: boolean
    sourceHandles?: number
  }
}>()

const kindLabel: Record<string, string> = {
  entrypoint: 'Entrypoint',
  flow: 'Flow',
  router: 'Router',
  balancer: 'Balancer',
  pool: 'Pool',
  proxy: 'Proxy',
}

const style = computed(() => flowNodeInline(props.data.kind))
const accent = computed(() => FLOW_NODE_COLORS[props.data.kind] || '#94a3b8')

const sourceHandleIds = computed(() => {
  const n = props.data.sourceHandles
  if (!n || n <= 1) return [null] as (string | null)[]
  return Array.from({ length: n }, (_, i) => `out-${i}`)
})

function sourceHandleTop(index: number, total: number): string {
  if (total <= 1) return '50%'
  // Spread handles between ~22% and ~78% of node height.
  const t = 22 + (index / (total - 1)) * 56
  return `${t}%`
}
</script>

<template>
  <div
    class="w-[200px] rounded-xl border px-3 py-2.5 shadow-lg text-left box-border relative"
    :class="data.dirty ? 'ring-2 ring-amber-400/60' : ''"
    :style="{
      borderColor: style.borderColor,
      backgroundColor: style.backgroundColor,
    }"
  >
    <Handle
      type="target"
      :position="Position.Left"
      class="!w-2 !h-2 !border-0"
      :style="{ backgroundColor: accent }"
    />
    <p
      class="text-[9px] font-black uppercase tracking-widest mb-0.5"
      :style="{ color: accent }"
    >
      {{ kindLabel[data.kind] || data.kind }}
    </p>
    <p class="font-mono text-sm font-semibold text-on-surface truncate">
      {{ data.label }}
    </p>
    <p v-if="data.subtitle" class="text-[10px] text-on-surface-variant mt-0.5 truncate">
      {{ data.subtitle }}
    </p>
    <Handle
      v-for="(hid, i) in sourceHandleIds"
      :id="hid ?? undefined"
      :key="hid ?? 'out'"
      type="source"
      :position="Position.Right"
      class="!w-2 !h-2 !border-0"
      :style="{
        backgroundColor: accent,
        top: sourceHandleTop(i, sourceHandleIds.length),
      }"
    />
  </div>
</template>
