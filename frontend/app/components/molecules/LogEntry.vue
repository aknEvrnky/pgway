<script setup lang="ts">
import type { LogLine } from '~/types'

defineProps<{
  log: LogLine
}>()
</script>

<template>
  <p class="font-mono text-[10px]">
    <span class="text-slate-600">{{ log.timestamp }}</span>&nbsp;
    <span
      :class="{
        'text-on-surface': log.level === 'info',
        'text-tertiary': log.level === 'warn',
        'text-error': log.level === 'error',
      }"
    >
      {{ log.message }}
    </span>
  </p>
</template>
