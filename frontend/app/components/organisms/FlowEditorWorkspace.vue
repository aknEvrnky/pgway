<script setup lang="ts">
import { VueFlow } from '@vue-flow/core'
import { MiniMap } from '@vue-flow/minimap'
import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'
import '@vue-flow/minimap/dist/style.css'
import type { NodeMouseEvent } from '@vue-flow/core'
import type {
  Entrypoint,
  Flow,
  FlowGraphKind,
  LoadBalancer,
  Pool,
  Proxy,
  Router,
} from '~/types'
import {
  bundleToVueFlow,
  bundleToYaml,
  emptyDraftBundle,
  loadFlowBundle,
  resolveFlowGraph,
  type FlowGraphBundle,
  type FlowNodeData,
} from '~/composables/useFlowGraph'
import { sortByApplyOrder } from '~/utils/applyOrder'
import { FLOW_NODE_COLORS } from '~/utils/flowNodeTheme'
import { validateFlowForStore } from '~/utils/flowValidation'

const props = defineProps<{
  draft?: boolean
  flowName?: string
}>()

const toast = useAppToast()
const { apply: applyFlow } = useFlows()
const { apply: applyEntrypoint, get: getEntrypoint, remove: removeEntrypoint } = useEntrypoints()
const { apply: applyBalancer } = useBalancers()
const { apply: applyPool } = usePools()

const isDraft = computed(() => !!props.draft)
const flowName = computed(() => {
  if (isDraft.value) return bundle.value?.flow.id || ''
  return props.flowName || ''
})

const loading = ref(true)
const error = ref('')
const bundle = ref<FlowGraphBundle | null>(null)
const tab = ref<'settings' | 'yaml'>('settings')
const selectedKind = ref<FlowGraphKind | null>(null)
const selectedId = ref<string | null>(null)
const deploying = ref(false)

const dirty = ref<Map<string, { kind: string, name: string, apply: () => Promise<void> }>>(new Map())

const attachOpen = ref(false)
const attachKind = ref<FlowGraphKind | null>(null)

const epDetachOpen = ref(false)
const epDetachError = ref<string | null>(null)
const epDetachLoading = ref(false)
const epDetaching = ref<Entrypoint | null>(null)

type FormKind = FlowGraphKind
const formOpen = ref(false)
const formMode = ref<'create' | 'edit'>('create')
const formKind = ref<FormKind | null>(null)
const editingFlow = ref<Flow | null>(null)
const editingEntrypoint = ref<Entrypoint | null>(null)
const editingRouter = ref<Router | null>(null)
const editingBalancer = ref<LoadBalancer | null>(null)
const editingPool = ref<Pool | null>(null)
const editingProxy = ref<Proxy | null>(null)

const dirtyIds = computed(() => new Set([...dirty.value.keys()]))
const dirtyCount = computed(() => dirty.value.size)

const graph = computed(() => {
  if (!bundle.value) return { nodes: [], edges: [] }
  return bundleToVueFlow(bundle.value, dirtyIds.value)
})

const yamlText = computed(() => (bundle.value ? bundleToYaml(bundle.value) : ''))

function flowDirtyKey(id?: string) {
  return `flow:${id || '__draft__'}`
}

function syncFlowDirtyApply() {
  if (!bundle.value) return
  const flow = bundle.value.flow
  markDirty(flowDirtyKey(flow.id), 'Flow', flow.id || '__draft__', async () => {
    const err = validateFlowForStore(flow)
    if (err) throw new Error(err)
    await applyFlow({
      metadata: { name: flow.id },
      spec: {
        router_id: flow.router_id,
        balancer_id: flow.balancer_id,
      },
    })
  })
}

async function reload() {
  loading.value = true
  error.value = ''
  try {
    if (isDraft.value) {
      bundle.value = emptyDraftBundle()
      dirty.value = new Map()
      selectedKind.value = 'flow'
      selectedId.value = '__draft__'
      tab.value = 'settings'
    }
    else {
      bundle.value = await loadFlowBundle(props.flowName || '')
      dirty.value = new Map()
    }
  }
  catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'failed to load flow'
    bundle.value = null
  }
  finally {
    loading.value = false
  }
}

