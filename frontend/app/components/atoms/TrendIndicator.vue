<script setup lang="ts">
defineProps<{
  direction: 'up' | 'down'
  percentage: number
}>()
</script>

<template>
  <span
    class="inline-flex items-center gap-0.5 px-2 py-0.5 rounded-full text-[10px] font-bold"
    :class="direction === 'up' ? 'bg-primary/10 text-primary' : 'bg-error/10 text-error'"
  >
    <svg
      class="w-3 h-3"
      :class="{ 'rotate-180': direction === 'down' }"
      viewBox="0 0 12 12"
      fill="none"
    >
      <path d="M6 2L10 7H2L6 2Z" fill="currentColor" />
    </svg>
    +{{ percentage }}%
  </span>
</template>
