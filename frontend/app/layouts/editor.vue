<script setup lang="ts">
const { width } = useSidebar()
</script>

<template>
  <div class="dark min-h-screen bg-surface text-on-surface">
    <OrganismsTheSidebar />
    <OrganismsTheTopBar />
    <main
      class="transition-all duration-300 h-[calc(100vh-4rem)] overflow-hidden"
      :style="{ marginLeft: width }"
    >
      <slot />
    </main>
  </div>
</template>
