<script setup lang="ts">
import type { Flow, FlowModeFilter } from '~/types'

const { items, totalCount, nextCursor, loading, error, pageSize, refresh, remove } = useFlows()
const toast = useAppToast()

const search = ref('')
const modeFilter = ref<FlowModeFilter>('')
const pageToken = ref<string | undefined>(undefined)
const cursorStack = ref<string[]>([])

const deleteOpen = ref(false)
const deleting = ref<Flow | null>(null)
const deleteError = ref<string | null>(null)
const deleteLoading = ref(false)

const pageIndex = computed(() => cursorStack.value.length)
const showingFrom = computed(() => {
  if (!totalCount.value || !items.value.length) return 0
  return pageIndex.value * pageSize + 1
})
const showingTo = computed(() => {
  if (!items.value.length) return 0
  return pageIndex.value * pageSize + items.value.length
})
const canPrev = computed(() => pageIndex.value > 0)
const canNext = computed(() => !!nextCursor.value)

async function loadPage(token?: string) {
  await refresh({
    search: search.value,
    mode: modeFilter.value,
    page_size: pageSize,
    page_token: token,
  })
}

function resetToFirstPage() {
  cursorStack.value = []
  pageToken.value = undefined
}

async function reloadFirst() {
  resetToFirstPage()
  await loadPage(undefined)
}

onMounted(() => {
  reloadFirst().catch(() => {})
})

let searchTimer: ReturnType<typeof setTimeout> | null = null
watch(search, () => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    reloadFirst().catch(() => {})
  }, 250)
})

watch(modeFilter, () => {
  reloadFirst().catch(() => {})
})

async function goNext() {
  if (!nextCursor.value) return
  cursorStack.value.push(pageToken.value || '')
  pageToken.value = nextCursor.value
  await loadPage(pageToken.value)
}

async function goPrev() {
  if (!canPrev.value) return
  const prev = cursorStack.value.pop()
  pageToken.value = prev || undefined
  await loadPage(pageToken.value)
}

function formatCreated(iso?: string) {
  if (!iso) return '—'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return '—'
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' })
}

function isRouterMode(f: Flow): boolean {
  return !!f.router_id
}

function openCreate() {
  navigateTo('/flows/new')
}

function openDelete(f: Flow) {
  deleting.value = f
  deleteError.value = null
  deleteOpen.value = true
}

async function confirmDelete() {
  if (!deleting.value) return
  deleteLoading.value = true
  deleteError.value = null
  try {
    const name = deleting.value.id
    await remove(name)
    deleteOpen.value = false
    deleting.value = null
    toast.success(`Flow "${name}" deleted`)
    await reloadFirst()
  }
  catch (e: unknown) {
    const msg = e instanceof Error ? e.message : 'delete failed'
    deleteError.value = msg
    toast.error(msg)
  }
  finally {
    deleteLoading.value = false
  }
}
</script>

