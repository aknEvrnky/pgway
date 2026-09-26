function yamlScalar(v: unknown): string {
  if (v === null || v === undefined) return 'null'
  if (typeof v === 'boolean' || typeof v === 'number') return String(v)
  const s = String(v)
  if (s === '' || /[:#{}[\],&*?|>!%@`']/.test(s) || /^\s|\s$/.test(s) || s.includes('\n')) {
    return JSON.stringify(s)
  }
  return s
}

function dumpObject(obj: Record<string, unknown>, indent: number): string {
  const pad = '  '.repeat(indent)
  const lines: string[] = []
  for (const [k, val] of Object.entries(obj)) {
    if (val === undefined) continue
    if (Array.isArray(val)) {
      if (!val.length) {
        lines.push(`${pad}${k}: []`)
        continue
      }
      lines.push(`${pad}${k}:`)
      for (const item of val) {
        if (item !== null && typeof item === 'object' && !Array.isArray(item)) {
          const entries = Object.entries(item as Record<string, unknown>).filter(([, v]) => v !== undefined)
          if (!entries.length) {
            lines.push(`${pad}- {}`)
            continue
          }
          entries.forEach(([ik, iv], idx) => {
            const prefix = idx === 0 ? `${pad}- ` : `${pad}  `
            if (iv !== null && typeof iv === 'object' && !Array.isArray(iv)) {
              lines.push(`${prefix}${ik}:`)
              lines.push(dumpObject(iv as Record<string, unknown>, indent + 2))
            }
            else if (Array.isArray(iv)) {
              lines.push(`${prefix}${ik}:`)
              for (const nested of iv) {
                lines.push(`${pad}    - ${yamlScalar(nested)}`)
              }
            }
            else {
              lines.push(`${prefix}${ik}: ${yamlScalar(iv)}`)
            }
          })
        }
        else {
          lines.push(`${pad}- ${yamlScalar(item)}`)
        }
      }
      continue
    }
    if (val !== null && typeof val === 'object') {
      const nested = dumpObject(val as Record<string, unknown>, indent + 1)
      if (!nested) {
        lines.push(`${pad}${k}: {}`)
      }
      else {
        lines.push(`${pad}${k}:`)
        lines.push(nested)
      }
      continue
    }
    lines.push(`${pad}${k}: ${yamlScalar(val)}`)
  }
  return lines.join('\n')
}

export type YamlExportKind = 'Proxy' | 'Pool' | 'LoadBalancer' | 'Router' | 'Flow' | 'Entrypoint'

export interface YamlDoc {
  kind: YamlExportKind
  name: string
  body: Record<string, unknown>
}

export function docsToMultiYaml(docs: YamlDoc[]): string {
  return docs.map((d) => {
    const root: Record<string, unknown> = {
      kind: d.kind,
      version: 'v1',
      metadata: { name: d.name },
      spec: d.body,
    }
    return dumpObject(root, 0)
  }).join('\n---\n')
}

export function proxyToYamlDoc(p: import('~/types').Proxy): YamlDoc {
  const spec: Record<string, unknown> = {
    protocol: p.protocol,
    host: p.host,
    port: p.port,
  }
  if (p.auth) spec.auth = { user: p.auth.user, pass: p.auth.pass }
  return { kind: 'Proxy', name: p.id, body: spec }
}

export function poolToYamlDoc(p: import('~/types').Pool): YamlDoc {
  const spec: Record<string, unknown> = { type: p.type }
  if (p.title) spec.title = p.title
  if (p.type === 'static' && p.members?.length) {
    spec.members = p.members.map(m => ({ proxy_id: m.proxy_id, weight: m.weight }))
  }
  if (p.type === 'dynamic' && p.selector?.allow) {
    spec.selector = { allow: p.selector.allow }
  }
  return { kind: 'Pool', name: p.id, body: spec }
}

export function balancerToYamlDoc(b: import('~/types').LoadBalancer): YamlDoc {
  const spec: Record<string, unknown> = {
    type: b.type,
    pool_id: b.pool_id,
  }
  if (b.title) spec.title = b.title
  if (b.type === 'least-bytes' && b.reset_interval) {
    spec.reset_interval = b.reset_interval
  }
  return { kind: 'LoadBalancer', name: b.id, body: spec }
}

export function routerToYamlDoc(r: import('~/types').Router): YamlDoc {
  const spec: Record<string, unknown> = {
    rules: (r.rules || []).map(rule => ({
      id: rule.id,
      target: rule.target,
      match: rule.match,
    })),
  }
  if (r.title) spec.title = r.title
  if (r.description) spec.description = r.description
  return { kind: 'Router', name: r.id, body: spec }
}

export function flowToYamlDoc(f: import('~/types').Flow): YamlDoc {
  const spec: Record<string, unknown> = {}
  if (f.router_id) spec.router_id = f.router_id
  if (f.balancer_id) spec.balancer_id = f.balancer_id
  return { kind: 'Flow', name: f.id, body: spec }
}

export function entrypointToYamlDoc(e: import('~/types').Entrypoint): YamlDoc {
  const spec: Record<string, unknown> = {
    protocol: e.protocol,
    host: e.host,
    port: e.port,
    flow_id: e.flow_id,
  }
  if (e.title) spec.title = e.title
  return { kind: 'Entrypoint', name: e.id, body: spec }
}
