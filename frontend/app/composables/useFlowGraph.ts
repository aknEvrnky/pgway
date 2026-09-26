import type {
  BalancerListResponse,
  Entrypoint,
  Flow,
  FlowGraphKind,
  LoadBalancer,
  Pool,
  PoolListResponse,
  Proxy,
  Router,
} from '~/types'
import {
  balancerToYamlDoc,
  docsToMultiYaml,
  entrypointToYamlDoc,
  flowToYamlDoc,
  poolToYamlDoc,
  proxyToYamlDoc,
  routerToYamlDoc,
  type YamlDoc,
} from '~/utils/resourceYaml'
import { sortByApplyOrder } from '~/utils/applyOrder'
import type { Edge, Node } from '@vue-flow/core'

export interface FlowGraphBundle {
  flow: Flow
  entrypoints: Entrypoint[]
  router: Router | null
  balancers: LoadBalancer[]
  pools: Pool[]
  proxies: Proxy[]
}

export interface FlowNodeData {
  kind: FlowGraphKind
  label: string
  subtitle?: string
  resourceId: string
  dirty?: boolean
}

const X_GAP = 220
const Y_BASE = 160

function nodeId(kind: FlowGraphKind, id: string) {
  return `${kind}:${id}`
}

function poolSubtitle(p: Pool): string {
  if (p.type === 'dynamic') {
    const n = Object.keys(p.selector?.allow || {}).length
    return n ? `dynamic · ${n} label${n === 1 ? '' : 's'}` : 'dynamic'
  }
  const n = (p.members || []).length
  return `static · ${n} member${n === 1 ? '' : 's'}`
}

export function bundleToVueFlow(bundle: FlowGraphBundle, dirtyIds: Set<string>): { nodes: Node<FlowNodeData>[], edges: Edge[] } {
  const nodes: Node<FlowNodeData>[] = []
  const edges: Edge[] = []
  let col = 0

  const mark = (kind: FlowGraphKind, id: string) => dirtyIds.has(`${kind}:${id}`)

  for (const ep of bundle.entrypoints) {
    nodes.push({
      id: nodeId('entrypoint', ep.id),
      type: 'pgway',
      position: { x: col * X_GAP, y: Y_BASE + nodes.filter(n => n.data?.kind === 'entrypoint').length * 90 },
      data: {
        kind: 'entrypoint',
        label: ep.id,
        subtitle: `${ep.protocol}://${ep.host}:${ep.port}`,
        resourceId: ep.id,
        dirty: mark('entrypoint', ep.id),
      },
    })
    edges.push({
      id: `e-ep-${ep.id}`,
      source: nodeId('entrypoint', ep.id),
      target: nodeId('flow', bundle.flow.id || '__draft__'),
      animated: true,
    })
  }
  col = 1

  nodes.push({
    id: nodeId('flow', bundle.flow.id || '__draft__'),
    type: 'pgway',
    position: { x: col * X_GAP, y: Y_BASE },
    data: {
      kind: 'flow',
      label: bundle.flow.id || '(unnamed)',
      subtitle: bundle.flow.router_id ? 'router mode' : bundle.flow.balancer_id ? 'direct balancer' : 'draft — bind next',
      resourceId: bundle.flow.id || '__draft__',
      dirty: mark('flow', bundle.flow.id || '__draft__'),
    },
  })
  col = 2

  if (bundle.router) {
    nodes.push({
      id: nodeId('router', bundle.router.id),
      type: 'pgway',
      position: { x: col * X_GAP, y: Y_BASE },
      data: {
        kind: 'router',
        label: bundle.router.id,
        subtitle: `${(bundle.router.rules || []).length} rules`,
        resourceId: bundle.router.id,
        dirty: mark('router', bundle.router.id),
      },
    })
    edges.push({
      id: `e-flow-router`,
      source: nodeId('flow', bundle.flow.id || '__draft__'),
      target: nodeId('router', bundle.router.id),
      animated: true,
    })
    col = 3
  }

  bundle.balancers.forEach((b, i) => {
    const x = col * X_GAP
    nodes.push({
      id: nodeId('balancer', b.id),
      type: 'pgway',
      position: { x, y: Y_BASE + i * 100 },
      data: {
        kind: 'balancer',
        label: b.id,
        subtitle: b.type,
        resourceId: b.id,
        dirty: mark('balancer', b.id),
      },
    })
    if (bundle.router) {
      const targets = new Set((bundle.router.rules || []).map(r => r.target))
      if (targets.has(b.id)) {
        edges.push({
          id: `e-router-lb-${b.id}`,
          source: nodeId('router', bundle.router.id),
          target: nodeId('balancer', b.id),
        })
      }
    }
    else if (bundle.flow.balancer_id === b.id) {
      edges.push({
        id: `e-flow-lb-${b.id}`,
        source: nodeId('flow', bundle.flow.id || '__draft__'),
        target: nodeId('balancer', b.id),
        animated: true,
      })
    }
  })
  col += 1

  bundle.pools.forEach((p, i) => {
    nodes.push({
      id: nodeId('pool', p.id),
      type: 'pgway',
      position: { x: col * X_GAP, y: Y_BASE + i * 100 },
      data: {
        kind: 'pool',
        label: p.id,
        subtitle: poolSubtitle(p),
        resourceId: p.id,
        dirty: mark('pool', p.id),
      },
    })
    for (const b of bundle.balancers) {
      if (b.pool_id === p.id) {
        edges.push({
          id: `e-lb-pool-${b.id}-${p.id}`,
          source: nodeId('balancer', b.id),
          target: nodeId('pool', p.id),
        })
      }
    }
  })
  col += 1

  bundle.proxies.forEach((px, i) => {
    nodes.push({
      id: nodeId('proxy', px.id),
      type: 'pgway',
      position: { x: col * X_GAP, y: 80 + i * 80 },
      data: {
        kind: 'proxy',
        label: px.id,
        subtitle: `${px.host}:${px.port}`,
        resourceId: px.id,
        dirty: mark('proxy', px.id),
      },
    })
    for (const p of bundle.pools) {
      if (p.type === 'static' && (p.members || []).some(m => m.proxy_id === px.id)) {
        edges.push({
          id: `e-pool-proxy-${p.id}-${px.id}`,
          source: nodeId('pool', p.id),
          target: nodeId('proxy', px.id),
        })
      }
    }
  })

  return { nodes, edges }
}