onMounted(() => {
  reload().catch(() => {})
})

watch(() => props.flowName, () => {
  if (!isDraft.value) reload().catch(() => {})
})

function onNodeClick(ev: NodeMouseEvent) {
  const data = ev.node.data as FlowNodeData | undefined
  if (!data) return
  selectedKind.value = data.kind
  selectedId.value = data.resourceId
  tab.value = 'settings'
}

function markDirty(key: string, kind: string, name: string, apply: () => Promise<void>) {
  const next = new Map(dirty.value)
  next.set(key, { kind, name, apply })
  dirty.value = next
}

function patchFlowName(name: string) {
  if (!bundle.value || !isDraft.value) return
  const prevKey = flowDirtyKey(bundle.value.flow.id)
  const flow: Flow = { ...bundle.value.flow, id: name }
  const eps = bundle.value.entrypoints.map(e => ({ ...e, flow_id: name || e.flow_id }))
  bundle.value = { ...bundle.value, flow, entrypoints: eps }
  if (selectedKind.value === 'flow') selectedId.value = name || '__draft__'
  const next = new Map(dirty.value)
  next.delete(prevKey)
  dirty.value = next
  syncFlowDirtyApply()
}

function patchFlow(patch: Partial<Pick<Flow, 'router_id' | 'balancer_id'>>) {
  if (!bundle.value) return
  const flow: Flow = { ...bundle.value.flow }
  if ('router_id' in patch) {
    if (patch.router_id) flow.router_id = patch.router_id
    else delete flow.router_id
  }
  if ('balancer_id' in patch) {
    if (patch.balancer_id) flow.balancer_id = patch.balancer_id
    else delete flow.balancer_id
  }
  bundle.value = { ...bundle.value, flow }
  syncFlowDirtyApply()
  reloadAfterLocalPatch().catch(() => {})
}

function openAttach(kind: FlowGraphKind) {
  if (kind === 'flow') return
  attachKind.value = kind
  attachOpen.value = true
}

/** Palette + — open attach picker starting at entrypoint (Create new is inside). */
function onPaletteCreate() {
  attachKind.value = 'entrypoint'
  attachOpen.value = true
}

function minimapNodeColor(node: { data?: FlowNodeData }) {
  const kind = node.data?.kind
  if (!kind) return '#64748b'
  return FLOW_NODE_COLORS[kind] || '#64748b'
}

function openCreateForm(kind: FlowGraphKind) {
  attachOpen.value = false
  formMode.value = 'create'
  formKind.value = kind
  editingFlow.value = null
  editingEntrypoint.value = null
  editingRouter.value = null
  editingBalancer.value = null
  editingPool.value = null
  editingProxy.value = null
  formOpen.value = true
}

function openEdit(kind: FlowGraphKind) {
  if (!bundle.value || !selectedId.value) return
  formMode.value = 'edit'
  formKind.value = kind
  editingFlow.value = null
  editingEntrypoint.value = null
  editingRouter.value = null
  editingBalancer.value = null
  editingPool.value = null
  editingProxy.value = null

  const b = bundle.value
  const id = selectedId.value
  switch (kind) {
    case 'flow':
      editingFlow.value = b.flow
      break
    case 'entrypoint':
      editingEntrypoint.value = b.entrypoints.find(e => e.id === id) || null
      break
    case 'router':
      editingRouter.value = b.router?.id === id ? b.router : null
      break
    case 'balancer':
      editingBalancer.value = b.balancers.find(x => x.id === id) || null
      break
    case 'pool':
      editingPool.value = b.pools.find(x => x.id === id) || null
      break
    case 'proxy':
      editingProxy.value = b.proxies.find(x => x.id === id) || null
      break
  }

  const hasResource = kind === 'flow'
    ? !!editingFlow.value
    : !!(editingEntrypoint.value || editingRouter.value || editingBalancer.value
      || editingPool.value || editingProxy.value)
  if (!hasResource) {
    toast.error('Resource not loaded in this graph')
    return
  }
  if (kind === 'flow' && isDraft.value) {
    // Name/bindings are edited inline in the sidebar for drafts
    selectedKind.value = 'flow'
    selectedId.value = b.flow.id || '__draft__'
    return
  }
  formOpen.value = true
}

