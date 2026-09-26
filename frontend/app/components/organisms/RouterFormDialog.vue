<script setup lang="ts">
import type {
  BalancerListResponse,
  LoadBalancer,
  MatchType,
  Router,
  RouterApplyRequest,
  RouterCondition,
  RouterMatch,
} from '~/types'

const props = defineProps<{
  open: boolean
  mode: 'create' | 'edit'
  initial?: Router | null
}>()

const emit = defineEmits<{
  close: []
  saved: [router: { id: string }]
}>()

const MATCH_TYPES: MatchType[] = [
  'host',
  'host_suffix',
  'path_prefix',
  'path_regex',
  'method',
  'header',
  'catch_all',
]

const CONDITION_TYPES: MatchType[] = [
  'host',
  'host_suffix',
  'path_prefix',
  'path_regex',
  'method',
  'header',
]

type RuleMode = 'simple' | 'all' | 'any'

type RuleDraft = {
  id: string
  mode: RuleMode
  type: MatchType
  value: string
  conditions: RouterCondition[]
  hasNot: boolean
  notType: MatchType
  notValue: string
  target: string
}

const { apply } = useRouters()
const { apiFetch } = useApi()

const name = ref('')
const title = ref('')
const description = ref('')
const rules = ref<RuleDraft[]>([])
const balancers = ref<LoadBalancer[]>([])
const balancersLoading = ref(false)
const formError = ref('')
const saving = ref(false)

const dialogTitle = computed(() => (props.mode === 'edit' ? 'Edit Router' : 'Add New Router'))
const nameReadonly = computed(() => props.mode === 'edit')
const ruleCount = computed(() => rules.value.filter(r => r.id.trim()).length)

const tipName = 'Resource ID (metadata.name). Immutable after create. Referenced by Flow.router_id.'
const tipRules = 'Rules are evaluated in list order. First match wins. At least one rule is required. Each target must be an existing Load Balancer.'
const tipComposite = 'Composite match uses ALL (AND) or ANY (OR) conditions, with optional NOT exclusion.'

function emptyCondition(): RouterCondition {
  return { type: 'host', value: '' }
}

function emptyRule(mode: RuleMode = 'simple'): RuleDraft {
  return {
    id: '',
    mode,
    type: 'host',
    value: '',
    conditions: [emptyCondition()],
    hasNot: false,
    notType: 'host',
    notValue: '',
    target: '',
  }
}

function detectMode(match: RouterMatch): RuleMode {
  if (match.all?.length) return 'all'
  if (match.any?.length) return 'any'
  return 'simple'
}

function ruleFromDomain(rule: { id: string; match: RouterMatch; target: string }): RuleDraft {
  const mode = detectMode(rule.match)
  const draft = emptyRule(mode)
  draft.id = rule.id
  draft.target = rule.target
  if (mode === 'simple') {
    draft.type = rule.match.type || 'host'
    draft.value = rule.match.value || ''
  }
  else if (mode === 'all') {
    draft.conditions = (rule.match.all || []).map(c => ({ type: c.type, value: c.value }))
    if (!draft.conditions.length) draft.conditions = [emptyCondition()]
  }
  else {
    draft.conditions = (rule.match.any || []).map(c => ({ type: c.type, value: c.value }))
    if (!draft.conditions.length) draft.conditions = [emptyCondition()]
  }
  if (rule.match.not) {
    draft.hasNot = true
    draft.notType = rule.match.not.type
    draft.notValue = rule.match.not.value
  }
  return draft
}

watch(
  () => [props.open, props.initial, props.mode] as const,
  async () => {
    if (!props.open) return
    formError.value = ''
    saving.value = false
    balancersLoading.value = true
    try {
      const res = await apiFetch<BalancerListResponse>('/api/v1/balancers?page_size=100')
      balancers.value = res.items || []
    }
    catch {
      balancers.value = []
    }
    finally {
      balancersLoading.value = false
    }

    if (props.mode === 'edit' && props.initial) {
      name.value = props.initial.id
      title.value = props.initial.title || ''
      description.value = props.initial.description || ''
      rules.value = (props.initial.rules || []).map(ruleFromDomain)
      if (!rules.value.length) rules.value = [emptyRule()]
    }
    else {
      name.value = ''
      title.value = ''
      description.value = ''
      rules.value = [
        emptyRule('simple'),
        emptyRule('simple'),
      ]
      rules.value[0].id = 'r1'
      rules.value[0].type = 'host'
      rules.value[0].value = ''
      rules.value[1].id = 'r-default'
      rules.value[1].type = 'catch_all'
    }
  },
  { immediate: true },
)

