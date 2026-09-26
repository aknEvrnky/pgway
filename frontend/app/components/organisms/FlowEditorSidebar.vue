<script setup lang="ts">
import type { Entrypoint, Flow, FlowGraphKind, LoadBalancer, Pool, Proxy, Router } from '~/types'
import type { FlowGraphBundle } from '~/composables/useFlowGraph'
import { validateFlowForStore } from '~/utils/flowValidation'

const props = defineProps<{
  tab: 'settings' | 'yaml'
  yamlText: string
  selectedKind: FlowGraphKind | null
  selectedId: string | null
  bundle: FlowGraphBundle | null
  dirtyCount: number
  deploying: boolean
  draft?: boolean
}>()

const emit = defineEmits<{
  'update:tab': [tab: 'settings' | 'yaml']
  patchFlow: [patch: Partial<Pick<Flow, 'router_id' | 'balancer_id'>>]
  patchFlowName: [name: string]
  edit: [kind: FlowGraphKind]
  deploy: []
}>()

const toast = useAppToast()

const selectedResource = computed(() => {
  if (!props.bundle || !props.selectedKind || !props.selectedId) return null
  const b = props.bundle
  switch (props.selectedKind) {
    case 'flow':
      if (props.selectedId === '__draft__' || b.flow.id === props.selectedId) return b.flow
      return null
    case 'entrypoint': return b.entrypoints.find(e => e.id === props.selectedId) || null
    case 'router': return b.router?.id === props.selectedId ? b.router : null
    case 'balancer': return b.balancers.find(x => x.id === props.selectedId) || null
    case 'pool': return b.pools.find(x => x.id === props.selectedId) || null
    case 'proxy': return b.proxies.find(x => x.id === props.selectedId) || null
    default: return null
  }
})

const flowDraft = computed(() => props.bundle?.flow)

const kindLabel = computed(() => {
  const k = props.selectedKind
  if (!k) return ''
  if (k === 'balancer') return 'Load Balancer'
  return k.charAt(0).toUpperCase() + k.slice(1)
})

const storeError = computed(() => {
  if (!flowDraft.value) return null
  return validateFlowForStore(flowDraft.value)
})

const canDeploy = computed(() => {
  if (props.deploying) return false
  if (props.draft) return !storeError.value
  return props.dirtyCount > 0
})

const deployLabel = computed(() => {
  if (props.deploying) return props.draft ? 'Creating…' : 'Deploying…'
  if (props.draft) return storeError.value ? 'Complete required fields' : 'Create Flow'
  if (props.dirtyCount) return `Deploy Changes (${props.dirtyCount})`
  return 'No Changes'
})

async function copyYaml() {
  try {
    await navigator.clipboard.writeText(props.yamlText)
    toast.success('YAML copied')
  }
  catch {
    toast.error('Copy failed')
  }
}

function onFlowName(e: Event) {
  emit('patchFlowName', (e.target as HTMLInputElement).value.trim())
}

function onFlowRouter(e: Event) {
  const v = (e.target as HTMLInputElement).value.trim()
  emit('patchFlow', { router_id: v || undefined })
}

function onFlowBalancer(e: Event) {
  const v = (e.target as HTMLInputElement).value.trim()
  emit('patchFlow', { balancer_id: v || undefined })
}

function poolSelectorPairs(p: Pool): [string, string][] {
  return Object.entries(p.selector?.allow || {})
}
</script>

