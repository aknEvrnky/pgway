<script setup lang="ts">
const props = defineProps<{
  open: boolean
  username: string
}>()

const emit = defineEmits<{
  close: []
  reset: []
}>()

const { changePassword } = useUsers()

const newPassword = ref('')
const formError = ref('')
const saving = ref(false)

watch(
  () => props.open,
  (open) => {
    if (!open) return
    newPassword.value = ''
    formError.value = ''
    saving.value = false
  },
)

async function onSubmit() {
  formError.value = ''
  if (!newPassword.value) {
    formError.value = 'new password is required'
    return
  }
  if (newPassword.value.length < 8) {
    formError.value = 'password must be at least 8 characters'
    return
  }

  saving.value = true
  try {
    await changePassword(props.username, {
      new_password: newPassword.value,
    })
    emit('reset')
    emit('close')
  }
  catch (e: unknown) {
    formError.value = e instanceof Error ? e.message : 'reset failed'
  }
  finally {
    saving.value = false
  }
}
</script>

<template>
  <Teleport to="body">
    <div
      v-if="open"
      class="fixed inset-0 z-[110] flex items-center justify-center p-4"
      role="dialog"
      aria-modal="true"
      aria-label="Reset password"
    >
      <div class="absolute inset-0 bg-black/60 backdrop-blur-sm" @click="!saving && emit('close')" />
      <div class="relative w-full max-w-md glass-panel rounded-xl border border-outline-variant/20 shadow-2xl p-6 space-y-4">
        <h2 class="text-lg font-black text-on-surface tracking-tight">
          Reset password
        </h2>

        <p class="text-sm text-on-surface-variant">
          Set a new password for <span class="font-mono text-primary">{{ username }}</span>. All of their sessions will be revoked.
        </p>

        <label class="block space-y-1.5">
          <span class="text-[10px] font-black uppercase tracking-widest text-on-surface-variant">New password</span>
          <input
            v-model="newPassword"
            type="password"
            autocomplete="new-password"
            class="w-full bg-surface-container-low border border-outline-variant/15 focus:border-primary focus:ring-4 focus:ring-primary/10 rounded-lg px-3 py-2.5 text-sm outline-none text-on-surface"
            placeholder="At least 8 characters"
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
            {{ saving ? 'Resetting…' : 'Reset' }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>