function addRule(mode: RuleMode = 'simple') {
  const draft = emptyRule(mode)
  draft.id = `r${rules.value.length + 1}`
  rules.value.push(draft)
}

function removeRule(i: number) {
  rules.value.splice(i, 1)
  if (!rules.value.length) rules.value.push(emptyRule())
}

function moveRule(i: number, dir: -1 | 1) {
  const j = i + dir
  if (j < 0 || j >= rules.value.length) return
  const copy = [...rules.value]
  const tmp = copy[i]!
  copy[i] = copy[j]!
  copy[j] = tmp
  rules.value = copy
}

function addCondition(rule: RuleDraft) {
  rule.conditions.push(emptyCondition())
}

function removeCondition(rule: RuleDraft, i: number) {
  rule.conditions.splice(i, 1)
  if (!rule.conditions.length) rule.conditions.push(emptyCondition())
}

function valuePlaceholder(t: MatchType): string {
  switch (t) {
    case 'host': return 'example.com or *.example.com'
    case 'host_suffix': return '.de'
    case 'path_prefix': return '/api/'
    case 'path_regex': return '^/v\\d+/'
    case 'method': return 'GET'
    case 'header': return 'X-Region:de'
    default: return ''
  }
}

function buildMatch(rule: RuleDraft): RouterMatch {
  if (rule.mode === 'simple') {
    const match: RouterMatch = { type: rule.type }
    if (rule.type !== 'catch_all') {
      match.value = rule.value.trim()
    }
    return match
  }

  const conds = rule.conditions
    .map(c => ({ type: c.type, value: c.value.trim() }))
    .filter(c => c.value !== '' || c.type === 'catch_all')

  const match: RouterMatch = rule.mode === 'all' ? { all: conds } : { any: conds }

  if (rule.hasNot && rule.notValue.trim()) {
    match.not = { type: rule.notType, value: rule.notValue.trim() }
  }
  return match
}

function buildBody(): RouterApplyRequest {
  const metaName = name.value.trim()
  if (!metaName) throw new Error('name is required')

  const outRules: RouterApplyRequest['spec']['rules'] = []
  for (const [i, rule] of rules.value.entries()) {
    const id = rule.id.trim()
    if (!id) throw new Error(`rules[${i}].id is required`)
    const target = rule.target.trim()
    if (!target) throw new Error(`rules[${i}].target is required`)

    if (rule.mode === 'simple') {
      if (rule.type !== 'catch_all' && !rule.value.trim()) {
        throw new Error(`rules[${i}].match value is required for type ${rule.type}`)
      }
    }
    else {
      const filled = rule.conditions.filter(c => c.value.trim())
      if (!filled.length) {
        throw new Error(`rules[${i}].match needs at least one condition`)
      }
    }

    outRules.push({
      id,
      target,
      match: buildMatch(rule),
    })
  }

  if (!outRules.length) throw new Error('at least one rule is required')

  const body: RouterApplyRequest = {
    metadata: { name: metaName },
    spec: { rules: outRules },
  }
  const t = title.value.trim()
  if (t) body.spec.title = t
  const d = description.value.trim()
  if (d) body.spec.description = d
  return body
}

async function onSave() {
  formError.value = ''
  saving.value = true
  try {
    const body = buildBody()
    const router = await apply(body)
    useAppToast().success(props.mode === 'edit' ? 'Router updated' : 'Router created')
    emit('saved', { id: router.id })
    emit('close')
  }
  catch (e: unknown) {
    const msg = e instanceof Error ? e.message : 'save failed'
    formError.value = msg
    useAppToast().error(msg)
  }
  finally {
    saving.value = false
  }
}

function onBackdrop() {
  if (!saving.value) emit('close')
}

function setMode(rule: RuleDraft, mode: RuleMode) {
  rule.mode = mode
  if (mode !== 'simple' && !rule.conditions.length) {
    rule.conditions = [emptyCondition()]
  }
}
</script>

