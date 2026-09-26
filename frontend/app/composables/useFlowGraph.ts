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
  /** Multiple right-side source handles (router fan-out). */
  sourceHandles?: number
}

/** Must stay ≥ FlowGraphNode fixed width so columns never overlap. */
const NODE_W = 200
/** Default space between node right edge and next column left edge. */
const COL_GAP = 88
/** Wider corridor for router→balancer rule labels. */
const RULE_EDGE_GAP = 200
const Y_BASE = 140
const Y_GAP = 150

function advanceX(x: number, gap = COL_GAP): number {
  return x + NODE_W + gap
}

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

/** Short match descriptor for router→balancer edge labels. */
function ruleMatchKind(rule: { match?: Router['rules'][number]['match'] }): string {
  const m = rule.match || {}
  if (m.all?.length) return 'all'
  if (m.any?.length) return 'any'
  return m.type || '?'
}

function ruleEdgeLabel(rule: Router['rules'][number]): string {
  const id = rule.id?.trim() || '?'
  return `${id} · ${ruleMatchKind(rule)}`
}

/** Spread Y positions evenly around Y_BASE for n siblings. */
function laneY(index: number, count: number): number {
  if (count <= 1) return Y_BASE
  const span = (count - 1) * Y_GAP
  return Y_BASE - span / 2 + index * Y_GAP
}

/**
 * Assign non-overlapping Y slots. Preferred Y is used when free;
 * collisions shift to the nearest free lane (step Y_GAP).
 */
function allocateYs(preferred: Map<string, number>): Map<string, number> {
  const sorted = [...preferred.entries()].sort((a, b) => a[1] - b[1] || a[0].localeCompare(b[0]))
  const used: number[] = []
  const out = new Map<string, number>()
  for (const [id, pref] of sorted) {
    let y = pref
    while (used.some(u => Math.abs(u - y) < Y_GAP - 1)) {
      y += Y_GAP
    }
    used.push(y)
    out.set(id, y)
  }
  return out
}

