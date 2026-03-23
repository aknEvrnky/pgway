<script setup lang="ts">
const model = defineModel<string>({ default: '1H' })
const options = ['1H', '6H', '24H']
</script>

<template>
  <div class="flex bg-surface-container-lowest p-1 rounded-lg">
    <button
      v-for="opt in options"
      :key="opt"
      class="px-3 py-1 text-[10px] font-bold rounded-md transition-colors"
      :class="model === opt ? 'text-primary bg-primary/10' : 'text-slate-500 hover:text-white'"
      @click="model = opt"
    >
      {{ opt }}
    </button>
  </div>
</template>
