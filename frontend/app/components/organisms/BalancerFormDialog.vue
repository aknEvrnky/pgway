<script setup lang="ts">
import type { BalancerApplyRequest, BalancerType, LoadBalancer, Pool, PoolListResponse } from '~/types'

const props = defineProps<{
  open: boolean
  mode: 'create' | 'edit'
  initial?: LoadBalancer | null
}>()

const emit = defineEmits<{
  close: []
  saved: []
}>()

const { apply } = useBalancers()
const { apiFetch } = useApi()

const name = ref('')
const title = ref('')
const strategy = ref<BalancerType>('round-robin')
const poolId = ref('')
const resetInterval = ref('')
const pools = ref<Pool[]>([])
const poolsLoading = ref(false)
const formError = ref('')
const saving = ref(false)

const dialogTitle = computed(() => (props.mode === 'edit' ? 'Edit Load Balancer' : 'Add New Load Balancer'))
const nameReadonly = computed(() => props.mode === 'edit')
const isLeastBytes = computed(() => strategy.value === 'least-bytes')
const isWeighted = computed(() => strategy.value === 'weighted')

const selectedPool = computed(() => pools.value.find(p => p.id === poolId.value) || null)
const selectablePools = computed(() => {
  if (isWeighted.value) {
    return pools.value.filter(p => p.type === 'static')
  }
  return pools.value
})

const tipName = 'Resource ID (metadata.name). Immutable after create. Referenced by Flow.balancer_id and Router rule targets.'
const tipPool = 'Must reference an existing Pool. Weighted type requires that pool to be static.'
const tipReset = 'Go duration string (e.g. 1m, 30s). Counters reset on this window. Default 1m when empty. Only valid for least-bytes.'

watch(
  () => [props.open, props.initial, props.mode] as const,
  async () => {
    if (!props.open) return
    formError.value = ''
    saving.value = false
    poolsLoading.value = true
    try {
      const res = await apiFetch<PoolListResponse>('/api/v1/pools?page_size=100')
      pools.value = res.items || []
    }
    catch {
      pools.value = []
    }
    finally {
      poolsLoading.value = false
    }

    if (props.mode === 'edit' && props.initial) {
      const lb = props.initial
      name.value = lb.id
      title.value = lb.title || ''
      strategy.value = lb.type
      poolId.value = lb.pool_id
      resetInterval.value = lb.reset_interval || ''
    }
    else {
      name.value = ''
      title.value = ''
      strategy.value = 'round-robin'
      poolId.value = ''
      resetInterval.value = ''
    }
  },
  { immediate: true },
)

watch(strategy, (next) => {
  if (next !== 'least-bytes') {
    resetInterval.value = ''
  }
  if (next === 'weighted' && selectedPool.value?.type === 'dynamic') {
    poolId.value = ''
  }
})

function buildBody(): BalancerApplyRequest {
  const metaName = name.value.trim()
  if (!metaName) {
    throw new Error('name is required')
  }
  const pid = poolId.value.trim()
  if (!pid) {
    throw new Error('pool_id is required')
  }

  const body: BalancerApplyRequest = {
    metadata: { name: metaName },
    spec: {
      type: strategy.value,
      pool_id: pid,
    },
  }

  const t = title.value.trim()
  if (t) body.spec.title = t

  if (strategy.value === 'least-bytes') {
    const ri = resetInterval.value.trim()
    if (ri) body.spec.reset_interval = ri
  }

  return body
}

