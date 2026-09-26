<script setup lang="ts">
const props = defineProps<{
  open: boolean
  name: string
  host?: string
  port?: number
  protocol?: string
  flowId?: string
  error?: string | null
  loading?: boolean
}>()

const emit = defineEmits<{
  close: []
  confirm: []
}>()

const listenAddress = computed(() => {
  const proto = props.protocol || 'http'
  const h = props.host || '—'
  const p = props.port != null ? props.port : '—'
  return `${proto}://${h}:${p}`
})

const socketLabel = computed(() => {
  if (!props.host || props.port == null) return listenAddress.value
  return `${props.host}:${props.port}`
})
</script>

<template>
  <Teleport to="body">
    <div
      v-if="open"
      class="fixed inset-0 z-[110] flex items-center justify-center p-4"
      role="dialog"
      aria-modal="true"
      aria-label="Delete entrypoint"
    >
      <div class="absolute inset-0 bg-black/60 backdrop-blur-sm" @click="!loading && emit('close')" />
      <div class="relative w-full max-w-md glass-panel rounded-xl border border-outline-variant/20 shadow-2xl overflow-hidden">
        <div class="p-6 space-y-4">
          <div class="flex items-start gap-3">
            <div class="w-10 h-10 rounded-lg bg-error-container/80 flex items-center justify-center shrink-0">
              <svg class="w-5 h-5 text-error" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
                <path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z" stroke-linejoin="round" />
                <line x1="12" y1="9" x2="12" y2="13" stroke-linecap="round" />
                <line x1="12" y1="17" x2="12.01" y2="17" stroke-linecap="round" />
              </svg>
            </div>
            <div class="min-w-0">
              <p class="text-[10px] font-black uppercase tracking-widest text-error/80 mb-1">
                Destructive protocol event
              </p>
              <h2 class="text-lg font-black text-on-surface tracking-tight">
                Delete Entrypoint
              </h2>
            </div>
          </div>

          <div class="flex items-center justify-between gap-3 rounded-lg bg-surface-container-lowest/80 border border-outline-variant/15 px-3.5 py-2.5">
            <div class="flex items-center gap-2 min-w-0">
              <span class="w-2 h-2 rounded-full bg-error shrink-0" />
              <span class="font-mono text-sm font-semibold text-on-surface truncate">{{ name }}</span>
            </div>
            <span class="font-mono text-[11px] text-on-surface-variant bg-surface-container-highest/60 px-2 py-1 rounded shrink-0">
              {{ listenAddress }}
            </span>
          </div>

          <div class="space-y-2 text-sm text-on-surface-variant">
            <p class="text-on-surface font-medium">
              Are you sure you want to permanently delete this entrypoint?
            </p>
            <p>
              This immediately closes the listen socket on
              <span class="font-mono text-on-surface">{{ socketLabel }}</span>
              and stops accepting client traffic.
            </p>
            <p v-if="flowId">
              The bound Flow
              (<span class="font-mono text-primary text-[12px]">{{ flowId }}</span>)
              and downstream routers will remain intact.
            </p>
          </div>

          <div class="rounded-lg bg-error-container/20 border-l-2 border-error px-3.5 py-3 flex items-start gap-2.5">
            <svg class="w-4 h-4 text-error shrink-0 mt-0.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
              <circle cx="12" cy="12" r="10" />
              <line x1="12" y1="8" x2="12" y2="12" stroke-linecap="round" />
              <circle cx="12" cy="16" r="0.5" fill="currentColor" />
            </svg>
            <p class="text-xs text-error leading-relaxed">
              Active client connections will terminate immediately. This action cannot be undone.
            </p>
          </div>

          <p v-if="error" class="text-error text-sm" role="alert">
            {{ error }}
          </p>
        </div>

        <div class="flex justify-end gap-3 px-6 py-4 border-t border-outline-variant/10 bg-surface-container-low/40">
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
            class="bg-error-container text-error font-bold px-5 py-2.5 rounded-xl hover:opacity-90 active:scale-[0.98] transition-all disabled:opacity-60 flex items-center gap-2"
            :disabled="loading"
            @click="emit('confirm')"
          >
            <svg v-if="!loading" class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" aria-hidden="true">
              <polyline points="3 6 5 6 21 6" />
              <path d="M19 6l-1 14a2 2 0 01-2 2H8a2 2 0 01-2-2L5 6" />
              <path d="M10 11v6M14 11v6" />
              <path d="M9 6V4a1 1 0 011-1h4a1 1 0 011 1v2" />
            </svg>
            {{ loading ? 'Deleting…' : 'Confirm Delete' }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>