<template>
  <div class="space-y-6">
    <section class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
      <div>
        <p class="text-xs font-bold uppercase tracking-widest text-primary mb-1">
          Network
        </p>
        <div class="flex items-center gap-3">
          <h2 class="text-3xl font-black tracking-tighter text-on-surface">
            Flows
          </h2>
          <span class="text-xs font-bold text-on-surface-variant bg-surface-container-highest border border-outline-variant/20 rounded-full min-w-7 h-7 px-1.5 inline-flex items-center justify-center">
            {{ totalCount }}
          </span>
        </div>
        <p class="text-sm text-on-surface-variant mt-1 max-w-xl">
          Open the visual editor to compose a pipeline. Add Flow starts a local draft — nothing is stored until the flow is valid.
        </p>
      </div>
      <button
        type="button"
        class="bg-gradient-to-br from-primary to-primary-container text-on-primary-container font-bold px-5 py-2.5 rounded-xl shadow-lg shadow-primary/10 hover:shadow-primary/20 active:scale-[0.98] transition-all flex items-center gap-2 self-start sm:self-auto"
        @click="openCreate"
      >
        <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true">
          <line x1="12" y1="5" x2="12" y2="19" stroke-linecap="round" />
          <line x1="5" y1="12" x2="19" y2="12" stroke-linecap="round" />
        </svg>
        Add Flow
      </button>
    </section>

    <div class="flex flex-col gap-3 sm:flex-row sm:items-center">
      <div class="relative flex-1 max-w-md">
        <svg class="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-outline" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
          <circle cx="11" cy="11" r="8" />
          <line x1="21" y1="21" x2="16.65" y2="16.65" stroke-linecap="round" />
        </svg>
        <input
          v-model="search"
          type="search"
          placeholder="Search by name, router, or balancer..."
          class="w-full bg-surface-container-low border border-outline-variant/15 focus:border-primary focus:ring-4 focus:ring-primary/10 rounded-lg pl-10 pr-4 py-2.5 text-sm outline-none transition-all text-on-surface placeholder:text-outline/50"
        >
      </div>
      <div class="grid grid-cols-3 gap-1 p-1 rounded-lg bg-surface-container-lowest border border-outline-variant/15 self-start">
        <button
          type="button"
          class="px-3 py-2 text-xs font-bold rounded-md transition-colors"
          :class="modeFilter === '' ? 'bg-surface-container-highest text-primary shadow-sm' : 'text-on-surface-variant hover:text-on-surface'"
          @click="modeFilter = ''"
        >
          All
        </button>
        <button
          type="button"
          class="px-3 py-2 text-xs font-bold rounded-md transition-colors"
          :class="modeFilter === 'router' ? 'bg-surface-container-highest text-primary shadow-sm' : 'text-on-surface-variant hover:text-on-surface'"
          @click="modeFilter = 'router'"
        >
          Has router
        </button>
        <button
          type="button"
          class="px-3 py-2 text-xs font-bold rounded-md transition-colors"
          :class="modeFilter === 'direct' ? 'bg-surface-container-highest text-primary shadow-sm' : 'text-on-surface-variant hover:text-on-surface'"
          @click="modeFilter = 'direct'"
        >
          Direct balancer
        </button>
      </div>
    </div>

    <p v-if="error" class="text-error text-sm" role="alert">
      {{ error }}
    </p>

    <div class="glass-panel rounded-xl border border-outline-variant/15 overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-left">
          <thead>
            <tr class="border-b border-outline-variant/15">
              <th class="px-6 py-4 text-[10px] font-black uppercase tracking-widest text-on-surface-variant">
                Name
              </th>
              <th class="px-6 py-4 text-[10px] font-black uppercase tracking-widest text-on-surface-variant">
                Preview
              </th>
              <th class="px-6 py-4 text-[10px] font-black uppercase tracking-widest text-on-surface-variant">
                Mode
              </th>
              <th class="px-6 py-4 text-[10px] font-black uppercase tracking-widest text-on-surface-variant">
                Created
              </th>
              <th class="px-6 py-4 text-[10px] font-black uppercase tracking-widest text-on-surface-variant text-right">
                Actions
              </th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading && !items.length">
              <td colspan="5" class="px-6 py-12 text-center text-sm text-on-surface-variant">
                Loading flows…
              </td>
            </tr>
            <tr v-else-if="!items.length">
              <td colspan="5" class="px-6 py-12 text-center text-sm text-on-surface-variant">
                {{ search || modeFilter ? 'No flows match your filters.' : 'No flows yet. Add one to get started.' }}
              </td>
            </tr>
            <tr
              v-for="f in items"
              :key="f.id"
              class="border-b border-outline-variant/10 last:border-0 hover:bg-white/[0.02] transition-colors"
            >
              <td class="px-6 py-4">
                <NuxtLink
                  :to="`/flows/${encodeURIComponent(f.id)}`"
                  class="font-mono text-sm font-semibold text-primary hover:underline"
                >
                  {{ f.id }}
                </NuxtLink>
              </td>
              <td class="px-6 py-4">
                <MoleculesFlowPreview :flow="f" />
              </td>
              <td class="px-6 py-4">
                <span
                  class="text-[11px] font-bold uppercase tracking-wide px-2 py-1 rounded border"
                  :class="isRouterMode(f)
                    ? 'text-primary border-primary/40 bg-primary/10'
                    : 'text-amber-200 border-amber-500/40 bg-amber-500/15'"
                >
                  {{ isRouterMode(f) ? 'router' : 'direct' }}
                </span>
              </td>
              <td class="px-6 py-4 text-sm text-on-surface-variant whitespace-nowrap">
                {{ formatCreated(f.created_at) }}
              </td>
              <td class="px-6 py-4">
                <div class="flex items-center justify-end gap-1">
                  <NuxtLink
                    :to="`/flows/${encodeURIComponent(f.id)}`"
                    class="p-2 rounded-lg text-outline hover:text-primary hover:bg-primary/10 transition-colors"
                    title="Open editor"
                    aria-label="Open flow editor"
                  >
                    <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" aria-hidden="true">
                      <rect x="3" y="3" width="7" height="7" rx="1" />
                      <rect x="14" y="3" width="7" height="7" rx="1" />
                      <rect x="3" y="14" width="7" height="7" rx="1" />
                      <rect x="14" y="14" width="7" height="7" rx="1" />
                    </svg>
                  </NuxtLink>
                  <button
                    type="button"
                    class="p-2 rounded-lg text-outline hover:text-error hover:bg-error/10 transition-colors"
                    title="Delete"
                    aria-label="Delete flow"
                    @click="openDelete(f)"
                  >
                    <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" aria-hidden="true">
                      <polyline points="3 6 5 6 21 6" />
                      <path d="M19 6l-1 14a2 2 0 01-2 2H8a2 2 0 01-2-2L5 6" />
                      <path d="M10 11v6M14 11v6" />
                      <path d="M9 6V4a1 1 0 011-1h4a1 1 0 011 1v2" />
                    </svg>
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div
        v-if="totalCount > 0"
        class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between px-6 py-4 border-t border-outline-variant/15"
      >
        <p class="text-xs text-on-surface-variant">
          Showing {{ showingFrom }} to {{ showingTo }} of {{ totalCount }} flows
        </p>
        <div class="flex items-center gap-2">
          <button
            type="button"
            class="px-3 py-1.5 text-xs font-bold rounded-lg border border-outline-variant/20 text-on-surface-variant hover:text-on-surface hover:border-primary/40 disabled:opacity-40 disabled:pointer-events-none transition-colors"
            :disabled="!canPrev || loading"
            @click="goPrev"
          >
            Prev
          </button>
          <span class="text-xs font-bold text-on-surface-variant min-w-6 text-center">
            {{ pageIndex + 1 }}
          </span>
          <button
            type="button"
            class="px-3 py-1.5 text-xs font-bold rounded-lg border border-outline-variant/20 text-on-surface-variant hover:text-on-surface hover:border-primary/40 disabled:opacity-40 disabled:pointer-events-none transition-colors"
            :disabled="!canNext || loading"
            @click="goNext"
          >
            Next
          </button>
        </div>
      </div>
    </div>

    <div class="flex items-start gap-3 text-sm text-on-surface-variant bg-surface-container-low/60 border border-outline-variant/15 rounded-xl px-4 py-3">
      <svg class="w-4 h-4 mt-0.5 shrink-0 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
        <circle cx="12" cy="12" r="10" />
        <line x1="12" y1="8" x2="12" y2="12" stroke-linecap="round" />
        <circle cx="12" cy="16" r="0.5" fill="currentColor" />
      </svg>
      <p>
        A Flow must reference a Router and/or a Load Balancer. Entrypoints point at a Flow. Delete is blocked while an Entrypoint still references this flow.
      </p>
    </div>

    <OrganismsFlowDeleteDialog
      :open="deleteOpen"
      :name="deleting?.id || ''"
      :error="deleteError"
      :loading="deleteLoading"
      @close="deleteOpen = false"
      @confirm="confirmDelete"
    />
  </div>
</template>
