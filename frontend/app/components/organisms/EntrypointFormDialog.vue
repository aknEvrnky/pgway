<script setup lang="ts">
import type { Entrypoint, EntrypointApplyRequest, Flow } from '~/types'

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
const { apiFetch } = useApi()

const name = ref('')
const title = ref('')
const protocol = ref('http')
const host = ref('0.0.0.0')
const port = ref(8080)
const flowId = ref('')
const formError = ref('')
const saving = ref(false)
const flows = ref<Flow[]>([])
const flowsLoading = ref(false)
const listenClash = ref<string | null>(null)
let listenCheckTimer: ReturnType<typeof setTimeout> | null = null

const dialogTitle = computed(() => (props.mode === 'edit' ? 'Edit Entrypoint' : 'Add Entrypoint'))
const modeBadge = computed(() => (props.mode === 'edit' ? 'Edit' : 'Create'))
const nameReadonly = computed(() => props.mode === 'edit')

const listenPreview = computed(() => {
  const h = host.value.trim() || '0.0.0.0'
  const p = Number(port.value)
  const portPart = Number.isFinite(p) && p > 0 ? p : '…'
  return `${protocol.value}://${h}:${portPart}`
})

const flowPreviewLabel = computed(() => {
  const id = flowId.value.trim()
  if (!id) return 'Flow'
  return `Flow (${id})`
})

/** Ensure the selected / prefilled flow appears even if not in the first page. */
const flowOptions = computed(() => {
  const list = [...flows.value]
  const id = flowId.value.trim()
  if (id && !list.some(f => f.id === id)) {
    list.unshift({ id } as Flow)
  }
  return list
})

watch(
  () => [props.open, props.initial, props.mode, props.defaultFlowId] as const,
  async () => {
    if (!props.open) return
    formError.value = ''
    listenClash.value = null
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
    await loadFlows()
    await checkListenClash()
  },
  { immediate: true },
)

watch(
  () => [host.value, port.value, protocol.value, name.value] as const,
  () => {
    if (!props.open) return
    if (listenCheckTimer) clearTimeout(listenCheckTimer)
    listenCheckTimer = setTimeout(() => {
      checkListenClash().catch(() => {})
    }, 300)
  },
)

async function loadFlows() {
  flowsLoading.value = true
  try {
    const res = await apiFetch<{ items: Flow[] }>('/api/v1/flows?page_size=100')
    flows.value = res.items || []
  }
  catch {
    flows.value = []
  }
  finally {
    flowsLoading.value = false
  }
}

/**
 * CP does not reject duplicate host:port; DP bind fails later.
 * Warn in the form when another entrypoint already uses the same listen tuple.
 */
async function checkListenClash() {
  listenClash.value = null
  const h = host.value.trim()
  const p = Number(port.value)
  if (!h || !Number.isFinite(p) || p < 1 || p > 65535) return

  try {
    const res = await apiFetch<{ items: Entrypoint[] }>(
      `/api/v1/entrypoints?host=${encodeURIComponent(h)}&page_size=100`,
    )
    const self = props.mode === 'edit' ? props.initial?.id : undefined
    const clash = (res.items || []).find(ep =>
      ep.id !== self
      && ep.host === h
      && ep.port === Math.floor(p)
      && (ep.protocol || 'http') === protocol.value,
    )
    if (clash) {
      listenClash.value = `Listen ${protocol.value}://${h}:${Math.floor(p)} is already used by entrypoint “${clash.id}” (flow ${clash.flow_id}). Control Plane allows this, but the data plane will fail to bind the socket.`
    }
  }
  catch {
    // non-blocking
  }
}

function setHost(value: string) {
  host.value = value
}

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
  await checkListenClash()
  if (listenClash.value) {
    formError.value = listenClash.value
    useAppToast().error('Listen address already in use')
    return
  }
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

