<template>
  <div class="dark min-h-screen bg-surface text-on-surface font-sans antialiased selection:bg-primary/30 selection:text-primary overflow-hidden">
    <div class="fixed inset-0 overflow-hidden pointer-events-none">
      <div class="absolute -top-[10%] -left-[10%] w-[40%] h-[40%] bg-primary/5 rounded-full blur-[120px]" />
      <div class="absolute -bottom-[10%] -right-[10%] w-[50%] h-[50%] bg-primary/10 rounded-full blur-[150px]" />
    </div>
    <slot />
  </div>
</template>