<template>
  <Teleport to="body">
    <div
      v-if="open"
      class="fixed inset-0 z-[100] flex items-center justify-center p-4 overflow-hidden"
      role="dialog"
      aria-modal="true"
      :aria-label="dialogTitle"
    >
      <div class="absolute inset-0 bg-black/60 backdrop-blur-sm" @click="onBackdrop" />
      <div class="relative w-full max-w-2xl max-h-[min(90vh,820px)] flex flex-col glass-panel rounded-xl border border-outline-variant/20 shadow-2xl overflow-hidden">
        <div class="shrink-0 px-6 pt-6 pb-3 border-b border-outline-variant/10">
          <h2 class="text-xl font-black tracking-tight text-on-surface">
            {{ dialogTitle }}
          </h2>
          <p class="text-sm text-on-surface-variant mt-1">
            Ordered rules that select a load balancer for matching requests.
          </p>
        </div>

        <div class="flex-1 min-h-0 overflow-y-auto overscroll-contain px-6 py-4 space-y-5">
          <div class="space-y-2">
            <div class="flex items-center gap-1.5">
              <label class="text-xs font-bold text-on-surface" for="router-name">Name</label>
              <AtomsInfoTip :text="tipName" />
              <span class="text-[10px] font-bold uppercase tracking-wider px-1.5 py-0.5 rounded bg-surface-container-highest text-on-surface-variant border border-outline-variant/20">
                RFC 1123
              </span>
            </div>
            <input
              id="router-name"
              v-model="name"
              type="text"
              :readonly="nameReadonly"
              required
              placeholder="e.g. edge-router"
              class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-xl px-4 py-3 text-sm font-mono text-on-surface focus:border-primary focus:ring-4 focus:ring-primary/20 outline-none transition-all read-only:opacity-70"
            >
          </div>

          <div class="space-y-2">
            <label class="text-xs font-bold text-on-surface" for="router-title">
              Title <span class="font-normal text-on-surface-variant">(optional)</span>
            </label>
            <input
              id="router-title"
              v-model="title"
              type="text"
              placeholder="e.g. Edge Host Router"
              class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-xl px-4 py-3 text-sm text-on-surface focus:border-primary focus:ring-4 focus:ring-primary/20 outline-none transition-all"
            >
          </div>

          <div class="space-y-2">
            <label class="text-xs font-bold text-on-surface" for="router-desc">
              Description <span class="font-normal text-on-surface-variant">(optional)</span>
            </label>
            <textarea
              id="router-desc"
              v-model="description"
              rows="2"
              placeholder="Short note about this router"
              class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-xl px-4 py-3 text-sm text-on-surface focus:border-primary focus:ring-4 focus:ring-primary/20 outline-none transition-all resize-y min-h-[4rem]"
            />
          </div>

          <div class="space-y-3">
            <div class="flex items-center gap-2 flex-wrap">
              <h3 class="text-[10px] font-black uppercase tracking-widest text-on-surface-variant">
                Rules
              </h3>
              <AtomsInfoTip :text="tipRules" />
              <span class="text-[10px] font-bold uppercase tracking-wider px-2 py-0.5 rounded bg-primary/15 text-primary">
                {{ ruleCount }} {{ ruleCount === 1 ? 'rule' : 'rules' }}
              </span>
            </div>

            <div
              v-for="(rule, i) in rules"
              :key="i"
              class="rounded-xl border border-outline-variant/15 bg-surface-container-low/40 p-4 space-y-3"
            >
              <div class="flex items-center justify-between gap-2">
                <span class="text-[10px] font-black uppercase tracking-widest text-on-surface-variant">
                  Rule {{ i + 1 }}
                </span>
                <div class="flex items-center gap-1">
                  <button
                    type="button"
                    class="p-1.5 rounded-lg text-outline hover:text-on-surface disabled:opacity-30"
                    :disabled="i === 0"
                    aria-label="Move rule up"
                    @click="moveRule(i, -1)"
                  >
                    ↑
                  </button>
                  <button
                    type="button"
                    class="p-1.5 rounded-lg text-outline hover:text-on-surface disabled:opacity-30"
                    :disabled="i === rules.length - 1"
                    aria-label="Move rule down"
                    @click="moveRule(i, 1)"
                  >
                    ↓
                  </button>
                  <button
                    type="button"
                    class="p-1.5 rounded-lg text-outline hover:text-error"
                    aria-label="Remove rule"
                    @click="removeRule(i)"
                  >
                    <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" aria-hidden="true">
                      <polyline points="3 6 5 6 21 6" />
                      <path d="M19 6l-1 14a2 2 0 01-2 2H8a2 2 0 01-2-2L5 6" />
                    </svg>
                  </button>
                </div>
              </div>

              <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <div class="space-y-1.5">
                  <label class="text-[10px] font-bold uppercase tracking-widest text-on-surface-variant">Rule ID</label>
                  <input
                    v-model="rule.id"
                    type="text"
                    placeholder="r1"
                    class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-lg px-3 py-2 text-sm font-mono text-on-surface outline-none focus:ring-1 focus:ring-primary/20"
                  >
                </div>
                <div class="space-y-1.5">
                  <label class="text-[10px] font-bold uppercase tracking-widest text-on-surface-variant">Target balancer</label>
                  <select
                    v-model="rule.target"
                    :disabled="balancersLoading"
                    class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-lg px-3 py-2 text-sm font-mono text-on-surface outline-none focus:ring-1 focus:ring-primary/20 disabled:opacity-60"
                  >
                    <option value="" disabled>
                      {{ balancersLoading ? 'Loading…' : 'Select balancer' }}
                    </option>
                    <option v-for="b in balancers" :key="b.id" :value="b.id">
                      {{ b.id }}{{ b.title ? ` — ${b.title}` : '' }}
                    </option>
                  </select>
                </div>
              </div>

              <div class="grid grid-cols-3 gap-1 p-1 rounded-lg bg-surface-container-lowest border border-outline-variant/15">
                <button
                  type="button"
                  class="py-2 text-[11px] font-bold rounded-md transition-colors"
                  :class="rule.mode === 'simple' ? 'bg-surface-container-highest text-primary shadow-sm' : 'text-on-surface-variant'"
                  @click="setMode(rule, 'simple')"
                >
                  Simple
                </button>
                <button
                  type="button"
                  class="py-2 text-[11px] font-bold rounded-md transition-colors"
                  :class="rule.mode === 'all' ? 'bg-surface-container-highest text-primary shadow-sm' : 'text-on-surface-variant'"
                  @click="setMode(rule, 'all')"
                >
                  ALL (AND)
                </button>
                <button
                  type="button"
                  class="py-2 text-[11px] font-bold rounded-md transition-colors"
                  :class="rule.mode === 'any' ? 'bg-surface-container-highest text-primary shadow-sm' : 'text-on-surface-variant'"
                  @click="setMode(rule, 'any')"
                >
                  ANY (OR)
                </button>
              </div>

              <template v-if="rule.mode === 'simple'">
                <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
                  <div class="space-y-1.5">
                    <label class="text-[10px] font-bold uppercase tracking-widest text-on-surface-variant">Match type</label>
                    <select
                      v-model="rule.type"
                      class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-lg px-3 py-2 text-sm text-on-surface outline-none focus:ring-1 focus:ring-primary/20"
                    >
                      <option v-for="t in MATCH_TYPES" :key="t" :value="t">
                        {{ t }}
                      </option>
                    </select>
                  </div>
                  <div v-if="rule.type !== 'catch_all'" class="space-y-1.5">
                    <label class="text-[10px] font-bold uppercase tracking-widest text-on-surface-variant">Value</label>
                    <input
                      v-model="rule.value"
                      type="text"
                      :placeholder="valuePlaceholder(rule.type)"
                      class="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-lg px-3 py-2 text-sm font-mono text-on-surface outline-none focus:ring-1 focus:ring-primary/20"
                    >
                  </div>
                  <p v-else class="sm:col-span-2 text-[11px] text-on-surface-variant">
                    catch_all matches any request and needs no value. Prefer as the last rule.
                  </p>
                </div>
              </template>

              <template v-else>
                <div class="flex items-center gap-1.5">
                  <span class="text-[10px] font-bold uppercase tracking-widest text-on-surface-variant">Conditions</span>
                  <AtomsInfoTip :text="tipComposite" />
                </div>
                <div
                  v-for="(cond, ci) in rule.conditions"
                  :key="ci"
                  class="grid grid-cols-[7.5rem_1fr_2rem] gap-2 items-center"
                >
                  <select
                    v-model="cond.type"
                    class="bg-surface-container-lowest border border-outline-variant/20 rounded-lg px-2 py-2 text-xs text-on-surface outline-none"
                  >
                    <option v-for="t in CONDITION_TYPES" :key="t" :value="t">
                      {{ t }}
                    </option>
                  </select>
                  <input
                    v-model="cond.value"
                    type="text"
                    :placeholder="valuePlaceholder(cond.type)"
                    class="bg-surface-container-lowest border border-outline-variant/20 rounded-lg px-3 py-2 text-sm font-mono text-on-surface outline-none focus:ring-1 focus:ring-primary/20"
                  >
                  <button
                    type="button"
                    class="p-1.5 text-on-surface-variant hover:text-error"
                    aria-label="Remove condition"
                    @click="removeCondition(rule, ci)"
                  >
                    ×
                  </button>
                </div>
                <button
                  type="button"
                  class="text-[10px] font-bold text-primary hover:text-primary-container"
                  @click="addCondition(rule)"
                >
                  + Add condition
                </button>

                <label class="flex items-center gap-2 text-xs text-on-surface-variant pt-1">
                  <input v-model="rule.hasNot" type="checkbox" class="rounded border-outline-variant">
                  NOT exclusion
                </label>
                <div v-if="rule.hasNot" class="grid grid-cols-[7.5rem_1fr] gap-2">
                  <select
                    v-model="rule.notType"
                    class="bg-surface-container-lowest border border-outline-variant/20 rounded-lg px-2 py-2 text-xs text-on-surface outline-none"
                  >
                    <option v-for="t in CONDITION_TYPES" :key="t" :value="t">
                      {{ t }}
                    </option>
                  </select>
                  <input
                    v-model="rule.notValue"
                    type="text"
                    :placeholder="valuePlaceholder(rule.notType)"
                    class="bg-surface-container-lowest border border-amber-500/30 rounded-lg px-3 py-2 text-sm font-mono text-amber-100 outline-none focus:ring-1 focus:ring-amber-500/30"
                  >
                </div>
              </template>
            </div>

            <div class="flex flex-wrap gap-3">
              <button
                type="button"
                class="flex items-center gap-2 text-[10px] font-bold text-primary hover:text-primary-container transition-colors"
                @click="addRule('simple')"
              >
                <span class="text-sm leading-none">+</span>
                Add simple rule
              </button>
              <button
                type="button"
                class="flex items-center gap-2 text-[10px] font-bold text-primary hover:text-primary-container transition-colors"
                @click="addRule('all')"
              >
                <span class="text-sm leading-none">+</span>
                Add ALL (AND) rule
              </button>
              <button
                type="button"
                class="flex items-center gap-2 text-[10px] font-bold text-primary hover:text-primary-container transition-colors"
                @click="addRule('any')"
              >
                <span class="text-sm leading-none">+</span>
                Add ANY (OR) rule
              </button>
            </div>
          </div>

          <div class="flex items-start gap-2 text-sm bg-amber-500/10 border border-amber-500/30 text-amber-200 rounded-xl px-3 py-2.5">
            <svg class="w-4 h-4 mt-0.5 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v4m0 4h.01M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z" />
            </svg>
            <p>
              Every <span class="font-mono">target</span> must reference an existing Load Balancer. Prefer a trailing
              <span class="font-mono">catch_all</span> rule for a default.
            </p>
          </div>

          <p v-if="formError" class="text-error text-sm" role="alert">
            {{ formError }}
          </p>
        </div>

        <div class="shrink-0 flex items-center justify-between gap-3 px-6 py-4 border-t border-outline-variant/10 bg-surface-container-low/40">
          <span class="text-[10px] font-mono text-outline">pgway.v1.Router</span>
          <div class="flex items-center gap-3">
            <button
              type="button"
              class="text-sm font-bold text-on-surface-variant hover:text-on-surface transition-colors px-4 py-2"
              :disabled="saving"
              @click="emit('close')"
            >
              Cancel
            </button>
            <button
              type="button"
              class="bg-gradient-to-br from-primary to-primary-container text-on-primary-container font-bold px-8 py-2.5 rounded-xl shadow-lg shadow-primary/10 hover:shadow-primary/20 active:scale-[0.98] transition-all disabled:opacity-60"
              :disabled="saving"
              @click="onSave"
            >
              {{ saving ? 'Saving…' : 'Save Router' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>
