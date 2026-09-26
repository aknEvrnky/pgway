<script setup lang="ts">
import type { Proxy } from '~/types'

const { items, loading, error, refresh, remove } = useProxies()

const search = ref('')
const formOpen = ref(false)
const formMode = ref<'create' | 'edit'>('create')
const editing = ref<Proxy | null>(null)

const deleteOpen = ref(false)
const deleting = ref<Proxy | null>(null)
const deleteError = ref<string | null>(null)
const deleteLoading = ref(false)

onMounted(() => {
  refresh().catch(() => {})
})

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return items.value
  return items.value.filter((p) => {
    const labelBlob = Object.entries(p.labels || {})
      .map(([k, v]) => `${k}:${v}`)
      .join(' ')
      .toLowerCase()
    return (
      p.id.toLowerCase().includes(q)
      || p.host.toLowerCase().includes(q)
      || p.protocol.toLowerCase().includes(q)
      || `${p.host}:${p.port}`.includes(q)
      || labelBlob.includes(q)
    )
  })
})

function formatCreated(iso?: string) {
  if (!iso) return '—'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return '—'
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' })
}

function openCreate() {
  formMode.value = 'create'
  editing.value = null
  formOpen.value = true
}

function openEdit(p: Proxy) {
  formMode.value = 'edit'
  editing.value = p
  formOpen.value = true
}

function openDelete(p: Proxy) {
  deleting.value = p
  deleteError.value = null
  deleteOpen.value = true
}

async function confirmDelete() {
  if (!deleting.value) return
  deleteLoading.value = true
  deleteError.value = null
  try {
    await remove(deleting.value.id)
    deleteOpen.value = false
    deleting.value = null
  }
  catch (e: unknown) {
    deleteError.value = e instanceof Error ? e.message : 'delete failed'
  }
  finally {
    deleteLoading.value = false
  }
}

async function onSaved() {
  // refresh already done in apply; keep for clarity
}
</script>

<template>
  <div class="space-y-6">
    <section class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
      <div>
        <p class="text-xs font-bold uppercase tracking-widest text-primary mb-1">
          Resources
        </p>
        <h2 class="text-3xl font-black tracking-tighter text-on-surface">
          Proxies
        </h2>
        <p class="text-sm text-on-surface-variant mt-1 max-w-xl">
          Manage outbound upstream proxies, authentication credentials, and labels.
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
        Add Proxy
      </button>
    </section>

    <div class="relative max-w-md">
      <svg class="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-outline" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
        <circle cx="11" cy="11" r="8" />
        <line x1="21" y1="21" x2="16.65" y2="16.65" stroke-linecap="round" />
      </svg>
      <input
        v-model="search"
        type="search"
        placeholder="Search by name, host, or label..."
        class="w-full bg-surface-container-low border border-outline-variant/15 focus:border-primary focus:ring-4 focus:ring-primary/10 rounded-lg pl-10 pr-4 py-2.5 text-sm outline-none transition-all text-on-surface placeholder:text-outline/50"
      >
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
                Protocol
              </th>
              <th class="px-6 py-4 text-[10px] font-black uppercase tracking-widest text-on-surface-variant">
                Host:Port
              </th>
              <th class="px-6 py-4 text-[10px] font-black uppercase tracking-widest text-on-surface-variant">
                Auth
              </th>
              <th class="px-6 py-4 text-[10px] font-black uppercase tracking-widest text-on-surface-variant">
                Labels
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
              <td colspan="7" class="px-6 py-12 text-center text-sm text-on-surface-variant">
                Loading proxies…
              </td>
            </tr>
            <tr v-else-if="!filtered.length">
              <td colspan="7" class="px-6 py-12 text-center text-sm text-on-surface-variant">
                {{ search ? 'No proxies match your search.' : 'No proxies yet. Add one to get started.' }}
              </td>
            </tr>
            <tr
              v-for="p in filtered"
              :key="p.id"
              class="border-b border-outline-variant/10 last:border-0 hover:bg-white/[0.02] transition-colors"
            >
              <td class="px-6 py-4">
                <span class="font-mono text-sm font-semibold text-on-surface">{{ p.id }}</span>
              </td>
              <td class="px-6 py-4">
                <span class="text-[11px] font-bold uppercase tracking-wide text-primary bg-primary/10 px-2 py-1 rounded">
                  {{ p.protocol }}
                </span>
              </td>
              <td class="px-6 py-4 font-mono text-sm text-on-surface-variant">
                {{ p.host }}:{{ p.port }}
              </td>
              <td class="px-6 py-4">
                <span
                  class="text-[10px] font-black uppercase tracking-wider px-2 py-1 rounded"
                  :class="p.auth ? 'text-tertiary bg-tertiary/10' : 'text-outline bg-white/5'"
                >
                  {{ p.auth ? 'Yes' : 'No' }}
                </span>
              </td>
              <td class="px-6 py-4">
                <div v-if="p.labels && Object.keys(p.labels).length" class="flex flex-wrap gap-1.5">
                  <span
                    v-for="(val, key) in p.labels"
                    :key="`${key}-${val}`"
                    class="text-[10px] font-mono px-2 py-0.5 rounded bg-surface-container-highest text-on-surface-variant border border-outline-variant/20"
                  >
                    {{ key }}={{ val }}
                  </span>
                </div>
                <span v-else class="text-outline text-xs">—</span>
              </td>
              <td class="px-6 py-4 text-sm text-on-surface-variant whitespace-nowrap">
                {{ formatCreated(p.created_at) }}
              </td>
              <td class="px-6 py-4">
                <div class="flex items-center justify-end gap-1">
                  <button
                    type="button"
                    class="p-2 rounded-lg text-outline hover:text-primary hover:bg-primary/10 transition-colors"
                    title="Edit"
                    aria-label="Edit proxy"
                    @click="openEdit(p)"
                  >
                    <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" aria-hidden="true">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M12 20h9" />
                      <path stroke-linecap="round" stroke-linejoin="round" d="M16.5 3.5a2.12 2.12 0 013 3L7 19l-4 1 1-4L16.5 3.5z" />
                    </svg>
                  </button>
                  <button
                    type="button"
                    class="p-2 rounded-lg text-outline hover:text-error hover:bg-error/10 transition-colors"
                    title="Delete"
                    aria-label="Delete proxy"
                    @click="openDelete(p)"
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
    </div>

    <OrganismsProxyFormDialog
      :open="formOpen"
      :mode="formMode"
      :initial="editing"
      @close="formOpen = false"
      @saved="onSaved"
    />

    <OrganismsProxyDeleteDialog
      :open="deleteOpen"
      :name="deleting?.id || ''"
      :error="deleteError"
      :loading="deleteLoading"
      @close="deleteOpen = false"
      @confirm="confirmDelete"
    />
  </div>
</template>
