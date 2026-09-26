/** Apply-order ranks matching schema.SortByApplyOrder. */
export const APPLY_KIND_RANK: Record<string, number> = {
  Proxy: 0,
  Pool: 1,
  LoadBalancer: 2,
  Router: 3,
  Flow: 4,
  Entrypoint: 5,
}

export function sortByApplyOrder<T extends { kind: string }>(items: T[]): T[] {
  return [...items].sort((a, b) => {
    const ra = APPLY_KIND_RANK[a.kind] ?? 99
    const rb = APPLY_KIND_RANK[b.kind] ?? 99
    return ra - rb
  })
}
