<script setup lang="ts">
const props = defineProps<{
  text: string
}>()

const open = ref(false)
const tipStyle = ref<Record<string, string>>({})
const btnRef = ref<HTMLElement | null>(null)

function place() {
  const el = btnRef.value
  if (!el) return
  const r = el.getBoundingClientRect()
  const tipWidth = 256
  const pad = 8
  let left = r.left + r.width / 2 - tipWidth / 2
  left = Math.max(pad, Math.min(left, window.innerWidth - tipWidth - pad))
  const below = r.bottom + 8
  const spaceBelow = window.innerHeight - below
  const preferBelow = spaceBelow > 96
  tipStyle.value = {
    position: 'fixed',
    width: `${tipWidth}px`,
    left: `${left}px`,
    ...(preferBelow
      ? { top: `${below}px` }
      : { bottom: `${window.innerHeight - r.top + 8}px` }),
    zIndex: '200',
  }
}

function show() {
  place()
  open.value = true
}

function hide() {
  open.value = false
}
</script>

<template>
  <span class="inline-flex align-middle">
    <button
      ref="btnRef"
      type="button"
      class="inline-flex items-center justify-center w-4 h-4 rounded-full text-outline hover:text-primary border border-outline-variant/40 hover:border-primary/50 transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
      :aria-label="text"
      :aria-expanded="open"
      @mouseenter="show"
      @mouseleave="hide"
      @focus="show"
      @blur="hide"
    >
      <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.25" aria-hidden="true">
        <circle cx="12" cy="12" r="10" />
        <line x1="12" y1="10" x2="12" y2="16" stroke-linecap="round" />
        <circle cx="12" cy="7" r="0.75" fill="currentColor" stroke="none" />
      </svg>
    </button>
    <Teleport to="body">
      <span
        v-if="open"
        role="tooltip"
        class="pointer-events-none rounded-lg border border-outline-variant/30 bg-surface-container-highest px-3 py-2 text-[11px] leading-relaxed text-on-surface-variant shadow-xl"
        :style="tipStyle"
      >
        {{ props.text }}
      </span>
    </Teleport>
  </span>
</template>
