<script setup lang="ts">
defineProps<{
  open: boolean
  name: string
  error?: string | null
  loading?: boolean
}>()

const emit = defineEmits<{
  close: []
  confirm: []
}>()
</script>

<template>
  <Teleport to="body">
    <div
      v-if="open"
      class="fixed inset-0 z-[110] flex items-center justify-center p-4"
      role="dialog"
      aria-modal="true"
      aria-label="Delete router"
    >
      <div class="absolute inset-0 bg-black/60 backdrop-blur-sm" @click="!loading && emit('close')" />
      <div class="relative w-full max-w-md glass-panel rounded-xl border border-outline-variant/20 shadow-2xl p-6 space-y-4">
        <h2 class="text-lg font-black text-on-surface tracking-tight">
          Delete router
        </h2>
        <p class="text-sm text-on-surface-variant">
          Permanently delete <span class="font-mono text-primary">{{ name }}</span>? This cannot be undone.
          Flows that reference this router will block deletion.
        </p>
        <p v-if="error" class="text-error text-sm" role="alert">
          {{ error }}
        </p>
        <div class="flex justify-end gap-3 pt-2">
          <button
            type="button"
            class="text-sm font-bold text-on-surface-variant hover:text-on-surface px-4 py-2"
            :disabled="loading"
            @click="emit('close')"
          >
            Cancel
          </button>
          <button
            type="button"
            class="bg-error-container text-error font-bold px-6 py-2.5 rounded-xl hover:opacity-90 active:scale-[0.98] transition-all disabled:opacity-60"
            :disabled="loading"
            @click="emit('confirm')"
          >
            {{ loading ? 'Deleting…' : 'Delete' }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>
