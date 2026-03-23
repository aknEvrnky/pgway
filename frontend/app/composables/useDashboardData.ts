import type { MetricCardData, ActivityEvent, LogLine, HealthDistribution } from '~/types'

export function useDashboardData() {
  const metrics: MetricCardData[] = [
    {
      label: 'Total Requests',
      value: '3.4M',
      icon: 'analytics',
      trend: { direction: 'up', percentage: 12 },
      progress: 66,
    },
    {
      label: 'Requests / Sec',
      value: '2.5k',
      icon: 'speed',
      subtitle: 'THROUGHPUT PEAK',
    },
    {
      label: 'Active Proxies',
      value: '452/500',
      icon: 'hub',
      badge: { text: '90.4% HEALTH', variant: 'primary' },
    },
    {
      label: 'Active Agents',
      value: '12',
      icon: 'dns',
      badge: { text: 'ALL-IN-ONE', variant: 'warning' },
      subtitle: 'Global Distribution',
    },
  ]

  const activityEvents: ActivityEvent[] = [
    {
      id: '1',
      type: 'success',
      title: 'proxy-1 applied',
      description: 'Routing rules updated successfully for production cluster.',
      timestamp: '2 mins ago',
      actor: 'user:admin',
    },
    {
      id: '2',
      type: 'delete',
      title: 'pool-2 deleted',
      description: 'Resource removed. Orphaned connections re-routed to default pool.',
      timestamp: '15 mins ago',
      actor: 'user:system',
    },
    {
      id: '3',
      type: 'warning',
      title: 'high-latency-alert',
      description: 'p99 latency exceeded 500ms on us-east-1 router.',
      timestamp: '42 mins ago',
      actor: 'agent:04',
    },
    {
      id: '4',
      type: 'success',
      title: 'lb-primary updated',
      description: 'Load balancer strategy changed to weighted round-robin.',
      timestamp: '1 hour ago',
      actor: 'user:admin',
    },
    {
      id: '5',
      type: 'info',
      title: 'entrypoint-8080 created',
      description: 'New HTTP entrypoint listening on 0.0.0.0:8080.',
      timestamp: '2 hours ago',
      actor: 'user:admin',
    },
  ]

  const throughputData = ref([
    { label: '14:00', value: 60 },
    { label: '14:05', value: 45 },
    { label: '14:10', value: 70 },
    { label: '14:15', value: 85 },
    { label: '14:20', value: 55 },
    { label: '14:25', value: 90 },
    { label: '14:30', value: 65 },
    { label: '14:35', value: 40 },
  ])

  const selectedRange = ref<'1H' | '6H' | '24H'>('1H')

  const healthDistribution: HealthDistribution = {
    healthy: 420,
    degraded: 25,
    down: 7,
  }

  const logLines: LogLine[] = [
    { timestamp: '14:20:01', level: 'info', message: 'GET /api/v1/status 200' },
    { timestamp: '14:20:04', level: 'info', message: 'POST /config/apply 202' },
    { timestamp: '14:20:05', level: 'warn', message: 'WARN agent-04 latency_spike' },
    { timestamp: '14:20:06', level: 'info', message: 'GET /api/v1/proxies 200' },
    { timestamp: '14:20:08', level: 'error', message: 'ERR proxy-eu-12 connection_refused' },
    { timestamp: '14:20:09', level: 'info', message: 'GET /api/v1/health 200' },
  ]

  return {
    metrics,
    activityEvents,
    throughputData,
    selectedRange,
    healthDistribution,
    logLines,
  }
}