<template>
  <aside class="h-full w-80 shrink-0 flex flex-col bg-surface-container-low border-l border-outline-variant/15">
    <div class="flex border-b border-outline-variant/15">
      <button
        type="button"
        class="flex-1 py-3.5 text-[11px] font-bold uppercase tracking-wider transition-colors"
        :class="tab === 'settings' ? 'text-primary border-b-2 border-primary' : 'text-on-surface-variant'"
        @click="emit('update:tab', 'settings')"
      >
        Node Settings
      </button>
      <button
        type="button"
        class="flex-1 py-3.5 text-[11px] font-bold uppercase tracking-wider transition-colors"
        :class="tab === 'yaml' ? 'text-primary border-b-2 border-primary' : 'text-on-surface-variant'"
        @click="emit('update:tab', 'yaml')"
      >
        YAML
      </button>
    </div>

    <div class="flex-1 min-h-0 overflow-y-auto p-5 space-y-6">
      <template v-if="tab === 'settings'">
        <div
          v-if="draft"
          class="rounded-lg border border-primary/25 bg-primary/10 px-3 py-2 text-[11px] text-on-surface-variant"
        >
          Draft — nothing is saved until the flow has a name and a router or balancer.
        </div>

        <div v-if="!selectedKind || !selectedResource" class="text-sm text-on-surface-variant">
          Select a node to edit, or use the palette to create and wire resources.
        </div>

        <section v-else-if="selectedKind === 'flow' && flowDraft" class="space-y-4">
          <div class="flex items-center justify-between gap-2">
            <h4 class="text-sm font-bold text-on-surface">
              Flow
            </h4>
            <button
              v-if="!draft"
              type="button"
              class="text-[10px] font-bold uppercase tracking-wider text-primary hover:underline"
              @click="emit('edit', 'flow')"
            >
              Edit
            </button>
          </div>

          <div class="space-y-1.5">
            <label class="text-[10px] font-bold uppercase tracking-widest text-on-surface-variant">Name *</label>
            <input
              :value="flowDraft.id"
              type="text"
              :readonly="!draft"
              placeholder="e.g. edge-flow"
              class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-lg px-3 py-2 text-sm font-mono outline-none focus:ring-1 focus:ring-primary/30 read-only:opacity-70"
              @change="onFlowName"
            >
          </div>

          <p class="text-[11px] text-on-surface-variant">
            Wire a router and/or balancer from the palette — no need to leave this editor.
          </p>

          <div class="space-y-1.5">
            <label class="text-[10px] font-bold uppercase tracking-widest text-on-surface-variant">Router ID</label>
            <input
              :value="flowDraft.router_id || ''"
              type="text"
              placeholder="use palette to attach"
              class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-lg px-3 py-2 text-sm font-mono outline-none focus:ring-1 focus:ring-primary/30"
              @change="onFlowRouter"
            >
          </div>
          <div class="space-y-1.5">
            <label class="text-[10px] font-bold uppercase tracking-widest text-on-surface-variant">Balancer ID</label>
            <input
              :value="flowDraft.balancer_id || ''"
              type="text"
              placeholder="use palette to attach"
              class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-lg px-3 py-2 text-sm font-mono outline-none focus:ring-1 focus:ring-primary/30"
              @change="onFlowBalancer"
            >
          </div>

          <p v-if="draft && storeError" class="text-[11px] text-amber-200/90">
            {{ storeError }}
          </p>
        </section>

        <section v-else class="space-y-4">
          <div class="flex items-center justify-between gap-2">
            <h4 class="text-sm font-bold text-on-surface">
              {{ kindLabel }}
            </h4>
            <button
              type="button"
              class="text-[10px] font-bold uppercase tracking-wider text-primary hover:underline"
              @click="emit('edit', selectedKind!)"
            >
              Edit
            </button>
          </div>
          <p class="font-mono text-sm text-primary">
            {{ selectedId }}
          </p>

          <template v-if="selectedKind === 'entrypoint'">
            <p class="text-xs text-on-surface-variant font-mono">
              {{ (selectedResource as Entrypoint).protocol }}://{{ (selectedResource as Entrypoint).host }}:{{ (selectedResource as Entrypoint).port }}
            </p>
            <p class="text-[11px] text-on-surface-variant">
              flow_id → {{ (selectedResource as Entrypoint).flow_id }}
            </p>
          </template>

          <template v-else-if="selectedKind === 'router'">
            <div v-if="(selectedResource as Router).rules?.length" class="space-y-1">
              <p class="text-[10px] font-bold uppercase tracking-widest text-on-surface-variant">
                Rules
              </p>
              <p
                v-for="rule in (selectedResource as Router).rules"
                :key="rule.id"
                class="text-[11px] font-mono text-on-surface-variant"
              >
                {{ rule.id }} → {{ rule.target }}
              </p>
            </div>
            <p v-else class="text-[11px] text-on-surface-variant">
              No rules yet.
            </p>
          </template>

          <template v-else-if="selectedKind === 'balancer'">
            <p class="text-[11px] font-mono text-on-surface-variant">
              type={{ (selectedResource as LoadBalancer).type }}
            </p>
            <p class="text-[11px] font-mono text-on-surface-variant">
              pool_id={{ (selectedResource as LoadBalancer).pool_id }}
            </p>
          </template>

          <template v-else-if="selectedKind === 'pool'">
            <p class="text-[11px] font-mono text-on-surface-variant">
              type={{ (selectedResource as Pool).type }}
            </p>
            <template v-if="(selectedResource as Pool).type === 'static'">
              <p
                v-for="m in (selectedResource as Pool).members || []"
                :key="m.proxy_id"
                class="text-[11px] font-mono text-on-surface-variant"
              >
                {{ m.proxy_id }} (w={{ m.weight }})
              </p>
              <p v-if="!(selectedResource as Pool).members?.length" class="text-[11px] text-on-surface-variant">
                No members.
              </p>
            </template>
            <template v-else>
              <p class="text-[10px] font-bold uppercase tracking-widest text-on-surface-variant">
                Selector (runtime)
              </p>
              <p
                v-for="[k, v] in poolSelectorPairs(selectedResource as Pool)"
                :key="k"
                class="text-[11px] font-mono text-on-surface-variant"
              >
                {{ k }}={{ v }}
              </p>
              <p v-if="!poolSelectorPairs(selectedResource as Pool).length" class="text-[11px] text-on-surface-variant">
                No allow labels.
              </p>
            </template>
          </template>

          <template v-else-if="selectedKind === 'proxy'">
            <p class="text-[11px] font-mono text-on-surface-variant">
              {{ (selectedResource as Proxy).protocol }}://{{ (selectedResource as Proxy).host }}:{{ (selectedResource as Proxy).port }}
            </p>
          </template>
        </section>
      </template>

      <template v-else>
        <div class="flex items-center justify-between mb-2">
          <h4 class="text-sm font-bold text-on-surface">
            Configuration YAML
          </h4>
          <button
            type="button"
            class="text-[10px] font-bold uppercase tracking-wider text-primary hover:underline"
            @click="copyYaml"
          >
            Copy
          </button>
        </div>
        <p class="text-[11px] text-on-surface-variant mb-3">
          Read-only projection (pgctl apply compatible). Edit via Node Settings, then Create / Deploy.
        </p>
        <pre class="bg-surface-container-lowest rounded-xl p-4 font-mono text-[11px] leading-relaxed text-on-surface-variant border border-outline-variant/15 overflow-x-auto whitespace-pre-wrap">{{ yamlText || '# empty stack' }}</pre>
      </template>
    </div>

    <div class="shrink-0 p-4 border-t border-outline-variant/15 space-y-2">
      <button
        type="button"
        class="w-full py-3 rounded-xl font-bold text-sm bg-gradient-to-br from-primary to-primary-container text-on-primary-container shadow-lg shadow-primary/15 disabled:opacity-50"
        :disabled="!canDeploy"
        @click="emit('deploy')"
      >
        {{ deployLabel }}
      </button>
    </div>
  </aside>
</template>