function resolveTargetBalancer(): LoadBalancer | null {
  if (!bundle.value) return null
  if (selectedKind.value === 'balancer' && selectedId.value) {
    return bundle.value.balancers.find(b => b.id === selectedId.value) || null
  }
  if (bundle.value.balancers.length === 1) return bundle.value.balancers[0]
  return null
}

function resolveTargetStaticPool(): Pool | null {
  if (!bundle.value) return null
  if (selectedKind.value === 'pool' && selectedId.value) {
    const p = bundle.value.pools.find(x => x.id === selectedId.value)
    if (p?.type === 'static') return p
  }
  const statics = bundle.value.pools.filter(p => p.type === 'static')
  if (statics.length === 1) return statics[0]
  return null
}

async function onAttachPick(id: string) {
  attachOpen.value = false
  const kind = attachKind.value
  if (!kind || !bundle.value) return

  try {
    if (kind === 'router') {
      patchFlow({ router_id: id })
      toast.info(`Router ${id} staged — Deploy to apply`)
      await reloadAfterLocalPatch()
      return
    }
    if (kind === 'balancer') {
      patchFlow({ balancer_id: id })
      toast.info(`Balancer ${id} staged — Deploy to apply`)
      await reloadAfterLocalPatch()
      return
    }
    if (kind === 'entrypoint') {
      if (!bundle.value.flow.id) {
        toast.error('Set the flow name before attaching an entrypoint')
        return
      }
      const ep = await getEntrypoint(id)
      const owner = (ep.flow_id || '').trim()
      const flowId = bundle.value.flow.id
      if (owner && owner !== flowId) {
        toast.error(`Entrypoint "${id}" belongs to flow "${owner}" — cannot rebind. Create a new entrypoint instead.`)
        return
      }
      if (bundle.value.entrypoints.some(e => e.id === id) && owner === flowId) {
        toast.info(`Entrypoint ${id} is already on this flow`)
        return
      }
      const next: Entrypoint = { ...ep, flow_id: flowId }
      const eps = [...bundle.value.entrypoints.filter(e => e.id !== id), next]
      bundle.value = { ...bundle.value, entrypoints: eps }
      markDirty(`entrypoint:${id}`, 'Entrypoint', id, async () => {
        await applyEntrypoint({
          metadata: { name: next.id },
          spec: {
            title: next.title,
            protocol: next.protocol,
            host: next.host,
            port: next.port,
            flow_id: next.flow_id,
          },
        })
      })
      toast.info(`Entrypoint ${id} staged — Create / Deploy to apply`)
      return
    }
    if (kind === 'pool') {
      const lb = resolveTargetBalancer()
      if (!lb) {
        toast.error('Select a Load Balancer node first, then attach a Pool')
        return
      }
      const next: LoadBalancer = { ...lb, pool_id: id }
      const balancers = bundle.value.balancers.map(b => (b.id === lb.id ? next : b))
      bundle.value = { ...bundle.value, balancers }
      markDirty(`balancer:${lb.id}`, 'LoadBalancer', lb.id, async () => {
        const spec: Parameters<typeof applyBalancer>[0]['spec'] = {
          title: next.title,
          type: next.type,
          pool_id: next.pool_id,
        }
        if (next.type === 'least-bytes' && next.reset_interval) {
          spec.reset_interval = next.reset_interval
        }
        await applyBalancer({
          metadata: { name: next.id },
          spec,
        })
      })
      toast.info(`Pool ${id} → balancer ${lb.id} staged — Deploy to apply`)
      await reloadAfterLocalPatch()
      return
    }
    if (kind === 'proxy') {
      const pool = resolveTargetStaticPool()
      if (!pool) {
        toast.error('Select a static Pool node first, then attach a Proxy')
        return
      }
      const members = [...(pool.members || [])]
      if (members.some(m => m.proxy_id === id)) {
        toast.info(`Proxy ${id} already in pool ${pool.id}`)
        return
      }
      members.push({ proxy_id: id, weight: 1 })
      const next: Pool = { ...pool, members }
      const pools = bundle.value.pools.map(p => (p.id === pool.id ? next : p))
      bundle.value = { ...bundle.value, pools }
      markDirty(`pool:${pool.id}`, 'Pool', pool.id, async () => {
        await applyPool({
          metadata: { name: next.id, labels: next.labels },
          spec: {
            title: next.title,
            type: 'static',
            members: members.map(m => ({ proxy_id: m.proxy_id, weight: m.weight })),
          },
        })
      })
      toast.info(`Proxy ${id} → pool ${pool.id} staged — Deploy to apply`)
      await reloadAfterLocalPatch()
      return
    }
  }
  catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : 'attach failed')
  }
}