async function onSave() {
  formError.value = ''
  saving.value = true
  try {
    const body = buildBody()
    await apply(body)
    useAppToast().success(props.mode === 'edit' ? 'Load balancer updated' : 'Load balancer created')
    emit('saved')
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

function strategyButtonClass(t: BalancerType) {
  return strategy.value === t
    ? 'bg-surface-container-highest text-primary shadow-sm'
    : 'text-on-surface-variant hover:text-on-surface'
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
            Bind a selection strategy to an existing pool.
          </p>
        </div>

        <div class="flex-1 min-h-0 overflow-y-auto overscroll-contain px-6 py-4 space-y-5">
          <div class="space-y-2">
            <div class="flex items-center gap-1.5">
              <label class="text-xs font-bold text-on-surface" for="lb-name">Name</label>
              <AtomsInfoTip :text="tipName" />
              <span class="text-[10px] font-bold uppercase tracking-wider px-1.5 py-0.5 rounded bg-surface-container-highest text-on-surface-variant border border-outline-variant/20">
                RFC 1123
              </span>
            </div>
            <input
              id="lb-name"
              v-model="name"
              type="text"
              :readonly="nameReadonly"
              required
              placeholder="e.g. edge-rr"
              class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-xl px-4 py-3 text-sm font-mono text-on-surface focus:border-primary focus:ring-4 focus:ring-primary/20 outline-none transition-all read-only:opacity-70"
            >
          </div>

          <div class="space-y-2">
            <label class="text-xs font-bold text-on-surface" for="lb-title">
              Title <span class="font-normal text-on-surface-variant">(optional)</span>
            </label>
            <input
              id="lb-title"
              v-model="title"
              type="text"
              placeholder="e.g. Edge Round Robin"
              class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-xl px-4 py-3 text-sm text-on-surface focus:border-primary focus:ring-4 focus:ring-primary/20 outline-none transition-all"
            >
          </div>

          <div>
            <label class="text-[10px] font-bold tracking-widest text-on-surface-variant uppercase mb-2 block">
              Strategy Type *
            </label>
            <div class="grid grid-cols-3 gap-1 p-1 rounded-lg bg-surface-container-lowest border border-outline-variant/15">
              <button
                type="button"
                class="py-2.5 text-xs font-bold rounded-md transition-colors"
                :class="strategyButtonClass('round-robin')"
                @click="strategy = 'round-robin'"
              >
                Round-robin
              </button>
              <button
                type="button"
                class="py-2.5 text-xs font-bold rounded-md transition-colors"
                :class="strategyButtonClass('weighted')"
                @click="strategy = 'weighted'"
              >
                Weighted
              </button>
              <button
                type="button"
                class="py-2.5 text-xs font-bold rounded-md transition-colors"
                :class="strategyButtonClass('least-bytes')"
                @click="strategy = 'least-bytes'"
              >
                Least-bytes
              </button>
            </div>
          </div>

          <div class="space-y-2">
            <div class="flex items-center justify-between gap-2">
              <div class="flex items-center gap-1.5">
                <label class="text-xs font-bold text-on-surface" for="lb-pool">Pool ID</label>
                <AtomsInfoTip :text="tipPool" />
              </div>
              <span
                v-if="selectedPool"
                class="text-[10px] font-bold uppercase tracking-wider px-2 py-0.5 rounded"
                :class="selectedPool.type === 'static'
                  ? 'bg-primary/15 text-primary'
                  : 'bg-tertiary/15 text-tertiary'"
              >
                {{ selectedPool.type }}
                <template v-if="selectedPool.type === 'static'">
                  ({{ selectedPool.members?.length || 0 }} nodes)
                </template>
              </span>
            </div>
            <select
              id="lb-pool"
              v-model="poolId"
              :disabled="poolsLoading"
              class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-xl px-4 py-3 text-sm font-mono text-on-surface focus:border-primary focus:ring-4 focus:ring-primary/20 outline-none transition-all disabled:opacity-60"
            >
              <option value="" disabled>
                {{ poolsLoading ? 'Loading pools…' : 'Select a pool' }}
              </option>
              <option
                v-for="p in selectablePools"
                :key="p.id"
                :value="p.id"
              >
                {{ p.id }}{{ p.title ? ` — ${p.title}` : '' }}
              </option>
            </select>
            <p v-if="isLeastBytes" class="text-[11px] text-on-surface-variant">
              Static or dynamic pools are allowed for least-bytes.
            </p>
          </div>

          <div
            v-if="isWeighted"
            class="flex items-start gap-2 text-sm bg-amber-500/10 border border-amber-500/30 text-amber-200 rounded-xl px-3 py-2.5"
          >
            <svg class="w-4 h-4 mt-0.5 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v4m0 4h.01M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z" />
            </svg>
            <p>
              <span class="font-bold">Strategy constraint:</span>
              Weighted load balancers only work with static pools (members + weights). Dynamic label-selector pools are rejected.
            </p>
          </div>

          <div v-if="isLeastBytes" class="space-y-2">
            <div class="flex items-center gap-1.5">
              <label class="text-xs font-bold text-on-surface" for="lb-reset">Reset Interval</label>
              <AtomsInfoTip :text="tipReset" />
              <span class="text-[10px] font-bold uppercase tracking-wider px-1.5 py-0.5 rounded bg-primary/15 text-primary">
                least-bytes
              </span>
            </div>
            <input
              id="lb-reset"
              v-model="resetInterval"
              type="text"
              placeholder="1m (default)"
              class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-xl px-4 py-3 text-sm font-mono text-on-surface focus:border-primary focus:ring-4 focus:ring-primary/20 outline-none transition-all"
            >
          </div>

          <p v-if="formError" class="text-error text-sm" role="alert">
            {{ formError }}
          </p>
        </div>

        <div class="shrink-0 flex items-center justify-between gap-3 px-6 py-4 border-t border-outline-variant/10 bg-surface-container-low/40">
          <span class="text-[10px] font-mono text-outline">pgway.v1.Balancer</span>
          <div class="flex items-center gap-3">
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
              {{ saving ? 'Saving…' : 'Save Load Balancer' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>
