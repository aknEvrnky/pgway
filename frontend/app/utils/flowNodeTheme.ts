import type { FlowGraphKind } from '~/types'

/** Node colors aligned with the flow editor palette design. */
export const FLOW_NODE_COLORS: Record<FlowGraphKind, string> = {
  entrypoint: '#4ade80', // green — entry
  flow: '#38bdf8', // sky blue — pipeline hub
  router: '#fb923c', // orange — fork / rules
  balancer: '#c084fc', // purple — distribute
  pool: '#2dd4bf', // teal — stack / group
  proxy: '#60a5fa', // light blue — upstream
}

/** Inline styles — Tailwind arbitrary values from dynamic hex are unreliable. */
export function flowNodeInline(kind: FlowGraphKind): {
  borderColor: string
  backgroundColor: string
  color: string
} {
  const c = FLOW_NODE_COLORS[kind] || '#94a3b8'
  return {
    borderColor: `${c}73`, // ~45%
    backgroundColor: `${c}1f`, // ~12%
    color: c,
  }
}