async function reloadAfterLocalPatch() {
  if (!bundle.value) return
  const kept = dirty.value
  const flow = bundle.value.flow
  const localEps = bundle.value.entrypoints.filter(e => kept.has(`entrypoint:${e.id}`))
  const localBalancers = bundle.value.balancers
  const localPools = bundle.value.pools
  try {
    // Draft (or missing flow on server): resolve refs only — do not GET the flow
    const fresh = (isDraft.value || !flow.id)
      ? await resolveFlowGraph(flow, { includeEntrypoints: false })
      : await loadFlowBundle(flow.id)
    fresh.flow = flow
    for (const lb of localBalancers) {
      const i = fresh.balancers.findIndex(b => b.id === lb.id)
      if (i >= 0 && kept.has(`balancer:${lb.id}`)) fresh.balancers[i] = lb
      else if (i < 0 && kept.has(`balancer:${lb.id}`)) fresh.balancers.push(lb)
    }
    for (const p of localPools) {
      const i = fresh.pools.findIndex(x => x.id === p.id)
      if (i >= 0 && kept.has(`pool:${p.id}`)) fresh.pools[i] = p
      else if (i < 0 && kept.has(`pool:${p.id}`)) fresh.pools.push(p)
    }
    for (const ep of localEps) {
      const i = fresh.entrypoints.findIndex(e => e.id === ep.id)
      if (i >= 0) fresh.entrypoints[i] = ep
      else fresh.entrypoints.push(ep)
    }
    bundle.value = fresh
    dirty.value = kept
  }
  catch {
    // keep local
  }
}

async function onResourceSaved() {
  formOpen.value = false
  if (isDraft.value || dirty.value.size) await reloadAfterLocalPatch()
  else await reload()
}

async function onEntrypointSaved(ep: Entrypoint) {
  formOpen.value = false
  if (isDraft.value) {
    if (!bundle.value?.flow.id) {
      toast.error('Set the flow name before creating an entrypoint')
      return
    }
    // Form already applied with defaultFlowId — hydrate into local graph
    if (ep.flow_id === bundle.value.flow.id) {
      const eps = [...bundle.value.entrypoints.filter(e => e.id !== ep.id), ep]
      bundle.value = { ...bundle.value, entrypoints: eps }
    }
    await reloadAfterLocalPatch()
    return
  }
  await onResourceSaved()
}

async function onFlowSaved(flow: Flow) {
  formOpen.value = false
  if (flow.id !== flowName.value) {
    await navigateTo(`/flows/${encodeURIComponent(flow.id)}`)
    return
  }
  await reload()
}

