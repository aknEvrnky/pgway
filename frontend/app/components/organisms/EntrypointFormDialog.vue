<script setup lang="ts">
import type { Entrypoint, EntrypointApplyRequest } from '~/types'

const props = defineProps<{
  open: boolean
  mode: 'create' | 'edit'
  initial?: Entrypoint | null
  /** Prefill flow_id when creating from the flow editor. */
  defaultFlowId?: string
}>()

const emit = defineEmits<{
  close: []
  saved: [ep: Entrypoint]
}>()

const { apply } = useEntrypoints()

const name = ref('')
const title = ref('')
const protocol = ref('http')
const host = ref('0.0.0.0')
const port = ref(8080)
const flowId = ref('')
const formError = ref('')
const saving = ref(false)

const dialogTitle = computed(() => (props.mode === 'edit' ? 'Edit Entrypoint' : 'Add Entrypoint'))
const nameReadonly = computed(() => props.mode === 'edit')

watch(
  () => [props.open, props.initial, props.mode, props.defaultFlowId] as const,
  () => {
    if (!props.open) return
    formError.value = ''
    saving.value = false
    if (props.mode === 'edit' && props.initial) {
      name.value = props.initial.id
      title.value = props.initial.title || ''
      protocol.value = props.initial.protocol || 'http'
      host.value = props.initial.host
      port.value = props.initial.port
      flowId.value = props.initial.flow_id
    }
    else {
      name.value = ''
      title.value = ''
      protocol.value = 'http'
      host.value = '0.0.0.0'
      port.value = 8080
      flowId.value = props.defaultFlowId || ''
    }
  },
  { immediate: true },
)

function buildBody(): EntrypointApplyRequest {
  const metaName = name.value.trim()
  if (!metaName) throw new Error('name is required')
  const h = host.value.trim()
  if (!h) throw new Error('host is required')
  const p = Number(port.value)
  if (!Number.isFinite(p) || p < 1 || p > 65535) throw new Error('port must be 1–65535')
  const fid = flowId.value.trim()
  if (!fid) throw new Error('flow_id is required')

  return {
    metadata: { name: metaName },
    spec: {
      title: title.value.trim() || undefined,
      protocol: protocol.value,
      host: h,
      port: Math.floor(p),
      flow_id: fid,
    },
  }
}

async function onSave() {
  formError.value = ''
  saving.value = true
  try {
    const ep = await apply(buildBody())
    useAppToast().success(props.mode === 'edit' ? 'Entrypoint updated' : 'Entrypoint created')
    emit('saved', ep)
    emit('close')
  }
  catch (e: unknown) {
    const msg = e instanceof Error ? e.message : 'save failed'
    formError.value = msg
    useAppToast().error(msg)
  }
  finally {
    saving.value = false
  }
}

function onBackdrop() {
  if (!saving.value) emit('close')
}
</script>

<template>
  <Teleport to="body">
    <div
      v-if="open"
      class="fixed inset-0 z-[100] flex items-center justify-center p-4 overflow-hidden"
      role="dialog"
      aria-modal="true"
      :aria-label="dialogTitle"
    >
      <div class="absolute inset-0 bg-black/60 backdrop-blur-sm" @click="onBackdrop" />
      <div class="relative w-full max-w-lg max-h-[min(90vh,640px)] flex flex-col glass-panel rounded-xl border border-outline-variant/20 shadow-2xl overflow-hidden">
        <div class="shrink-0 px-6 pt-6 pb-3 border-b border-outline-variant/10">
          <h2 class="text-xl font-black tracking-tight text-on-surface">
            {{ dialogTitle }}
          </h2>
          <p class="text-sm text-on-surface-variant mt-1">
            Listen address that starts this flow.
          </p>
        </div>

        <div class="flex-1 min-h-0 overflow-y-auto overscroll-contain px-6 py-4 space-y-4">
          <div class="space-y-2">
            <label class="text-xs font-bold text-on-surface" for="ep-name">Name</label>
            <input
              id="ep-name"
              v-model="name"
              type="text"
              :readonly="nameReadonly"
              required
              placeholder="e.g. edge-http"
              class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-xl px-4 py-3 text-sm font-mono text-on-surface focus:border-primary focus:ring-4 focus:ring-primary/20 outline-none transition-all read-only:opacity-70"
            >
          </div>

          <div class="space-y-2">
            <label class="text-xs font-bold text-on-surface" for="ep-title">Title</label>
            <input
              id="ep-title"
              v-model="title"
              type="text"
              placeholder="optional"
              class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-xl px-4 py-3 text-sm text-on-surface outline-none focus:ring-1 focus:ring-primary/20"
            >
          </div>

          <div class="grid grid-cols-2 gap-3">
            <div class="space-y-2">
              <label class="text-xs font-bold text-on-surface" for="ep-protocol">Protocol</label>
              <select
                id="ep-protocol"
                v-model="protocol"
                class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-xl px-4 py-3 text-sm text-on-surface outline-none focus:ring-1 focus:ring-primary/20"
              >
                <option value="http">
                  HTTP
                </option>
              </select>
            </div>
            <div class="space-y-2">
              <label class="text-xs font-bold text-on-surface" for="ep-port">Port</label>
              <input
                id="ep-port"
                v-model.number="port"
                type="number"
                min="1"
                max="65535"
                class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-xl px-4 py-3 text-sm font-mono text-on-surface outline-none focus:ring-1 focus:ring-primary/20"
              >
            </div>
          </div>

          <div class="space-y-2">
            <label class="text-xs font-bold text-on-surface" for="ep-host">Host</label>
            <input
              id="ep-host"
              v-model="host"
              type="text"
              placeholder="0.0.0.0"
              class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-xl px-4 py-3 text-sm font-mono text-on-surface outline-none focus:ring-1 focus:ring-primary/20"
            >
          </div>

          <div class="space-y-2">
            <label class="text-xs font-bold text-on-surface" for="ep-flow">Flow ID</label>
            <input
              id="ep-flow"
              v-model="flowId"
              type="text"
              required
              placeholder="flow resource id"
              class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-xl px-4 py-3 text-sm font-mono text-on-surface outline-none focus:ring-1 focus:ring-primary/20"
            >
          </div>

          <p v-if="formError" class="text-error text-sm" role="alert">
            {{ formError }}
          </p>
        </div>

        <div class="shrink-0 flex items-center justify-end gap-3 px-6 py-4 border-t border-outline-variant/10 bg-surface-container-low/40">
          <button
            type="button"
            class="text-sm font-bold text-on-surface-variant hover:text-on-surface px-4 py-2"
            :disabled="saving"
            @click="emit('close')"
          >
            Cancel
          </button>
          <button
            type="button"
            class="bg-gradient-to-br from-primary to-primary-container text-on-primary-container font-bold px-8 py-2.5 rounded-xl shadow-lg shadow-primary/10 disabled:opacity-60"
            :disabled="saving"
            @click="onSave"
          >
            {{ saving ? 'Saving…' : 'Save Entrypoint' }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>