export function bundleToVueFlow(bundle: FlowGraphBundle, dirtyIds: Set<string>): { nodes: Node<FlowNodeData>[], edges: Edge[] } {
  const nodes: Node<FlowNodeData>[] = []
  const edges: Edge[] = []
  let x = 0

  const mark = (kind: FlowGraphKind, id: string) => dirtyIds.has(`${kind}:${id}`)

  const epCount = bundle.entrypoints.length
  if (epCount) {
    bundle.entrypoints.forEach((ep, i) => {
      nodes.push({
        id: nodeId('entrypoint', ep.id),
        type: 'pgway',
        position: { x, y: laneY(i, epCount) },
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
    })
    x = advanceX(x)
  }

  nodes.push({
    id: nodeId('flow', bundle.flow.id || '__draft__'),
    type: 'pgway',
    position: { x, y: Y_BASE },
    data: {
      kind: 'flow',
      label: bundle.flow.id || '(unnamed)',
      subtitle: bundle.flow.router_id ? 'router mode' : bundle.flow.balancer_id ? 'direct balancer' : 'draft — bind next',
      resourceId: bundle.flow.id || '__draft__',
      dirty: mark('flow', bundle.flow.id || '__draft__'),
    },
  })
  x = advanceX(x)

  if (bundle.router) {
    const outCount = Math.max(bundle.balancers.length, 1)
    nodes.push({
      id: nodeId('router', bundle.router.id),
      type: 'pgway',
      position: { x, y: Y_BASE },
      data: {
        kind: 'router',
        label: bundle.router.id,
        subtitle: `${(bundle.router.rules || []).length} rules`,
        resourceId: bundle.router.id,
        dirty: mark('router', bundle.router.id),
        sourceHandles: outCount,
      },
    })
    edges.push({
      id: `e-flow-router`,
      source: nodeId('flow', bundle.flow.id || '__draft__'),
      target: nodeId('router', bundle.router.id),
      animated: true,
    })
    // Leave room for rule edge labels between router and balancers.
    x = advanceX(x, RULE_EDGE_GAP)
  }

  const balancerYs = new Map<string, number>()
  const lbCount = bundle.balancers.length
  const balancerX = x
  bundle.balancers.forEach((b, i) => {
    const y = laneY(i, lbCount)
    balancerYs.set(b.id, y)
    nodes.push({
      id: nodeId('balancer', b.id),
      type: 'pgway',
      position: { x: balancerX, y },
      data: {
        kind: 'balancer',
        label: b.id,
        subtitle: b.type,
        resourceId: b.id,
        dirty: mark('balancer', b.id),
      },
    })
    if (!bundle.router && bundle.flow.balancer_id === b.id) {
      edges.push({
        id: `e-flow-lb-${b.id}`,
        source: nodeId('flow', bundle.flow.id || '__draft__'),
        target: nodeId('balancer', b.id),
        animated: true,
      })
    }
  })

  if (bundle.router) {
    const known = new Set(bundle.balancers.map(b => b.id))
    const rulesByTarget = new Map<string, Router['rules']>()
    for (const rule of bundle.router.rules || []) {
      if (!rule.target || !known.has(rule.target)) continue
      const list = rulesByTarget.get(rule.target) || []
      list.push(rule)
      rulesByTarget.set(rule.target, list)
    }
    // Stable label order: follow balancer lane order so top/bottom labels match lanes.
    const orderedTargets = bundle.balancers.map(b => b.id).filter(id => rulesByTarget.has(id))
    for (const extra of rulesByTarget.keys()) {
      if (!orderedTargets.includes(extra)) orderedTargets.push(extra)
    }
    orderedTargets.forEach((target, i) => {
      const rules = rulesByTarget.get(target) || []
      edges.push({
        id: `e-router-lb-${target}`,
        source: nodeId('router', bundle.router!.id),
        sourceHandle: `out-${i}`,
        target: nodeId('balancer', target),
        label: rules.map(ruleEdgeLabel).join(' | '),
        labelShowBg: true,
        labelBgPadding: [5, 7] as [number, number],
        labelBgBorderRadius: 6,
        labelStyle: {
          fill: '#fb923c',
          fontSize: 10,
          fontWeight: 700,
          fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
        },
        labelBgStyle: {
          fill: '#171f33',
          fillOpacity: 0.95,
          stroke: 'rgba(251, 146, 60, 0.35)',
          strokeWidth: 1,
        },
        style: { stroke: 'rgb(251 146 60 / 0.55)' },
        data: {
          rules: rules.map(r => ({ id: r.id, matchType: ruleMatchKind(r) })),
          lane: i,
        },
      })
    })
  }
  x = advanceX(balancerX)

  // Prefer placing each pool on the same lane as its balancer(s).
  const poolPreferred = new Map<string, number>()
  bundle.pools.forEach((p, i) => {
    const linked = bundle.balancers.filter(b => b.pool_id === p.id)
    if (linked.length) {
      const sum = linked.reduce((acc, b) => acc + (balancerYs.get(b.id) ?? Y_BASE), 0)
      poolPreferred.set(p.id, sum / linked.length)
    }
    else {
      poolPreferred.set(p.id, laneY(i, bundle.pools.length))
    }
  })
  const poolYs = allocateYs(poolPreferred)

  const poolX = x
  for (const p of bundle.pools) {
    nodes.push({
      id: nodeId('pool', p.id),
      type: 'pgway',
      position: { x: poolX, y: poolYs.get(p.id) ?? Y_BASE },
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
  }
  x = advanceX(poolX)

  // Group proxies under their static pool lane; collide-safe across pools.
  const proxyPreferred = new Map<string, number>()
  for (const p of bundle.pools) {
    if (p.type !== 'static') continue
    const members = p.members || []
    const baseY = poolYs.get(p.id) ?? Y_BASE
    members.forEach((m, i) => {
      if (!m.proxy_id || proxyPreferred.has(m.proxy_id)) return
      const offset = members.length <= 1 ? 0 : (i - (members.length - 1) / 2) * Y_GAP
      proxyPreferred.set(m.proxy_id, baseY + offset)
    })
  }
  // Proxies present in bundle but not linked (shouldn't happen) — fallback lane.
  bundle.proxies.forEach((px, i) => {
    if (!proxyPreferred.has(px.id)) proxyPreferred.set(px.id, laneY(i, bundle.proxies.length))
  })
  const proxyYs = allocateYs(proxyPreferred)

  const proxyX = x
  for (const px of bundle.proxies) {
    nodes.push({
      id: nodeId('proxy', px.id),
      type: 'pgway',
      position: { x: proxyX, y: proxyYs.get(px.id) ?? Y_BASE },
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
  }

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
