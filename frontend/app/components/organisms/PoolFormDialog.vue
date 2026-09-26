<script setup lang="ts">
import type { Pool, PoolApplyRequest, PoolType } from '~/types'

const props = defineProps<{
  open: boolean
  mode: 'create' | 'edit'
  initial?: Pool | null
}>()

const emit = defineEmits<{
  close: []
  saved: []
}>()

const { apply } = usePools()

type LabelRow = { key: string; value: string }
type MemberRow = { proxy_id: string; weight: number }

const name = ref('')
const title = ref('')
const poolType = ref<PoolType>('static')
const members = ref<MemberRow[]>([{ proxy_id: '', weight: 1 }])
const allowLabels = ref<LabelRow[]>([{ key: '', value: '' }])
const labels = ref<LabelRow[]>([{ key: '', value: '' }])
const formError = ref('')
const saving = ref(false)

const dialogTitle = computed(() => (props.mode === 'edit' ? 'Edit Pool' : 'Add New Pool'))
const nameReadonly = computed(() => props.mode === 'edit')
const isStatic = computed(() => poolType.value === 'static')
const memberCount = computed(() => members.value.filter(m => m.proxy_id.trim()).length)

const tipName = 'Resource ID (metadata.name). Immutable after create. Referenced as LoadBalancer pool_id.'
const tipMembers = 'Explicit list of proxy IDs. Each proxy must already exist. Weight is used by weighted load balancers (default: 1, minimum: 1). Duplicate proxy IDs are not allowed. At least one member is required.'
const tipSelector = 'Dynamic pools resolve proxies whose labels match ALL of these allow pairs (AND). At least one allow pair is required. Static members are not allowed on dynamic pools. Weighted load balancers require a static pool; round-robin and least-bytes can use either type.'
const tipLabels = 'Metadata labels on the pool resource itself (key-value). Separate from the dynamic selector.'

watch(
  () => [props.open, props.initial, props.mode] as const,
  () => {
    if (!props.open) return
    formError.value = ''
    saving.value = false
    if (props.mode === 'edit' && props.initial) {
      const p = props.initial
      name.value = p.id
      title.value = p.title || ''
      poolType.value = p.type
      if (p.type === 'static') {
        members.value = (p.members || []).map(m => ({
          proxy_id: m.proxy_id,
          weight: m.weight >= 1 ? m.weight : 1,
        }))
        if (members.value.length === 0) {
          members.value = [{ proxy_id: '', weight: 1 }]
        }
        allowLabels.value = [{ key: '', value: '' }]
      }
      else {
        allowLabels.value = Object.entries(p.selector?.allow || {}).map(([key, value]) => ({ key, value }))
        if (allowLabels.value.length === 0) {
          allowLabels.value = [{ key: '', value: '' }]
        }
        members.value = [{ proxy_id: '', weight: 1 }]
      }
      labels.value = Object.entries(p.labels || {}).map(([key, value]) => ({ key, value }))
      if (labels.value.length === 0) {
        labels.value = [{ key: '', value: '' }]
      }
    }
    else {
      name.value = ''
      title.value = ''
      poolType.value = 'static'
      members.value = [{ proxy_id: '', weight: 1 }]
      allowLabels.value = [{ key: '', value: '' }]
      labels.value = [{ key: '', value: '' }]
    }
  },
  { immediate: true },
)

function addMember() {
  members.value.push({ proxy_id: '', weight: 1 })
}

function removeMember(i: number) {
  members.value.splice(i, 1)
  if (members.value.length === 0) {
    members.value.push({ proxy_id: '', weight: 1 })
  }
}

function addAllow() {
  allowLabels.value.push({ key: '', value: '' })
}

function removeAllow(i: number) {
  allowLabels.value.splice(i, 1)
  if (allowLabels.value.length === 0) {
    allowLabels.value.push({ key: '', value: '' })
  }
}

function addLabel() {
  labels.value.push({ key: '', value: '' })
}