function flowOptionLabel(f: Flow) {
  return f.title ? `${f.id} — ${f.title}` : f.id
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
      <div class="relative w-full max-w-[540px] max-h-[min(90vh,720px)] flex flex-col glass-panel rounded-xl border border-outline-variant/20 shadow-2xl overflow-hidden">
        <div class="h-0.5 w-full bg-gradient-to-r from-[#4ade80] via-primary to-primary-container shrink-0" />

        <div class="shrink-0 px-6 pt-5 pb-3 border-b border-outline-variant/10 flex items-start justify-between gap-3">
          <div>
            <div class="flex items-center gap-2.5 flex-wrap">
              <h2 class="text-xl font-black tracking-tight text-on-surface">
                {{ dialogTitle }}
              </h2>
              <span
                class="text-[10px] font-black uppercase tracking-wider px-2 py-0.5 rounded"
                :class="mode === 'edit'
                  ? 'bg-tertiary/15 text-tertiary'
                  : 'bg-primary/15 text-primary'"
              >
                {{ modeBadge }}
              </span>
            </div>
            <p class="text-sm text-on-surface-variant mt-1">
              Listen address that receives client requests and starts a flow pipeline.
            </p>
          </div>
          <button
            type="button"
            class="p-1.5 rounded-lg text-outline hover:text-on-surface hover:bg-surface-container transition-colors"
            aria-label="Close"
            :disabled="saving"
            @click="emit('close')"
          >
            <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
              <line x1="18" y1="6" x2="6" y2="18" stroke-linecap="round" />
              <line x1="6" y1="6" x2="18" y2="18" stroke-linecap="round" />
            </svg>
          </button>
        </div>

        <div class="flex-1 min-h-0 overflow-y-auto overscroll-contain px-6 py-4 space-y-4">
          <div class="space-y-1.5">
            <div class="flex items-center justify-between gap-2">
              <label class="text-xs font-bold text-on-surface" for="ep-name">
                Name <span class="text-primary">*</span>
                <span class="text-on-surface-variant font-normal">(Resource ID)</span>
              </label>
              <span class="text-[10px] font-mono text-on-surface-variant/80">schema: string</span>
            </div>
            <input
              id="ep-name"
              v-model="name"
              type="text"
              :readonly="nameReadonly"
              required
              placeholder="public-http"
              spellcheck="false"
              class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-lg px-3.5 py-2.5 text-sm font-mono text-primary focus:border-primary focus:ring-4 focus:ring-primary/20 outline-none transition-all read-only:opacity-70"
            >
            <p class="text-[11px] text-on-surface-variant">
              metadata.name — RFC 1123 format. Immutable after creation.
            </p>
          </div>

          <div class="space-y-1.5">
            <div class="flex items-center justify-between gap-2">
              <label class="text-xs font-bold text-on-surface" for="ep-title">
                Title <span class="text-on-surface-variant font-normal">(Optional)</span>
              </label>
              <span class="text-[10px] font-mono text-on-surface-variant/80">display label</span>
            </div>
            <input
              id="ep-title"
              v-model="title"
              type="text"
              placeholder="e.g. Public Edge HTTP Ingress"
              class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-lg px-3.5 py-2.5 text-sm text-on-surface outline-none focus:ring-1 focus:ring-primary/20"
            >
            <p class="text-[11px] text-on-surface-variant">
              Human-readable label for dashboard display.
            </p>
          </div>

          <div class="grid grid-cols-2 gap-3">
            <div class="space-y-1.5">
              <label class="text-xs font-bold text-on-surface" for="ep-protocol">
                Protocol <span class="text-primary">*</span>
              </label>
              <div class="relative">
                <select
                  id="ep-protocol"
                  v-model="protocol"
                  class="w-full appearance-none bg-surface-container-lowest border border-outline-variant/20 rounded-lg pl-3.5 pr-9 py-2.5 text-sm text-on-surface outline-none focus:ring-1 focus:ring-primary/20"
                >
                  <option value="http">
                    HTTP · http/1.1
                  </option>
                </select>
                <svg class="w-4 h-4 absolute right-3 top-1/2 -translate-y-1/2 text-outline pointer-events-none" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
                  <path d="M7 10l5 5 5-5" stroke-linecap="round" stroke-linejoin="round" />
                </svg>
              </div>
              <p class="text-[11px] text-on-surface-variant">
                Only HTTP supported
              </p>
            </div>
            <div class="space-y-1.5">
              <div class="flex items-center justify-between gap-2">
                <label class="text-xs font-bold text-on-surface" for="ep-port">
                  Port <span class="text-primary">*</span>
                </label>
                <span class="text-[10px] font-mono text-on-surface-variant/80">1–65535 · TCP</span>
              </div>
              <input
                id="ep-port"
                v-model.number="port"
                type="number"
                min="1"
                max="65535"
                class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-lg px-3.5 py-2.5 text-sm font-mono text-on-surface outline-none focus:ring-1 focus:ring-primary/20"
              >
              <p class="text-[11px] text-on-surface-variant">
                Standard high port listener
              </p>
            </div>
          </div>

          <div class="space-y-1.5">
            <div class="flex items-center justify-between gap-2">
              <label class="text-xs font-bold text-on-surface" for="ep-host">
                Host <span class="text-primary">*</span>
                <span class="text-on-surface-variant font-normal">(Listen Host)</span>
              </label>
              <div class="flex items-center gap-2 text-[10px] font-mono">
                <button
                  type="button"
                  class="text-primary hover:underline"
                  :class="host.trim() === '0.0.0.0' ? 'underline' : 'text-on-surface-variant hover:text-primary'"
                  @click="setHost('0.0.0.0')"
                >
                  all
                </button>
                <span class="text-outline-variant/40">|</span>
                <button
                  type="button"
                  class="hover:underline"
                  :class="host.trim() === '127.0.0.1' ? 'text-primary underline' : 'text-on-surface-variant hover:text-on-surface'"
                  @click="setHost('127.0.0.1')"
                >
                  local
                </button>
              </div>
            </div>
            <input
              id="ep-host"
              v-model="host"
              type="text"
              placeholder="0.0.0.0 or 127.0.0.1"
              spellcheck="false"
              class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-lg px-3.5 py-2.5 text-sm font-mono text-on-surface outline-none focus:ring-1 focus:ring-primary/20"
            >
            <p class="text-[11px] text-on-surface-variant">
              Network interface to bind socket (use <code class="text-primary font-mono text-[10px]">0.0.0.0</code> for all interfaces).
            </p>
          </div>

          <div class="space-y-1.5">
            <div class="flex items-center justify-between gap-2">
              <label class="text-xs font-bold text-on-surface" for="ep-flow">
                Flow ID <span class="text-primary">*</span>
                <span class="text-on-surface-variant font-normal">(Target Pipeline)</span>
              </label>
              <span class="px-1.5 py-0.5 rounded text-[10px] font-mono bg-tertiary/15 text-tertiary">
                Required
              </span>
            </div>
            <div class="relative">
              <select
                id="ep-flow"
                v-model="flowId"
                required
                class="w-full appearance-none bg-surface-container-lowest border border-outline-variant/20 rounded-lg pl-3.5 pr-9 py-2.5 text-sm font-mono text-on-surface outline-none focus:ring-1 focus:ring-primary/20 cursor-pointer"
              >
                <option disabled value="">
                  {{ flowsLoading ? 'Loading flows…' : 'Select a flow…' }}
                </option>
                <option
                  v-for="f in flowOptions"
                  :key="f.id"
                  :value="f.id"
                >
                  {{ flowOptionLabel(f) }}
                </option>
              </select>
              <svg class="w-4 h-4 absolute right-3 top-1/2 -translate-y-1/2 text-outline pointer-events-none" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
                <path d="M7 10l5 5 5-5" stroke-linecap="round" stroke-linejoin="round" />
              </svg>
            </div>
            <p class="text-[11px] text-on-surface-variant">
              Request traffic is dispatched directly to this Flow pipeline.
            </p>
          </div>

          <div class="p-3.5 rounded-lg bg-surface-container-lowest/90 flex items-start gap-3 border border-outline-variant/10">
            <div class="w-6 h-6 rounded bg-primary/15 shrink-0 flex items-center justify-center mt-0.5">
              <svg class="w-3.5 h-3.5 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
                <circle cx="12" cy="12" r="3" />
                <path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4" stroke-linecap="round" />
              </svg>
            </div>
            <div class="space-y-1 min-w-0">
              <div class="flex items-center gap-2">
                <span class="text-[11px] font-mono font-semibold text-primary uppercase tracking-wide">Socket Preview</span>
                <span class="w-1.5 h-1.5 rounded-full bg-[#4ade80] animate-pulse" />
              </div>
              <p class="text-xs text-on-surface-variant font-mono leading-normal">
                Traffic binding:
                <span class="text-on-surface font-semibold">Entrypoint</span>
                →
                <span class="text-primary font-semibold">{{ flowPreviewLabel }}</span>.
                <br>
                Listen address
                <span class="text-primary-container underline decoration-primary-container/40 underline-offset-2">{{ listenPreview }}</span>
                will open immediately upon save.
              </p>
            </div>
          </div>

          <div
            v-if="listenClash"
            class="flex items-start gap-2.5 text-sm text-amber-100/95 bg-amber-500/10 border border-amber-500/25 rounded-lg px-3.5 py-2.5"
            role="status"
          >
            <svg class="w-4 h-4 mt-0.5 shrink-0 text-amber-300" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
              <path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z" />
              <line x1="12" y1="9" x2="12" y2="13" stroke-linecap="round" />
              <circle cx="12" cy="17" r="0.5" fill="currentColor" />
            </svg>
            <p class="leading-relaxed">
              {{ listenClash }}
            </p>
          </div>

          <p v-if="formError && formError !== listenClash" class="text-error text-sm" role="alert">
            {{ formError }}
          </p>
        </div>

        <div class="shrink-0 flex items-center justify-between gap-3 px-6 py-4 border-t border-outline-variant/10 bg-surface-container-low/40">
          <div class="flex items-center gap-2 min-w-0">
            <span class="w-1.5 h-1.5 rounded-full bg-outline shrink-0" />
            <span class="font-mono text-[11px] text-on-surface-variant tracking-tight truncate">
              pgway.v1.Entrypoint
            </span>
          </div>
          <div class="flex items-center gap-3 shrink-0">
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
              class="bg-gradient-to-br from-primary to-primary-container text-on-primary-container font-bold px-6 py-2.5 rounded-xl shadow-lg shadow-primary/10 disabled:opacity-60 flex items-center gap-2"
              :disabled="saving || !!listenClash"
              @click="onSave"
            >
              <svg v-if="!saving" class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
                <path d="M19 21H5a2 2 0 01-2-2V5a2 2 0 012-2h11l5 5v11a2 2 0 01-2 2z" stroke-linejoin="round" />
                <polyline points="17 21 17 13 7 13 7 21" />
                <polyline points="7 3 7 8 15 8" />
              </svg>
              {{ saving ? 'Saving…' : 'Save Entrypoint' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>
