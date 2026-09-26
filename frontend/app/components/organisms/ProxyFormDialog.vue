<script setup lang="ts">
import type { Proxy, ProxyApplyRequest, ProxyProtocol } from '~/types'

const props = defineProps<{
  open: boolean
  mode: 'create' | 'edit'
  initial?: Proxy | null
}>()

const emit = defineEmits<{
  close: []
  saved: []
}>()

const { apply } = useProxies()

type InputMode = 'url' | 'manual'
type LabelRow = { key: string; value: string }

const inputMode = ref<InputMode>('manual')
const name = ref('')
const url = ref('')
const protocol = ref<ProxyProtocol>('http')
const host = ref('')
const port = ref<number | null>(1080)
const username = ref('')
const password = ref('')
const labels = ref<LabelRow[]>([])
const formError = ref('')
const saving = ref(false)

const title = computed(() => (props.mode === 'edit' ? 'Edit Proxy' : 'Add New Proxy'))
const nameReadonly = computed(() => props.mode === 'edit')
// URL mode is create-only; edit always uses discrete fields.
const showInputModeToggle = computed(() => props.mode === 'create')
const isUrlMode = computed(() => props.mode === 'create' && inputMode.value === 'url')

watch(
  () => [props.open, props.initial, props.mode] as const,
  () => {
    if (!props.open) return
    formError.value = ''
    saving.value = false
    if (props.mode === 'edit' && props.initial) {
      const p = props.initial
      inputMode.value = 'manual'
      name.value = p.id
      url.value = ''
      protocol.value = p.protocol
      host.value = p.host
      port.value = p.port
      username.value = p.auth?.user || ''
      password.value = ''
      labels.value = Object.entries(p.labels || {}).map(([key, value]) => ({ key, value }))
      if (labels.value.length === 0) {
        labels.value = [{ key: '', value: '' }]
      }
    }
    else {
      inputMode.value = 'manual'
      name.value = ''
      url.value = ''
      protocol.value = 'http'
      host.value = ''
      port.value = 1080
      username.value = ''
      password.value = ''
      labels.value = [{ key: '', value: '' }]
    }
  },
  { immediate: true },
)

function addLabel() {
  labels.value.push({ key: '', value: '' })
}

function removeLabel(i: number) {
  labels.value.splice(i, 1)
  if (labels.value.length === 0) {
    labels.value.push({ key: '', value: '' })
  }
}

function labelsMap(): Record<string, string> | undefined {
  const out: Record<string, string> = {}
  for (const row of labels.value) {
    const k = row.key.trim()
    if (!k) continue
    out[k] = row.value
  }
  return Object.keys(out).length ? out : undefined
}

function buildBody(): ProxyApplyRequest {
  const metaName = name.value.trim()
  if (!metaName) {
    throw new Error('name is required')
  }

  const body: ProxyApplyRequest = {
    metadata: {
      name: metaName,
      labels: labelsMap(),
    },
    spec: {},
  }

  if (isUrlMode.value) {
    const u = url.value.trim()
    if (!u) throw new Error('proxy URL is required')
    body.spec.url = u
    return body
  }

  if (!host.value.trim()) throw new Error('host is required')
  if (!port.value || port.value <= 0) throw new Error('port is required')

  body.spec.protocol = protocol.value
  body.spec.host = host.value.trim()
  body.spec.port = port.value

  const user = username.value.trim()
  const pass = password.value

  if (props.mode === 'edit') {
    // Omit auth entirely to preserve existing credentials when password left blank
    // and username unchanged/empty-with-existing.
    if (pass) {
      body.spec.auth = { user, pass }
    }
    else if (user && user !== (props.initial?.auth?.user || '')) {
      throw new Error('password is required when changing username')
    }
    // else omit auth → CP preserves
  }
  else if (user || pass) {
    body.spec.auth = { user, pass }
  }

  return body
}