function removeLabel(i: number) {
  labels.value.splice(i, 1)
  if (labels.value.length === 0) {
    labels.value.push({ key: '', value: '' })
  }
}

function labelsMap(rows: LabelRow[]): Record<string, string> | undefined {
  const out: Record<string, string> = {}
  for (const row of rows) {
    const k = row.key.trim()
    if (!k) continue
    out[k] = row.value
  }
  return Object.keys(out).length ? out : undefined
}

function buildBody(): PoolApplyRequest {
  const metaName = name.value.trim()
  if (!metaName) {
    throw new Error('name is required')
  }

  const body: PoolApplyRequest = {
    metadata: {
      name: metaName,
      labels: labelsMap(labels.value),
    },
    spec: {
      type: poolType.value,
    },
  }

  const t = title.value.trim()
  if (t) body.spec.title = t

  if (poolType.value === 'static') {
    const seen = new Set<string>()
    const list: { proxy_id: string; weight: number }[] = []
    for (const m of members.value) {
      const id = m.proxy_id.trim()
      if (!id) continue
      if (seen.has(id)) {
        throw new Error(`duplicate proxy_id ${id}`)
      }
      seen.add(id)
      const w = Number(m.weight)
      if (!Number.isFinite(w) || w < 1) {
        throw new Error(`weight for ${id} must be >= 1`)
      }
      list.push({ proxy_id: id, weight: Math.floor(w) })
    }
    if (list.length === 0) {
      throw new Error('at least one member is required for static pools')
    }
    body.spec.members = list
  }
  else {
    const allow = labelsMap(allowLabels.value)
    if (!allow) {
      throw new Error('at least one allow label is required for dynamic pools')
    }
    body.spec.selector = { allow }
  }

  return body
}

