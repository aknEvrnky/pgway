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
</script>

<template>
  <div
    class="min-w-[150px] rounded-xl border px-3 py-2.5 shadow-lg text-left"
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
      type="source"
      :position="Position.Right"
      class="!w-2 !h-2 !border-0"
      :style="{ backgroundColor: accent }"
    />
  </div>
</template>
