<script setup lang="ts">
import type { ManagedUser, UserRole } from '~/types'

definePageMeta({
  middleware: ['admin'],
})

const { items, totalCount, nextCursor, loading, error, pageSize, refresh, remove } = useUsers()
const toast = useAppToast()
const auth = useAuth()

const search = ref('')
const roleFilter = ref<UserRole | ''>('')
const pageToken = ref<string | undefined>(undefined)
const cursorStack = ref<string[]>([])

const formOpen = ref(false)
const deleteOpen = ref(false)
const deleting = ref<ManagedUser | null>(null)
const deleteError = ref<string | null>(null)
const deleteLoading = ref(false)

const resetOpen = ref(false)
const resetting = ref<ManagedUser | null>(null)

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
    role: roleFilter.value,
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

watch(roleFilter, () => {
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

function openCreate() {
  formOpen.value = true
}

function openDelete(u: ManagedUser) {
  deleting.value = u
  deleteError.value = null
  deleteOpen.value = true
}

function openReset(u: ManagedUser) {
  resetting.value = u
  resetOpen.value = true
}

async function onCreated() {
  await reloadFirst().catch(() => {})
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
    toast.success(`User "${name}" deleted`)
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

function onResetDone() {
  toast.success('Password reset')
}
</script>

<template>
  <div class="space-y-6">
    <section class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
      <div>
        <p class="text-xs font-bold uppercase tracking-widest text-primary mb-1">
          Admin
        </p>
        <div class="flex items-center gap-3">
          <h2 class="text-3xl font-black tracking-tighter text-on-surface">
            Users
          </h2>
          <span class="text-xs font-bold text-on-surface-variant bg-surface-container-highest border border-outline-variant/20 rounded-full min-w-7 h-7 px-1.5 inline-flex items-center justify-center">
            {{ totalCount }}
          </span>
        </div>
        <p class="text-sm text-on-surface-variant mt-1 max-w-xl">
          Create and manage dashboard accounts. Only admins can access this page.
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
        Add User
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
          placeholder="Search by username..."
          class="w-full bg-surface-container-low border border-outline-variant/15 focus:border-primary focus:ring-4 focus:ring-primary/10 rounded-lg pl-10 pr-4 py-2.5 text-sm outline-none transition-all text-on-surface placeholder:text-outline/50"
        >
      </div>
      <div class="grid grid-cols-3 gap-1 p-1 rounded-lg bg-surface-container-lowest border border-outline-variant/15 self-start">
        <button
          type="button"
          class="px-2.5 py-2 text-xs font-bold rounded-md transition-colors"
          :class="roleFilter === '' ? 'bg-surface-container-highest text-primary shadow-sm' : 'text-on-surface-variant hover:text-on-surface'"
          @click="roleFilter = ''"
        >
          All
        </button>
        <button
          type="button"
          class="px-2.5 py-2 text-xs font-bold rounded-md transition-colors"
          :class="roleFilter === 'admin' ? 'bg-surface-container-highest text-primary shadow-sm' : 'text-on-surface-variant hover:text-on-surface'"
          @click="roleFilter = 'admin'"
        >
          Admin
        </button>
        <button
          type="button"
          class="px-2.5 py-2 text-xs font-bold rounded-md transition-colors"
          :class="roleFilter === 'member' ? 'bg-surface-container-highest text-primary shadow-sm' : 'text-on-surface-variant hover:text-on-surface'"
          @click="roleFilter = 'member'"
        >
          Member
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
                Username
              </th>
              <th class="px-6 py-4 text-[10px] font-black uppercase tracking-widest text-on-surface-variant">
                Role
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
              <td colspan="4" class="px-6 py-12 text-center text-sm text-on-surface-variant">
                Loading users…
              </td>
            </tr>
            <tr v-else-if="!items.length">
              <td colspan="4" class="px-6 py-12 text-center text-sm text-on-surface-variant">
                {{ search || roleFilter ? 'No users match your filters.' : 'No users yet.' }}
              </td>
            </tr>
            <tr
              v-for="u in items"
              :key="u.id"
              class="border-b border-outline-variant/10 last:border-0 hover:bg-white/[0.02] transition-colors"
            >
              <td class="px-6 py-4">
                <span class="font-mono text-sm font-semibold text-on-surface">{{ u.id }}</span>
                <span
                  v-if="u.id === auth.user.value?.id"
                  class="ml-2 text-[10px] font-bold uppercase tracking-wider text-on-surface-variant"
                >you</span>
              </td>
              <td class="px-6 py-4">
                <span
                  class="text-[11px] font-bold uppercase tracking-wide px-2 py-1 rounded"
                  :class="u.role === 'admin' ? 'text-primary bg-primary/10' : 'text-on-surface-variant bg-white/5'"
                >
                  {{ u.role }}
                </span>
              </td>
              <td class="px-6 py-4 text-sm text-on-surface-variant whitespace-nowrap">
                {{ formatCreated(u.created_at) }}
              </td>
              <td class="px-6 py-4">
                <div class="flex items-center justify-end gap-1">
                  <button
                    type="button"
                    class="p-2 rounded-lg text-outline hover:text-primary hover:bg-primary/10 transition-colors"
                    title="Reset password"
                    aria-label="Reset password"
                    @click="openReset(u)"
                  >
                    <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" aria-hidden="true">
                      <rect x="3" y="11" width="18" height="11" rx="2" />
                      <path d="M7 11V7a5 5 0 0110 0v4" />
                    </svg>
                  </button>
                  <button
                    type="button"
                    class="p-2 rounded-lg text-outline hover:text-error hover:bg-error/10 transition-colors"
                    title="Delete"
                    aria-label="Delete user"
                    @click="openDelete(u)"
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
          Showing {{ showingFrom }} to {{ showingTo }} of {{ totalCount }} users
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

    <OrganismsUserFormDialog
      :open="formOpen"
      @close="formOpen = false"
      @created="onCreated"
    />

    <OrganismsUserDeleteDialog
      :open="deleteOpen"
      :name="deleting?.id || ''"
      :error="deleteError"
      :loading="deleteLoading"
      @close="deleteOpen = false"
      @confirm="confirmDelete"
    />

    <OrganismsUserResetPasswordDialog
      :open="resetOpen"
      :username="resetting?.id || ''"
      @close="resetOpen = false; resetting = null"
      @reset="onResetDone"
    />
  </div>
</template>
