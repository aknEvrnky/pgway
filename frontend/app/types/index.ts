export interface Proxy {
  id: string
  protocol: 'http' | 'https' | 'socks5'
  host: string
  port: number
  auth?: { user: string; pass: string }
  labels?: Record<string, string>
}

export interface Pool {
  id: string
  title: string
  type: 'static' | 'dynamic'
  labels?: Record<string, string>
  proxyIds?: string[]
  selector?: { allow: Record<string, string> }
}

export interface LoadBalancer {
  id: string
  title: string
  type: 'round-robin' | 'weighted' | 'least-bytes'
  poolId: string
}

export interface Router {
  id: string
  title: string
  description?: string
  rules: RouterRule[]
}

export interface RouterRule {
  id: string
  match: RouterMatch
  target: string
}

export interface RouterMatch {
  type?: 'host' | 'host_suffix' | 'path_prefix' | 'path_regex' | 'method' | 'header' | 'catch_all'
  value?: string
  all?: { type: string; value: string }[]
  any?: { type: string; value: string }[]
}

export interface Flow {
  id: string
  routerId?: string
  balancerId?: string
}

export interface Entrypoint {
  id: string
  title: string
  protocol: string
  host: string
  port: number
  flowId: string
}

export interface MetricCardData {
  label: string
  value: string
  icon: string
  trend?: { direction: 'up' | 'down'; percentage: number }
  badge?: { text: string; variant: 'primary' | 'warning' | 'info' }
  subtitle?: string
  progress?: number
}

export interface ActivityEvent {
  id: string
  type: 'success' | 'delete' | 'warning' | 'info'
  title: string
  description: string
  timestamp: string
  actor: string
}

export interface LogLine {
  timestamp: string
  level: 'info' | 'warn' | 'error'
  message: string
}

export interface HealthDistribution {
  healthy: number
  degraded: number
  down: number
}