async function onSave() {
  formError.value = ''
  saving.value = true
  try {
    const body = buildBody()
    await apply(body)
    useToast().success(props.mode === 'edit' ? 'Proxy updated' : 'Proxy created')
    emit('saved')
    emit('close')
  }
  catch (e: unknown) {
    const msg = e instanceof Error ? e.message : 'save failed'
    formError.value = msg
    useToast().error(msg)
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
      class="fixed inset-0 z-[100] flex items-center justify-center p-4"
      role="dialog"
      aria-modal="true"
      :aria-label="title"
    >
      <div class="absolute inset-0 bg-black/60 backdrop-blur-sm" @click="onBackdrop" />
      <div class="relative w-full max-w-lg glass-panel rounded-xl border border-outline-variant/20 shadow-2xl p-6 space-y-5">
        <div>
          <h2 class="text-xl font-black tracking-tight text-on-surface">
            {{ title }}
          </h2>
          <p class="text-sm text-on-surface-variant mt-1">
            Configure an upstream endpoint for the gateway.
          </p>
        </div>

        <div v-if="showInputModeToggle">
          <label class="text-[10px] font-bold tracking-widest text-on-surface-variant uppercase mb-2 block">
            Input Mode
          </label>
          <div class="grid grid-cols-2 gap-1 p-1 rounded-lg bg-surface-container-lowest border border-outline-variant/15">
            <button
              type="button"
              class="py-2 text-xs font-bold rounded-md transition-colors"
              :class="inputMode === 'url' ? 'bg-surface-container-highest text-primary shadow-sm' : 'text-on-surface-variant hover:text-on-surface'"
              @click="inputMode = 'url'"
            >
              URL Mode
            </button>
            <button
              type="button"
              class="py-2 text-xs font-bold rounded-md transition-colors"
              :class="inputMode === 'manual' ? 'bg-surface-container-highest text-primary shadow-sm' : 'text-on-surface-variant hover:text-on-surface'"
              @click="inputMode = 'manual'"
            >
              Manual Mode
            </button>
          </div>
        </div>

        <div class="space-y-2">
          <label class="text-xs font-bold text-on-surface" for="proxy-name">Name</label>
          <input
            id="proxy-name"
            v-model="name"
            type="text"
            :readonly="nameReadonly"
            required
            placeholder="e.g. us-west-socks"
            class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-xl px-4 py-3 text-sm font-mono text-on-surface focus:border-primary focus:ring-4 focus:ring-primary/20 outline-none transition-all read-only:opacity-70"
          >
        </div>

        <div v-if="isUrlMode" class="space-y-2">
          <div class="flex items-baseline justify-between gap-2">
            <label class="text-sm font-bold text-on-surface" for="proxy-url">Proxy URL</label>
            <span class="text-[10px] text-on-surface-variant font-mono">protocol://user:pass@host:port</span>
          </div>
          <input
            id="proxy-url"
            v-model="url"
            type="text"
            placeholder="socks5://admin:secret@127.0.0.1:1080"
            class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-xl px-4 py-3 text-sm font-mono text-on-surface focus:border-primary focus:ring-4 focus:ring-primary/20 outline-none transition-all"
          >
        </div>

        <div v-else class="space-y-4">
          <div class="grid grid-cols-2 gap-3">
            <div class="space-y-2">
              <label class="text-xs font-bold text-on-surface" for="proxy-protocol">Protocol</label>
              <select
                id="proxy-protocol"
                v-model="protocol"
                class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-xl px-4 py-3 text-sm text-on-surface focus:ring-1 focus:ring-primary/20 outline-none"
              >
                <option value="http">HTTP</option>
                <option value="https">HTTPS</option>
                <option value="socks5">SOCKS5</option>
              </select>
            </div>
            <div class="space-y-2">
              <label class="text-xs font-bold text-on-surface" for="proxy-port">Port</label>
              <input
                id="proxy-port"
                v-model.number="port"
                type="number"
                min="1"
                max="65535"
                placeholder="1080"
                class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-xl px-4 py-3 text-sm font-mono text-primary focus:ring-1 focus:ring-primary/20 outline-none"
              >
            </div>
          </div>
          <div class="space-y-2">
            <label class="text-xs font-bold text-on-surface" for="proxy-host">Host Address</label>
            <input
              id="proxy-host"
              v-model="host"
              type="text"
              placeholder="e.g. 127.0.0.1 or proxy.example.com"
              class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-xl px-4 py-3 text-sm text-on-surface focus:ring-1 focus:ring-primary/20 outline-none"
            >
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div class="space-y-2">
              <label class="text-xs font-bold text-on-surface" for="proxy-user">Username</label>
              <input
                id="proxy-user"
                v-model="username"
                type="text"
                autocomplete="off"
                placeholder="Optional"
                class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-xl px-4 py-3 text-sm text-on-surface focus:ring-1 focus:ring-primary/20 outline-none"
              >
            </div>
            <div class="space-y-2">
              <label class="text-xs font-bold text-on-surface" for="proxy-pass">Password</label>
              <input
                id="proxy-pass"
                v-model="password"
                type="password"
                autocomplete="new-password"
                :placeholder="mode === 'edit' ? 'Unchanged if empty' : 'Optional'"
                class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-xl px-4 py-3 text-sm focus:ring-1 focus:ring-primary/20 outline-none"
              >
            </div>
          </div>
        </div>

        <div class="space-y-3">
          <div class="flex items-center gap-2 text-xs font-bold text-on-surface">
            <svg class="w-4 h-4 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" aria-hidden="true">
              <path stroke-linecap="round" d="M7 7h.01M7 3h5l7 7-5 5-7-7V3z" />
              <path stroke-linecap="round" d="M3 14l3 3" />
            </svg>
            Resource Labels
          </div>
          <div v-for="(row, i) in labels" :key="i" class="flex items-center gap-2">
            <input
              v-model="row.key"
              type="text"
              placeholder="Key"
              class="flex-1 bg-surface-container-lowest border border-outline-variant/20 rounded-lg px-3 py-2 text-[11px] text-on-surface focus:ring-1 focus:ring-primary/20 outline-none"
            >
            <input
              v-model="row.value"
              type="text"
              placeholder="Value"
              class="flex-1 bg-surface-container-lowest border border-outline-variant/20 rounded-lg px-3 py-2 text-[11px] text-primary focus:ring-1 focus:ring-primary/20 outline-none"
            >
            <button
              type="button"
              class="p-1.5 text-on-surface-variant hover:text-error transition-colors"
              aria-label="Remove label"
              @click="removeLabel(i)"
            >
              <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
                <line x1="18" y1="6" x2="6" y2="18" stroke-linecap="round" />
                <line x1="6" y1="6" x2="18" y2="18" stroke-linecap="round" />
              </svg>
            </button>
          </div>
          <button
            type="button"
            class="flex items-center gap-2 text-[10px] font-bold text-primary hover:text-primary-container transition-colors"
            @click="addLabel"
          >
            <span class="text-sm leading-none">+</span>
            Add Key-Value Label
          </button>
        </div>

        <p v-if="formError" class="text-error text-sm" role="alert">
          {{ formError }}
        </p>

        <div class="flex items-center justify-end gap-3 pt-2">
          <button
            type="button"
            class="text-sm font-bold text-on-surface-variant hover:text-on-surface transition-colors px-4 py-2"
            :disabled="saving"
            @click="emit('close')"
          >
            Cancel
          </button>
          <button
            type="button"
            class="bg-gradient-to-br from-primary to-primary-container text-on-primary-container font-bold px-8 py-2.5 rounded-xl shadow-lg shadow-primary/10 hover:shadow-primary/20 active:scale-[0.98] transition-all disabled:opacity-60"
            :disabled="saving"
            @click="onSave"
          >
            {{ saving ? 'Saving…' : 'Save Proxy' }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>