async function deploy() {
  if (!bundle.value) return
  const flow = bundle.value.flow
  const err = validateFlowForStore(flow)
  if (err) {
    toast.error(err)
    selectedKind.value = 'flow'
    selectedId.value = flow.id || '__draft__'
    tab.value = 'settings'
    return
  }

  syncFlowDirtyApply()
  // Refresh entrypoint dirty applies with current flow_id
  for (const ep of bundle.value.entrypoints) {
    if (!dirty.value.has(`entrypoint:${ep.id}`)) continue
    const next = { ...ep, flow_id: flow.id }
    markDirty(`entrypoint:${ep.id}`, 'Entrypoint', ep.id, async () => {
      await applyEntrypoint({
        metadata: { name: next.id },
        spec: {
          title: next.title,
          protocol: next.protocol,
          host: next.host,
          port: next.port,
          flow_id: flow.id,
        },
      })
    })
  }

  if (!dirty.value.size && !isDraft.value) return

  deploying.value = true
  try {
    const jobs = sortByApplyOrder([...dirty.value.values()])
    for (const job of jobs) {
      await job.apply()
    }
    if (isDraft.value) {
      toast.success(`Flow "${flow.id}" created`)
      await navigateTo(`/flows/${encodeURIComponent(flow.id)}`)
      return
    }
    toast.success(`Deployed ${jobs.length} resource(s)`)
    await reload()
  }
  catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : 'deploy failed')
  }
  finally {
    deploying.value = false
  }
}

function closeForm() {
  formOpen.value = false
}

function openCreateFormGuarded(kind: FlowGraphKind) {
  if (kind === 'entrypoint' && isDraft.value) {
    toast.error('Create the flow first, then add entrypoints')
    return
  }
  openCreateForm(kind)
}

function openDetachEntrypoint() {
  if (!bundle.value || selectedKind.value !== 'entrypoint' || !selectedId.value) return
  const ep = bundle.value.entrypoints.find(e => e.id === selectedId.value)
  if (!ep) {
    toast.error('Entrypoint not found on this flow')
    return
  }
  epDetaching.value = ep
  epDetachError.value = null
  epDetachOpen.value = true
}

async function confirmDetachEntrypoint() {
  if (!bundle.value || !epDetaching.value) return
  const ep = epDetaching.value
  const id = ep.id
  epDetachLoading.value = true
  epDetachError.value = null
  try {
    await removeEntrypoint(id)
    const nextDirty = new Map(dirty.value)
    nextDirty.delete(`entrypoint:${id}`)
    dirty.value = nextDirty
    const eps = bundle.value.entrypoints.filter(e => e.id !== id)
    bundle.value = { ...bundle.value, entrypoints: eps }
    if (selectedId.value === id) {
      selectedKind.value = 'flow'
      selectedId.value = bundle.value.flow.id || '__draft__'
    }
    epDetachOpen.value = false
    epDetaching.value = null
    toast.success(`Entrypoint "${id}" detached`)
  }
  catch (e: unknown) {
    const msg = e instanceof Error ? e.message : 'detach failed'
    epDetachError.value = msg
    toast.error(msg)
  }
  finally {
    epDetachLoading.value = false
  }
}
</script>

