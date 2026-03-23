<script setup lang="ts">
const { width } = useSidebar()
</script>

<template>
  <div class="dark min-h-screen bg-surface text-on-surface">
    <OrganismsTheSidebar />
    <OrganismsTheTopBar />
    <main class="p-8 space-y-8 transition-all duration-300" :style="{ marginLeft: width }">
      <slot />
    </main>
    <!-- Aura background effects -->
    <div class="fixed top-0 right-0 w-[500px] h-[500px] bg-primary/5 blur-[120px] rounded-full pointer-events-none -z-10 translate-x-1/2 -translate-y-1/2" />
    <div class="fixed bottom-0 left-64 w-[300px] h-[300px] bg-primary/5 blur-[100px] rounded-full pointer-events-none -z-10" />
  </div>
</template>
