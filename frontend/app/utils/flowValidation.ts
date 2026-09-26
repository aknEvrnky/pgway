import type { Flow } from '~/types'

/** Client-side checks mirroring FlowSpecV1 before POST. */
export function validateFlowForStore(flow: Flow): string | null {
  const name = (flow.id || '').trim()
  if (!name) return 'Flow name is required'
  if (/\s/.test(name)) return 'Flow name must not contain spaces'
  if (!flow.router_id && !flow.balancer_id) {
    return 'Attach a router or a load balancer before saving'
  }
  return null
}

export function isFlowStoreReady(flow: Flow): boolean {
  return validateFlowForStore(flow) === null
}