export function bundleToYaml(bundle: FlowGraphBundle): string {
  const docs: YamlDoc[] = []
  for (const p of bundle.proxies) docs.push(proxyToYamlDoc(p))
  for (const p of bundle.pools) docs.push(poolToYamlDoc(p))
  for (const b of bundle.balancers) docs.push(balancerToYamlDoc(b))
  if (bundle.router) docs.push(routerToYamlDoc(bundle.router))
  docs.push(flowToYamlDoc(bundle.flow))
  for (const e of bundle.entrypoints) docs.push(entrypointToYamlDoc(e))
  return docsToMultiYaml(sortByApplyOrder(docs))
}

export function emptyDraftBundle(): FlowGraphBundle {
  return {
    flow: { id: '' },
    entrypoints: [],
    router: null,
    balancers: [],
    pools: [],
    proxies: [],
  }
}

/** Resolve graph from a Flow object (does not GET the flow itself). */
export async function resolveFlowGraph(flow: Flow, opts?: { includeEntrypoints?: boolean }): Promise<FlowGraphBundle> {
  const { apiFetch } = useApi()
  const { list: listEntrypoints } = useEntrypoints()

  let entrypoints: Entrypoint[] = []
  if (opts?.includeEntrypoints !== false && flow.id) {
    try {
      const epRes = await listEntrypoints({ flow_id: flow.id, page_size: 100 })
      entrypoints = epRes.items || []
    }
    catch {
      entrypoints = []
    }
  }

  let router: Router | null = null
  const balancers: LoadBalancer[] = []
  const balancerIds = new Set<string>()

  if (flow.router_id) {
    try {
      router = await apiFetch<Router>(`/api/v1/routers/${encodeURIComponent(flow.router_id)}`)
      for (const rule of router.rules || []) {
        if (rule.target) balancerIds.add(rule.target)
      }
    }
    catch {
      router = null
    }
  }
  if (flow.balancer_id) balancerIds.add(flow.balancer_id)

  for (const id of balancerIds) {
    try {
      balancers.push(await apiFetch<LoadBalancer>(`/api/v1/balancers/${encodeURIComponent(id)}`))
    }
    catch {
      // missing ref — skip
    }
  }

  const poolIds = new Set(balancers.map(b => b.pool_id).filter(Boolean))
  const pools: Pool[] = []
  for (const id of poolIds) {
    try {
      pools.push(await apiFetch<Pool>(`/api/v1/pools/${encodeURIComponent(id)}`))
    }
    catch {
      // skip
    }
  }

  const proxyIds = new Set<string>()
  for (const p of pools) {
    if (p.type === 'static') {
      for (const m of p.members || []) proxyIds.add(m.proxy_id)
    }
  }

  const proxies: Proxy[] = []
  for (const id of proxyIds) {
    try {
      proxies.push(await apiFetch<Proxy>(`/api/v1/proxies/${encodeURIComponent(id)}`))
    }
    catch {
      // skip
    }
  }

  return { flow, entrypoints, router, balancers, pools, proxies }
}

export async function loadFlowBundle(flowName: string): Promise<FlowGraphBundle> {
  const { apiFetch } = useApi()
  const flow = await apiFetch<Flow>(`/api/v1/flows/${encodeURIComponent(flowName)}`)
  return resolveFlowGraph(flow)
}

/** Keep helpers referenced for list pickers in palette. */
export async function listBalancersPage() {
  const { apiFetch } = useApi()
  return apiFetch<BalancerListResponse>('/api/v1/balancers?page_size=100')
}

export async function listPoolsPage() {
  const { apiFetch } = useApi()
  return apiFetch<PoolListResponse>('/api/v1/pools?page_size=100')
}
