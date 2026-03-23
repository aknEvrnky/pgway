<script setup lang="ts">
const props = withDefaults(defineProps<{
  status?: 'healthy' | 'degraded' | 'down' | 'active'
}>(), {
  status: 'active',
})

const colorMap: Record<string, string> = {
  healthy: '#4ade80',
  active: '#8ed5ff',
  degraded: '#ffc176',
  down: '#ffb4ab',
}

const color = computed(() => colorMap[props.status] ?? colorMap.active)
</script>

<template>
  <span
    class="pulse-dot"
    :style="{ backgroundColor: color, boxShadow: `0 0 8px ${color}` }"
  />
</template>
