<script setup lang="ts">
const { toasts, dismiss } = useAppToast()

function kindClass(kind: string) {
  switch (kind) {
    case 'success':
      return 'border-tertiary/40 bg-tertiary/15 text-tertiary'
    case 'error':
      return 'border-error/40 bg-error/15 text-error'
    default:
      return 'border-primary/40 bg-primary/15 text-primary'
  }
}
</script>

<template>
  <Teleport to="body">
    <div
      class="fixed top-4 right-4 z-[300] flex flex-col gap-2 w-[min(22rem,calc(100vw-2rem))] pointer-events-none"
      aria-live="polite"
      aria-relevant="additions"
    >
      <div
        v-for="t in toasts"
        :key="t.id"
        class="pointer-events-auto flex items-start gap-3 rounded-xl border px-4 py-3 shadow-xl backdrop-blur-md text-sm"
        :class="kindClass(t.kind)"
        role="status"
      >
        <p class="flex-1 text-on-surface leading-snug">
          {{ t.message }}
        </p>
        <button
          type="button"
          class="shrink-0 text-on-surface-variant hover:text-on-surface transition-colors"
          aria-label="Dismiss"
          @click="dismiss(t.id)"
        >
          <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
            <line x1="18" y1="6" x2="6" y2="18" stroke-linecap="round" />
            <line x1="6" y1="6" x2="18" y2="18" stroke-linecap="round" />
          </svg>
        </button>
      </div>
    </div>
  </Teleport>
</template>
