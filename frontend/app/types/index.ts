export interface Proxy {
  id: string
  protocol: 'http' | 'https' | 'socks5'
  host: string
  port: number
  auth?: { user: string; pass: string }
  labels?: Record<string, string>
  created_at?: string
  updated_at?: string
}

export type ProxyProtocol = Proxy['protocol']

export interface ProxyListQuery {
  search?: string
  protocol?: ProxyProtocol | ''
  page_size?: number
  page_token?: string
}

export interface ProxyListResponse {
  items: Proxy[]
  next_cursor?: string
  total_count: number
}

export interface ProxyApplyRequest {
  metadata: {
    name: string
    labels?: Record<string, string>
  }
  spec: {
    url?: string
    protocol?: string
    host?: string
    port?: number
    auth?: { user: string; pass: string }
  }
}

export interface PoolMember {
  proxy_id: string
  weight: number
}

export interface Pool {
  id: string
  title?: string
  type: 'static' | 'dynamic'
  labels?: Record<string, string>
  members?: PoolMember[]
  selector?: { allow?: Record<string, string> }
  created_at?: string
  updated_at?: string
}

export type PoolType = Pool['type']

export interface PoolListQuery {
  search?: string
  type?: PoolType | ''
  page_size?: number
  page_token?: string
}

export interface PoolListResponse {
  items: Pool[]
  next_cursor?: string
  total_count: number
}

export interface PoolApplyRequest {
  metadata: {
    name: string
    labels?: Record<string, string>
  }
  spec: {
    title?: string
    type: PoolType
    members?: { proxy_id: string; weight?: number }[]
    selector?: { allow: Record<string, string> }
  }
}

export interface LoadBalancer {
  id: string
  title?: string
  type: 'round-robin' | 'weighted' | 'least-bytes'
  pool_id: string
  reset_interval?: string
  created_at?: string
  updated_at?: string
}

export type BalancerType = LoadBalancer['type']

export interface BalancerListQuery {
  search?: string
  type?: BalancerType | ''
  pool_id?: string
  page_size?: number
  page_token?: string
}

export interface BalancerListResponse {
  items: LoadBalancer[]
  next_cursor?: string
  total_count: number
}

export interface BalancerApplyRequest {
  metadata: {
    name: string
  }
  spec: {
    title?: string
    type: BalancerType
    pool_id: string
    reset_interval?: string
  }
}

export type MatchType = 'host' | 'host_suffix' | 'path_prefix' | 'path_regex' | 'method' | 'header' | 'catch_all'

export interface RouterCondition {
  type: MatchType
  value: string
}

export interface Router {
  id: string
  title?: string
  description?: string
  rules: RouterRule[]
  created_at?: string
  updated_at?: string
}

export interface RouterRule {
  id: string
  match: RouterMatch
  target: string
}

export interface RouterMatch {
  type?: MatchType
  value?: string
  all?: RouterCondition[]
  any?: RouterCondition[]
  not?: RouterCondition
}

export interface RouterListQuery {
  search?: string
  has_catch_all?: boolean | ''
  target?: string
  page_size?: number
  page_token?: string
}

export interface RouterListResponse {
  items: Router[]
  next_cursor?: string
  total_count: number
}

export interface RouterApplyRequest {
  metadata: {
    name: string
  }
  spec: {
    title?: string
    description?: string
    rules: {
      id: string
      match: RouterMatch
      target: string
    }[]
  }
}

export interface Flow {
  id: string
  router_id?: string
  balancer_id?: string
  created_at?: string
  updated_at?: string
}

export type FlowModeFilter = '' | 'router' | 'direct'

export interface FlowListQuery {
  search?: string
  mode?: FlowModeFilter
  router_id?: string
  balancer_id?: string
  page_size?: number
  page_token?: string
}

export interface FlowListResponse {
  items: Flow[]
  next_cursor?: string
  total_count: number
}

export interface FlowApplyRequest {
  metadata: { name: string }
  spec: {
    router_id?: string
    balancer_id?: string
  }
}

export interface Entrypoint {
  id: string
  title?: string
  protocol: string
  host: string
  port: number
  flow_id: string
  created_at?: string
  updated_at?: string
}

export interface EntrypointListQuery {
  search?: string
  protocol?: string
  host?: string
  flow_id?: string
  page_size?: number
  page_token?: string
}

export interface EntrypointListResponse {
  items: Entrypoint[]
  next_cursor?: string
  total_count: number
}

export interface EntrypointApplyRequest {
  metadata: { name: string }
  spec: {
    title?: string
    protocol: string
    host: string
    port: number
    flow_id: string
  }
}

export type UserRole = 'admin' | 'member'

export interface ManagedUser {
  id: string
  role: UserRole
  created_at?: string
  updated_at?: string
}

export interface UserListQuery {
  search?: string
  role?: UserRole | ''
  page_size?: number
  page_token?: string
}

export interface UserListResponse {
  items: ManagedUser[]
  next_cursor?: string
  total_count: number
}

export interface CreateUserRequest {
  username: string
  password?: string
  role?: UserRole
}

export interface CreateUserResponse {
  user: ManagedUser
  generated_password?: string
}

export interface ChangeUserPasswordRequest {
  old_password?: string
  new_password?: string
}

export type FlowGraphKind = 'entrypoint' | 'flow' | 'router' | 'balancer' | 'pool' | 'proxy'

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
