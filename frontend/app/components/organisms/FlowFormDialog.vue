<script setup lang="ts">
import type { BalancerListResponse, Flow, FlowApplyRequest, LoadBalancer, Router, RouterListResponse } from '~/types'

const props = defineProps<{
  open: boolean
  mode?: 'create' | 'edit'
  initial?: Flow | null
}>()

const emit = defineEmits<{
  close: []
  saved: [flow: Flow]
}>()

const { apply } = useFlows()
const { apiFetch } = useApi()

type BindMode = 'router' | 'direct'

const name = ref('')
const bindMode = ref<BindMode>('router')
const routerId = ref('')
const balancerId = ref('')
const routers = ref<Router[]>([])
const balancers = ref<LoadBalancer[]>([])
const listsLoading = ref(false)
const formError = ref('')
const saving = ref(false)

const isEdit = computed(() => props.mode === 'edit')
const dialogTitle = computed(() => (isEdit.value ? 'Edit Flow' : 'Add New Flow'))
const nameReadonly = computed(() => isEdit.value)

const tipName = 'Resource ID (metadata.name). Immutable after create. Referenced by Entrypoint.flow_id.'
const tipMode = 'Router mode evaluates ordered rules. Direct mode always uses one balancer. Data plane prefers router_id when both are set.'

watch(
  () => [props.open, props.initial, props.mode] as const,
  async () => {
    if (!props.open) return
    formError.value = ''
    saving.value = false
    listsLoading.value = true
    try {
      const [rRes, bRes] = await Promise.all([
        apiFetch<RouterListResponse>('/api/v1/routers?page_size=100'),
        apiFetch<BalancerListResponse>('/api/v1/balancers?page_size=100'),
      ])
      routers.value = rRes.items || []
      balancers.value = bRes.items || []
    }
    catch {
      routers.value = []
      balancers.value = []
    }
    finally {
      listsLoading.value = false
    }

    if (isEdit.value && props.initial) {
      name.value = props.initial.id
      if (props.initial.router_id) {
        bindMode.value = 'router'
        routerId.value = props.initial.router_id
        balancerId.value = props.initial.balancer_id || ''
      }
      else {
        bindMode.value = 'direct'
        routerId.value = ''
        balancerId.value = props.initial.balancer_id || ''
      }
    }
    else {
      name.value = ''
      bindMode.value = 'router'
      routerId.value = ''
      balancerId.value = ''
    }
  },
  { immediate: true },
)

function buildBody(): FlowApplyRequest {
  const metaName = name.value.trim()
  if (!metaName) throw new Error('name is required')

  const body: FlowApplyRequest = {
    metadata: { name: metaName },
    spec: {},
  }

  if (bindMode.value === 'router') {
    const rid = routerId.value.trim()
    if (!rid) throw new Error('router_id is required in router mode')
    body.spec.router_id = rid
  }
  else {
    const bid = balancerId.value.trim()
    if (!bid) throw new Error('balancer_id is required in direct mode')
    body.spec.balancer_id = bid
  }

  return body
}

async function onSave() {
  formError.value = ''
  saving.value = true
  try {
    const flow = await apply(buildBody())
    useAppToast().success(isEdit.value ? 'Flow updated' : 'Flow created')
    emit('saved', flow)
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
            Bind this flow to a router or a load balancer.
          </p>
        </div>

        <div class="flex-1 min-h-0 overflow-y-auto overscroll-contain px-6 py-4 space-y-5">
          <div class="space-y-2">
            <div class="flex items-center gap-1.5">
              <label class="text-xs font-bold text-on-surface" for="flow-name">Name</label>
              <AtomsInfoTip :text="tipName" />
            </div>
            <input
              id="flow-name"
              v-model="name"
              type="text"
              :readonly="nameReadonly"
              required
              placeholder="e.g. edge-flow"
              class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-xl px-4 py-3 text-sm font-mono text-on-surface focus:border-primary focus:ring-4 focus:ring-primary/20 outline-none transition-all read-only:opacity-70"
            >
          </div>

          <div>
            <div class="flex items-center gap-1.5 mb-2">
              <label class="text-[10px] font-bold tracking-widest text-on-surface-variant uppercase">
                Binding Mode *
              </label>
              <AtomsInfoTip :text="tipMode" />
            </div>
            <div class="grid grid-cols-2 gap-1 p-1 rounded-lg bg-surface-container-lowest border border-outline-variant/15">
              <button
                type="button"
                class="py-2.5 text-xs font-bold rounded-md transition-colors"
                :class="bindMode === 'router' ? 'bg-surface-container-highest text-primary shadow-sm' : 'text-on-surface-variant hover:text-on-surface'"
                @click="bindMode = 'router'"
              >
                Router
              </button>
              <button
                type="button"
                class="py-2.5 text-xs font-bold rounded-md transition-colors"
                :class="bindMode === 'direct' ? 'bg-surface-container-highest text-primary shadow-sm' : 'text-on-surface-variant hover:text-on-surface'"
                @click="bindMode = 'direct'"
              >
                Direct balancer
              </button>
            </div>
          </div>

          <div v-if="bindMode === 'router'" class="space-y-2">
            <label class="text-xs font-bold text-on-surface" for="flow-router">Router ID</label>
            <select
              id="flow-router"
              v-model="routerId"
              :disabled="listsLoading"
              class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-xl px-4 py-3 text-sm font-mono text-on-surface outline-none focus:ring-1 focus:ring-primary/20 disabled:opacity-60"
            >
              <option value="" disabled>
                {{ listsLoading ? 'Loading…' : 'Select a router' }}
              </option>
              <option v-for="r in routers" :key="r.id" :value="r.id">
                {{ r.id }}{{ r.title ? ` — ${r.title}` : '' }}
              </option>
            </select>
          </div>

          <div v-else class="space-y-2">
            <label class="text-xs font-bold text-on-surface" for="flow-balancer">Balancer ID</label>
            <select
              id="flow-balancer"
              v-model="balancerId"
              :disabled="listsLoading"
              class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-xl px-4 py-3 text-sm font-mono text-on-surface outline-none focus:ring-1 focus:ring-primary/20 disabled:opacity-60"
            >
              <option value="" disabled>
                {{ listsLoading ? 'Loading…' : 'Select a balancer' }}
              </option>
              <option v-for="b in balancers" :key="b.id" :value="b.id">
                {{ b.id }}{{ b.title ? ` — ${b.title}` : '' }}
              </option>
            </select>
          </div>

          <p v-if="formError" class="text-error text-sm" role="alert">
            {{ formError }}
          </p>
        </div>

        <div class="shrink-0 flex items-center justify-between gap-3 px-6 py-4 border-t border-outline-variant/10 bg-surface-container-low/40">
          <span class="text-[10px] font-mono text-outline">pgway.v1.Flow</span>
          <div class="flex items-center gap-3">
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
              {{ saving ? 'Saving…' : 'Save Flow' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>