<template>
  <div class="relative h-full w-full flex bg-surface">
    <div class="relative flex-1 min-w-0 h-full">
      <div class="absolute top-4 left-4 z-20 flex items-center gap-3">
        <NuxtLink
          to="/flows"
          class="text-xs font-bold text-on-surface-variant hover:text-primary transition-colors"
        >
          ← Flows
        </NuxtLink>
        <h1 class="text-sm font-black text-on-surface font-mono">
          {{ flowName || (isDraft ? 'New flow' : '') }}
        </h1>
        <span
          v-if="isDraft"
          class="text-[10px] font-bold uppercase tracking-wider px-2 py-0.5 rounded-full bg-primary/20 text-primary border border-primary/30"
        >
          draft
        </span>
        <span
          v-else-if="dirtyCount"
          class="text-[10px] font-bold uppercase tracking-wider px-2 py-0.5 rounded-full bg-amber-500/20 text-amber-200 border border-amber-500/30"
        >
          {{ dirtyCount }} dirty
        </span>
      </div>

      <p v-if="error" class="absolute top-16 left-4 z-20 text-error text-sm" role="alert">
        {{ error }}
      </p>
      <p
        v-else-if="loading"
        class="absolute inset-0 z-10 flex items-center justify-center text-sm text-on-surface-variant bg-surface/40"
      >
        Loading pipeline…
      </p>

      <ClientOnly>
        <VueFlow
          v-if="bundle"
          class="h-full w-full bg-surface-container-lowest/40"
          :nodes="graph.nodes"
          :edges="graph.edges"
          :default-viewport="{ zoom: 0.85 }"
          fit-view-on-init
          @node-click="onNodeClick"
        >
          <template #node-pgway="nodeProps">
            <MoleculesFlowGraphNode v-bind="nodeProps" />
          </template>
          <MiniMap
            position="bottom-left"
            :node-color="minimapNodeColor"
            :mask-color="'rgb(11 19 38 / 0.72)'"
            pannable
            zoomable
            class="!bg-[#121826]/95 !border !border-white/10 !rounded-xl !shadow-xl !m-4"
          />
        </VueFlow>
      </ClientOnly>

      <OrganismsFlowEditorPalette
        @add="openAttach"
        @create="onPaletteCreate"
      />
    </div>

    <OrganismsFlowEditorSidebar
      :tab="tab"
      :yaml-text="yamlText"
      :selected-kind="selectedKind"
      :selected-id="selectedId"
      :bundle="bundle"
      :dirty-count="dirtyCount"
      :deploying="deploying"
      :draft="isDraft"
      @update:tab="tab = $event"
      @patch-flow="patchFlow"
      @patch-flow-name="patchFlowName"
      @edit="openEdit"
      @detach-entrypoint="openDetachEntrypoint"
      @deploy="deploy"
    />

    <OrganismsFlowAttachDialog
      :open="attachOpen"
      :kind="attachKind"
      :flow-id="flowName || '(set name)'"
      @close="attachOpen = false"
      @pick="onAttachPick"
      @create="openCreateFormGuarded(attachKind!)"
    />

    <OrganismsEntrypointDeleteDialog
      :open="epDetachOpen"
      :name="epDetaching?.id || ''"
      :host="epDetaching?.host"
      :port="epDetaching?.port"
      :protocol="epDetaching?.protocol"
      :flow-id="epDetaching?.flow_id"
      :error="epDetachError"
      :loading="epDetachLoading"
      @close="epDetachOpen = false"
      @confirm="confirmDetachEntrypoint"
    />

    <OrganismsFlowFormDialog
      :open="formOpen && formKind === 'flow'"
      :mode="formMode"
      :initial="editingFlow"
      @close="closeForm"
      @saved="onFlowSaved"
    />

    <OrganismsEntrypointFormDialog
      :open="formOpen && formKind === 'entrypoint'"
      :mode="formMode"
      :initial="editingEntrypoint"
      :default-flow-id="flowName"
      @close="closeForm"
      @saved="onEntrypointSaved"
    />

    <OrganismsRouterFormDialog
      :open="formOpen && formKind === 'router'"
      :mode="formMode"
      :initial="editingRouter"
      @close="closeForm"
      @saved="onResourceSaved"
    />

    <OrganismsBalancerFormDialog
      :open="formOpen && formKind === 'balancer'"
      :mode="formMode"
      :initial="editingBalancer"
      @close="closeForm"
      @saved="onResourceSaved"
    />

    <OrganismsPoolFormDialog
      :open="formOpen && formKind === 'pool'"
      :mode="formMode"
      :initial="editingPool"
      @close="closeForm"
      @saved="onResourceSaved"
    />

    <OrganismsProxyFormDialog
      :open="formOpen && formKind === 'proxy'"
      :mode="formMode"
      :initial="editingProxy"
      @close="closeForm"
      @saved="onResourceSaved"
    />
  </div>
</template>

<style>
.vue-flow__edge-path {
  stroke: rgb(100 160 220 / 0.55);
}
.vue-flow__attribution {
  display: none;
}
.vue-flow__minimap {
  overflow: hidden;
}
.vue-flow__minimap-mask {
  fill: rgb(11 19 38 / 0.72);
}
</style>
