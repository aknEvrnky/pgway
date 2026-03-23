<script setup lang="ts">
withDefaults(defineProps<{
  label: string
  variant?: 'primary' | 'secondary'
}>(), {
  variant: 'secondary',
})
</script>

<template>
  <button
    v-if="variant === 'primary'"
    class="w-full flex items-center justify-between px-4 py-3 bg-gradient-to-r from-primary to-primary-container text-on-primary-container rounded-lg font-bold text-xs transition-transform active:scale-95 group"
  >
    <span class="flex items-center gap-2">
      <slot name="icon" />
      {{ label }}
    </span>
    <svg class="w-4 h-4 group-hover:translate-x-1 transition-transform" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <path d="M5 12h14m-7-7l7 7-7 7" />
    </svg>
  </button>
  <button
    v-else
    class="w-full flex items-center gap-3 px-4 py-3 bg-surface-container-highest hover:bg-surface-bright text-white rounded-lg font-medium text-xs transition-all active:scale-95 ghost-border"
  >
    <slot name="icon" />
    {{ label }}
  </button>
</template>
