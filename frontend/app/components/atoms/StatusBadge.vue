<script setup lang="ts">
const props = withDefaults(defineProps<{
  text: string
  variant?: 'primary' | 'warning' | 'info'
}>(), {
  variant: 'primary',
})

const classes: Record<string, string> = {
  primary: 'bg-primary/10 text-primary',
  warning: 'bg-tertiary text-on-tertiary-container',
  info: 'bg-surface-container-highest text-on-surface-variant',
}
</script>

<template>
  <span
    class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-sm text-[9px] font-black uppercase"
    :class="classes[variant]"
  >
    <slot>{{ text }}</slot>
  </span>
</template>
