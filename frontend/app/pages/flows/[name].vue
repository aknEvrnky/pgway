<script setup lang="ts">
definePageMeta({
  layout: 'editor',
})

const route = useRoute()
const flowName = computed(() => String(route.params.name || ''))
</script>

<template>
  <OrganismsFlowEditorWorkspace :flow-name="flowName" />
</template>
