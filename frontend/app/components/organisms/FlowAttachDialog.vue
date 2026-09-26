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
const options = ref<{ id: string, label: string, disabled?: boolean }[]>([])
const loading = ref(false)
const selected = ref('')
const error = ref('')
const hint = ref('')

const isEntrypoint = computed(() => props.kind === 'entrypoint')
const canAttachPick = computed(() => options.value.some(o => !o.disabled))

watch(
  () => [props.open, props.kind, props.flowId] as const,
  async () => {
    if (!props.open || !props.kind) return
    selected.value = ''
    error.value = ''
    hint.value = ''
    loading.value = true
    options.value = []
    try {
      switch (props.kind) {
        case 'entrypoint': {
          // flow_id is required on every entrypoint — picking one bound to
          // another flow would steal it. Only offer unbound (legacy) or
          // already-owned by this flow.
          const res = await listEntrypoints({ page_size: 100 })
          const flow = props.flowId
          const usable = (res.items || []).filter((e) => {
            const owner = (e.flow_id || '').trim()
            return !owner || owner === flow
          })
          const foreign = (res.items || []).length - usable.length
          options.value = usable.map(e => ({
            id: e.id,
            label: e.flow_id === flow
              ? `${e.id} (already on this flow)`
              : `${e.id} (unbound)`,
          }))
          if (!usable.length) {
            hint.value = foreign > 0
              ? `No attachable entrypoints. ${foreign} exist but belong to other flows — create a new listener for this flow instead.`
              : 'No entrypoints yet. Create a new listener bound to this flow.'
          }
          else if (foreign > 0) {
            hint.value = `${foreign} entrypoint(s) hidden — already bound to other flows (exclusive ownership).`
          }
          else {
            hint.value = 'Entrypoints are exclusive to one flow. Prefer Create new for an additional listen address.'
          }
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
  if (isEntrypoint.value && !canAttachPick.value) {
    error.value = 'Create a new entrypoint instead — none are available to attach'
    return
  }
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
          <template v-if="isEntrypoint">
            Bind a listen address to flow
            <span class="font-mono text-primary">{{ flowId }}</span>.
            An entrypoint can belong to only one flow.
          </template>
          <template v-else>
            Pick an existing resource to wire into flow
            <span class="font-mono text-primary">{{ flowId }}</span>,
            or create a new one.
          </template>
        </p>
        <p
          v-if="hint"
          class="text-[11px] text-on-surface-variant bg-surface-container-lowest/80 border border-outline-variant/15 rounded-lg px-3 py-2"
        >
          {{ hint }}
        </p>
        <select
          v-if="!isEntrypoint || canAttachPick || loading"
          v-model="selected"
          :disabled="loading || (isEntrypoint && !canAttachPick)"
          class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-xl px-4 py-3 text-sm font-mono outline-none disabled:opacity-50"
        >
          <option value="" disabled>
            {{ loading ? 'Loading…' : 'Select…' }}
          </option>
          <option v-for="o in options" :key="o.id" :value="o.id" :disabled="o.disabled">
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
              v-if="!isEntrypoint || canAttachPick"
              type="button"
              class="bg-gradient-to-br from-primary to-primary-container text-on-primary-container font-bold px-6 py-2.5 rounded-xl disabled:opacity-50"
              :disabled="isEntrypoint && !canAttachPick"
              @click="confirm"
            >
              Attach
            </button>
            <button
              v-else
              type="button"
              class="bg-gradient-to-br from-primary to-primary-container text-on-primary-container font-bold px-6 py-2.5 rounded-xl"
              @click="emit('create')"
            >
              Create Entrypoint
            </button>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>