async function onSave() {
  formError.value = ''
  saving.value = true
  try {
    const body = buildBody()
    await apply(body)
    emit('saved')
    emit('close')
  }
  catch (e: unknown) {
    formError.value = e instanceof Error ? e.message : 'save failed'
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
      <div class="relative w-full max-w-lg max-h-[min(90vh,720px)] flex flex-col glass-panel rounded-xl border border-outline-variant/20 shadow-2xl overflow-hidden">
        <div class="shrink-0 px-6 pt-6 pb-3 border-b border-outline-variant/10">
          <h2 class="text-xl font-black tracking-tight text-on-surface">
            {{ dialogTitle }}
          </h2>
          <p class="text-sm text-on-surface-variant mt-1">
            Group upstream proxies with static members or a label selector.
          </p>
        </div>

        <div class="flex-1 min-h-0 overflow-y-auto overscroll-contain px-6 py-4 space-y-5">
          <div class="space-y-2">
            <div class="flex items-center gap-1.5">
              <label class="text-xs font-bold text-on-surface" for="pool-name">Name</label>
              <AtomsInfoTip :text="tipName" />
            </div>
            <input
              id="pool-name"
              v-model="name"
              type="text"
              :readonly="nameReadonly"
              required
              placeholder="e.g. am-static-pool"
              class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-xl px-4 py-3 text-sm font-mono text-on-surface focus:border-primary focus:ring-4 focus:ring-primary/20 outline-none transition-all read-only:opacity-70"
            >
          </div>

          <div class="space-y-2">
            <label class="text-xs font-bold text-on-surface" for="pool-title">
              Title <span class="font-normal text-on-surface-variant">(optional)</span>
            </label>
            <input
              id="pool-title"
              v-model="title"
              type="text"
              placeholder="e.g. Amproxy Static Weighted Pool"
              class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-xl px-4 py-3 text-sm text-on-surface focus:border-primary focus:ring-4 focus:ring-primary/20 outline-none transition-all"
            >
          </div>

          <div>
            <label class="text-[10px] font-bold tracking-widest text-on-surface-variant uppercase mb-2 block">
              Type *
            </label>
            <div class="grid grid-cols-2 gap-1 p-1 rounded-lg bg-surface-container-lowest border border-outline-variant/15">
              <button
                type="button"
                class="py-2.5 text-xs font-bold rounded-md transition-colors"
                :class="isStatic ? 'bg-surface-container-highest text-primary shadow-sm' : 'text-on-surface-variant hover:text-on-surface'"
                @click="poolType = 'static'"
              >
                Static
              </button>
              <button
                type="button"
                class="py-2.5 text-xs font-bold rounded-md transition-colors"
                :class="!isStatic ? 'bg-surface-container-highest text-primary shadow-sm' : 'text-on-surface-variant hover:text-on-surface'"
                @click="poolType = 'dynamic'"
              >
                Dynamic
              </button>
            </div>
          </div>

          <div v-if="isStatic" class="space-y-3">
            <div class="flex items-center gap-2">
              <h3 class="text-[10px] font-black uppercase tracking-widest text-on-surface-variant">
                Members
              </h3>
              <AtomsInfoTip :text="tipMembers" />
              <span class="text-[10px] font-bold uppercase tracking-wider px-2 py-0.5 rounded bg-primary/15 text-primary">
                {{ memberCount }} {{ memberCount === 1 ? 'proxy' : 'proxies' }}
              </span>
            </div>
            <div class="grid grid-cols-[1fr_5.5rem_2rem] gap-2 px-1">
              <span class="text-[10px] font-black uppercase tracking-widest text-on-surface-variant">Proxy ID</span>
              <span class="text-[10px] font-black uppercase tracking-widest text-on-surface-variant">Weight</span>
              <span />
            </div>
            <div v-for="(row, i) in members" :key="i" class="grid grid-cols-[1fr_5.5rem_2rem] gap-2 items-center">
              <input
                v-model="row.proxy_id"
                type="text"
                placeholder="proxy-id"
                class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-lg px-3 py-2 text-sm font-mono text-on-surface focus:ring-1 focus:ring-primary/20 outline-none"
              >
              <input
                v-model.number="row.weight"
                type="number"
                min="1"
                class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-lg px-3 py-2 text-sm font-mono text-primary focus:ring-1 focus:ring-primary/20 outline-none"
              >
              <button
                type="button"
                class="p-1.5 text-on-surface-variant hover:text-error transition-colors"
                aria-label="Remove member"
                @click="removeMember(i)"
              >
                <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" aria-hidden="true">
                  <polyline points="3 6 5 6 21 6" />
                  <path d="M19 6l-1 14a2 2 0 01-2 2H8a2 2 0 01-2-2L5 6" />
                </svg>
              </button>
            </div>
            <button
              type="button"
              class="flex items-center gap-2 text-[10px] font-bold text-primary hover:text-primary-container transition-colors"
              @click="addMember"
            >
              <span class="text-sm leading-none">+</span>
              Add member
            </button>
          </div>

          <div v-else class="space-y-3">
            <div class="flex items-center gap-2">
              <h3 class="text-[10px] font-black uppercase tracking-widest text-on-surface-variant">
                Label selector (allow)
              </h3>
              <AtomsInfoTip :text="tipSelector" />
            </div>
            <div v-for="(row, i) in allowLabels" :key="i" class="flex items-center gap-2">
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
                aria-label="Remove allow label"
                @click="removeAllow(i)"
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
              @click="addAllow"
            >
              <span class="text-sm leading-none">+</span>
              Add allow label
            </button>
          </div>

          <div class="space-y-3">
            <div class="flex items-center gap-2 text-xs font-bold text-on-surface">
              Resource Labels
              <span class="font-normal text-on-surface-variant">(optional)</span>
              <AtomsInfoTip :text="tipLabels" />
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
        </div>

        <div class="shrink-0 flex items-center justify-end gap-3 px-6 py-4 border-t border-outline-variant/10 bg-surface-container-low/40">
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
            {{ saving ? 'Saving…' : 'Save Pool' }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>
