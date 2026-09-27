<script setup lang="ts">
import type { UserRole } from '~/types'

const props = defineProps<{
  open: boolean
}>()

const emit = defineEmits<{
  close: []
  created: [generatedPassword?: string]
}>()

const { create } = useUsers()
const toast = useAppToast()

const username = ref('')
const password = ref('')
const role = ref<UserRole>('member')
const formError = ref('')
const saving = ref(false)
const revealedPassword = ref('')

watch(
  () => props.open,
  (open) => {
    if (!open) return
    username.value = ''
    password.value = ''
    role.value = 'member'
    formError.value = ''
    saving.value = false
    revealedPassword.value = ''
  },
)

async function onSubmit() {
  formError.value = ''
  const name = username.value.trim()
  if (!name) {
    formError.value = 'username is required'
    return
  }

  saving.value = true
  try {
    const res = await create({
      username: name,
      password: password.value || undefined,
      role: role.value,
    })
    if (res.generated_password) {
      revealedPassword.value = res.generated_password
      emit('created', res.generated_password)
      return
    }
    emit('created')
    emit('close')
  }
  catch (e: unknown) {
    formError.value = e instanceof Error ? e.message : 'create failed'
  }
  finally {
    saving.value = false
  }
}

async function copyPassword() {
  if (!revealedPassword.value || !import.meta.client) return
  try {
    await navigator.clipboard.writeText(revealedPassword.value)
    toast.success('Password copied')
  }
  catch {
    toast.error('Could not copy password')
  }
}

function doneWithPassword() {
  revealedPassword.value = ''
  emit('close')
}
</script>

<template>
  <Teleport to="body">
    <div
      v-if="open"
      class="fixed inset-0 z-[110] flex items-center justify-center p-4"
      role="dialog"
      aria-modal="true"
      aria-label="Create user"
    >
      <div class="absolute inset-0 bg-black/60 backdrop-blur-sm" @click="!saving && emit('close')" />
      <div class="relative w-full max-w-md glass-panel rounded-xl border border-outline-variant/20 shadow-2xl p-6 space-y-4">
        <h2 class="text-lg font-black text-on-surface tracking-tight">
          {{ revealedPassword ? 'Temporary password' : 'Create user' }}
        </h2>

        <template v-if="revealedPassword">
          <p class="text-sm text-on-surface-variant">
            Copy this password now — it will not be shown again.
          </p>
          <div class="flex items-center gap-2">
            <code class="flex-1 text-sm font-mono bg-surface-container-highest border border-outline-variant/20 rounded-lg px-3 py-2 text-primary break-all">
              {{ revealedPassword }}
            </code>
            <button
              type="button"
              class="text-xs font-bold text-primary hover:underline px-2 py-2"
              @click="copyPassword"
            >
              Copy
            </button>
          </div>
          <div class="flex justify-end pt-2">
            <button
              type="button"
              class="bg-gradient-to-br from-primary to-primary-container text-on-primary-container font-bold px-6 py-2.5 rounded-xl"
              @click="doneWithPassword"
            >
              Done
            </button>
          </div>
        </template>

        <template v-else>
          <label class="block space-y-1.5">
            <span class="text-[10px] font-black uppercase tracking-widest text-on-surface-variant">Username</span>
            <input
              v-model="username"
              type="text"
              autocomplete="off"
              class="w-full bg-surface-container-low border border-outline-variant/15 focus:border-primary focus:ring-4 focus:ring-primary/10 rounded-lg px-3 py-2.5 text-sm outline-none text-on-surface"
              placeholder="alice"
            >
          </label>

          <label class="block space-y-1.5">
            <span class="text-[10px] font-black uppercase tracking-widest text-on-surface-variant">Role</span>
            <select
              v-model="role"
              class="w-full bg-surface-container-low border border-outline-variant/15 focus:border-primary focus:ring-4 focus:ring-primary/10 rounded-lg px-3 py-2.5 text-sm outline-none text-on-surface"
            >
              <option value="member">
                member
              </option>
              <option value="admin">
                admin
              </option>
            </select>
          </label>

          <label class="block space-y-1.5">
            <span class="text-[10px] font-black uppercase tracking-widest text-on-surface-variant">Password (optional)</span>
            <input
              v-model="password"
              type="password"
              autocomplete="new-password"
              class="w-full bg-surface-container-low border border-outline-variant/15 focus:border-primary focus:ring-4 focus:ring-primary/10 rounded-lg px-3 py-2.5 text-sm outline-none text-on-surface"
              placeholder="Leave empty to generate"
            >
          </label>

          <p v-if="formError" class="text-error text-sm" role="alert">
            {{ formError }}
          </p>

          <div class="flex justify-end gap-3 pt-2">
            <button
              type="button"
              class="text-sm font-bold text-on-surface-variant hover:text-on-surface px-4 py-2"
              :disabled="saving"
              @click="emit('close')"
            >
              Cancel
            </button>
            <button
              type="button"
              class="bg-gradient-to-br from-primary to-primary-container text-on-primary-container font-bold px-6 py-2.5 rounded-xl hover:opacity-90 active:scale-[0.98] transition-all disabled:opacity-60"
              :disabled="saving"
              @click="onSubmit"
            >
              {{ saving ? 'Creating…' : 'Create' }}
            </button>
          </div>
        </template>
      </div>
    </div>
  </Teleport>
</template>
