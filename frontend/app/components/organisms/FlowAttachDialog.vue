<script setup lang="ts">
import type { FlowGraphKind } from '~/types'

const props = defineProps<{
  open: boolean
  kind: FlowGraphKind | null
  flowId: string
}>()

const emit = defineEmits<{
  close: []
  pick: [id: string]
  create: []
}>()

const { apiFetch } = useApi()
const { list: listEntrypoints } = useEntrypoints()
const options = ref<{ id: string, label: string }[]>([])
const loading = ref(false)
const selected = ref('')
const error = ref('')

watch(
  () => [props.open, props.kind] as const,
  async () => {
    if (!props.open || !props.kind) return
    selected.value = ''
    error.value = ''
    loading.value = true
    options.value = []
    try {
      switch (props.kind) {
        case 'entrypoint': {
          const res = await listEntrypoints({ page_size: 100 })
          options.value = (res.items || []).map(e => ({
            id: e.id,
            label: `${e.id} → ${e.flow_id || '—'}`,
          }))
          break
        }
        case 'router': {
          const res = await apiFetch<{ items: { id: string, title?: string }[] }>('/api/v1/routers?page_size=100')
          options.value = (res.items || []).map(r => ({ id: r.id, label: r.title ? `${r.id} — ${r.title}` : r.id }))
          break
        }
        case 'balancer': {
          const res = await apiFetch<{ items: { id: string, title?: string, type?: string }[] }>('/api/v1/balancers?page_size=100')
          options.value = (res.items || []).map(b => ({ id: b.id, label: `${b.id} (${b.type || '?'})` }))
          break
        }
        case 'pool': {
          const res = await apiFetch<{ items: { id: string, type?: string }[] }>('/api/v1/pools?page_size=100')
          options.value = (res.items || []).map(p => ({ id: p.id, label: `${p.id} (${p.type || '?'})` }))
          break
        }
        case 'proxy': {
          const res = await apiFetch<{ items: { id: string, host?: string, port?: number }[] }>('/api/v1/proxies?page_size=100')
          options.value = (res.items || []).map(p => ({ id: p.id, label: `${p.id} ${p.host || ''}:${p.port || ''}` }))
          break
        }
        default:
          error.value = 'This kind cannot be attached from the palette'
      }
    }
    catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'failed to load options'
    }
    finally {
      loading.value = false
    }
  },
)

function confirm() {
  if (!selected.value) {
    error.value = 'Select a resource'
    return
  }
  emit('pick', selected.value)
}
</script>

<template>
  <Teleport to="body">
    <div
      v-if="open"
      class="fixed inset-0 z-[120] flex items-center justify-center p-4"
      role="dialog"
      aria-modal="true"
      aria-label="Attach resource"
    >
      <div class="absolute inset-0 bg-black/60 backdrop-blur-sm" @click="emit('close')" />
      <div class="relative w-full max-w-md glass-panel rounded-xl border border-outline-variant/20 shadow-2xl p-6 space-y-4">
        <h2 class="text-lg font-black text-on-surface tracking-tight capitalize">
          Attach {{ kind }}
        </h2>
        <p class="text-sm text-on-surface-variant">
          Pick an existing resource to wire into flow
          <span class="font-mono text-primary">{{ flowId }}</span>,
          or create a new one.
        </p>
        <select
          v-model="selected"
          :disabled="loading"
          class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-xl px-4 py-3 text-sm font-mono outline-none"
        >
          <option value="" disabled>
            {{ loading ? 'Loading…' : 'Select…' }}
          </option>
          <option v-for="o in options" :key="o.id" :value="o.id">
            {{ o.label }}
          </option>
        </select>
        <p v-if="error" class="text-error text-sm" role="alert">
          {{ error }}
        </p>
        <div class="flex justify-between gap-3">
          <button
            type="button"
            class="text-sm font-bold text-primary hover:underline px-2 py-2"
            @click="emit('create')"
          >
            + Create new
          </button>
          <div class="flex gap-3">
            <button type="button" class="text-sm font-bold text-on-surface-variant px-4 py-2" @click="emit('close')">
              Cancel
            </button>
            <button
              type="button"
              class="bg-gradient-to-br from-primary to-primary-container text-on-primary-container font-bold px-6 py-2.5 rounded-xl"
              @click="confirm"
            >
              Attach
            </button>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>
