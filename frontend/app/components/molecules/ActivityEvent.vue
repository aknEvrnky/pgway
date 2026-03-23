<script setup lang="ts">
import type { ActivityEvent } from '~/types'

defineProps<{
  event: ActivityEvent
  isLast?: boolean
}>()

const iconMap: Record<string, { icon: string; bg: string; border: string; color: string }> = {
  success: {
    icon: 'M9 12l2 2 4-4',
    bg: 'bg-primary/10',
    border: 'border-primary/20',
    color: 'text-primary',
  },
  delete: {
    icon: 'M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6M1 7h22M8 7V5a2 2 0 012-2h4a2 2 0 012 2v2',
    bg: 'bg-surface-variant',
    border: 'border-white/5',
    color: 'text-slate-400',
  },
  warning: {
    icon: 'M12 9v4m0 4h.01',
    bg: 'bg-tertiary/10',
    border: 'border-tertiary/20',
    color: 'text-tertiary',
  },
  info: {
    icon: 'M12 8v4m0 4h.01',
    bg: 'bg-primary/10',
    border: 'border-primary/20',
    color: 'text-primary',
  },
}
</script>

<template>
  <div class="flex gap-4">
    <div class="flex flex-col items-center">
      <div
        class="w-8 h-8 rounded-full flex items-center justify-center border"
        :class="[iconMap[event.type].bg, iconMap[event.type].border]"
      >
        <svg class="w-4 h-4" :class="iconMap[event.type].color" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path v-if="event.type === 'success'" d="M9 12l2 2 4-4" />
          <path v-else-if="event.type === 'delete'" d="M6 7h12M9 7V5a1 1 0 011-1h4a1 1 0 011 1v2m2 0l-.5 9a2 2 0 01-2 2h-5a2 2 0 01-2-2L7 7" />
          <path v-else-if="event.type === 'warning'" d="M12 9v4m0 4h.01M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z" />
          <path v-else d="M12 8v4m0 4h.01" />
        </svg>
      </div>
      <div v-if="!isLast" class="w-px flex-1 bg-white/5 my-2" />
    </div>
    <div class="pb-6">
      <p class="text-sm text-white font-medium">{{ event.title }}</p>
      <p class="text-xs text-slate-500">{{ event.description }}</p>
      <p class="text-[10px] font-mono text-slate-600 mt-1">{{ event.timestamp }} &bull; {{ event.actor }}</p>
    </div>
  </div>
</template>
