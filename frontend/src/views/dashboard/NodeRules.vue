<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { storeToRefs } from 'pinia'
import { useNodeRulesStore } from '../../stores/noderules'
import PopConfirm from '../../components/PopConfirm.vue'
import Button from '../../components/Button.vue'
import Input from '../../components/Input.vue'
import SearchPicker, { type PickerOption } from '../../components/SearchPicker.vue'
import { Dialog } from '../../volt'
import { Select } from '../../volt'
import type { Filter, Group, Matcher, MatcherType, FilterOutboundType } from '../../types/noderules'
import { URLTEST_DEFAULTS } from '../../types/noderules'
import {
  PencilIcon,
  TrashIcon,
  PlusIcon,
  XMarkIcon,
  AdjustmentsHorizontalIcon,
  SparklesIcon,
  InformationCircleIcon
} from '@heroicons/vue/24/outline'

const { t } = useI18n()
const store = useNodeRulesStore()
const { filters, groups, keywords, templates, preview, loading, error } = storeToRefs(store)

const applying = ref(false)
const notice = ref('')

/**
 * The page is two modes, not one page with an appendix.
 *
 * The preview used to sit below the editor, which put it off-screen exactly
 * when it mattered: a real config has enough filters and groups that reading
 * "what would this collect?" meant scrolling past the thing you had just
 * edited, and losing sight of it. Swapping the body keeps both halves at the
 * top of the viewport and makes the relationship explicit — same data, two
 * views of it.
 *
 * Both modes are read-only with respect to the config. Nothing is written
 * until Apply, which is why the switch itself needs no confirmation and why
 * Apply stays reachable from either side.
 */
type PageMode = 'edit' | 'preview'
const mode = ref<PageMode>('edit')

/**
 * Entering preview re-runs it.
 *
 * A stale preview is worse than none: it describes filters that may have been
 * edited since, and it looks authoritative. The run is awaited only for its
 * loading state — `store.runPreview()` already writes into `preview`.
 */
async function showPreview() {
  mode.value = 'preview'
  await store.runPreview()
}

// ---- Select option catalogs ----
// Static (non-translated) option lists for the volt Select dropdowns.
const outboundTypeOptions: { value: FilterOutboundType; label: string }[] = [
  { value: 'urltest', label: 'urltest' },
  { value: 'selector', label: 'selector' },
]

const modeOptions = computed<{ value: PageMode; label: string }[]>(() => [
  { value: 'edit', label: t('nodeRules.modes.edit') },
  { value: 'preview', label: t('nodeRules.modes.preview') },
])

const matcherTypeOptions: { value: MatcherType; label: string }[] = [
  { value: 'keyword', label: 'keyword' },
  { value: 'code', label: 'code' },
  { value: 'emoji', label: 'emoji' },
]

// ---- Filter editor state ----
const editingFilterId = ref<string | null>(null) // null = not editing; '' = creating
const filterForm = reactive<{
  name: string
  outbound_type: FilterOutboundType
  priority: number
  matchers: Matcher[]
  excludes: Matcher[]
  test_url: string
  test_interval: string
  test_tolerance: number
}>({
  name: '',
  outbound_type: 'urltest',
  priority: 0,
  matchers: [],
  excludes: [],
  test_url: URLTEST_DEFAULTS.test_url,
  test_interval: URLTEST_DEFAULTS.test_interval,
  test_tolerance: URLTEST_DEFAULTS.test_tolerance,
})

// Opt-in tags reported by the preview: outbounds that are legal filter members
// but are never collected automatically — today the `direct` outbounds. They are
// absent from `members`/`unmatched` until a filter claims one, so they have to
// come from the preview's own `optional` list.
const optionalTags = computed<string[]>(() => preview.value?.optional ?? [])

// All node tags currently known (from the latest preview): the union of every
// filter's members, the unmatched list and the opt-in list. Drives both node
// pickers so users can name a specific outbound — a `direct` one included — by
// its exact tag.
const endpointTags = computed<string[]>(() => {
  const p = preview.value
  if (!p) return []
  const seen = new Set<string>()
  for (const pf of p.filters) for (const tag of pf.members) seen.add(tag)
  for (const tag of p.unmatched) seen.add(tag)
  for (const tag of p.optional ?? []) seen.add(tag)
  return [...seen].sort((a, b) => a.localeCompare(b))
})

// A `direct` outbound only ever joins a filter by an explicit matcher, so the
// picker labels it — picking one is a deliberate "bypass" member, not the same
// act as picking a proxy node.
function isOptionalTag(tag: string): boolean {
  return optionalTags.value.includes(tag)
}

/**
 * The node pool, as picker options.
 *
 * One list feeds three pickers — include a node, exclude a node, and name an
 * outbound directly in a group. The DIRECT badge marks an `optional` tag: a
 * `direct` outbound is never collected by a filter's matchers, so an explicit
 * pick is the only way it can join one, and seeing which entries those are is
 * the difference between a deliberate choice and an accident.
 *
 * Search, capping, open/close and the reset-after-pick all live in
 * SearchPicker.vue now; this file only says what is pickable and what a pick
 * means.
 */
const nodeOptions = computed<PickerOption[]>(() =>
  endpointTags.value.map((tag) => ({
    value: tag,
    badge: isOptionalTag(tag) ? t('nodeRules.directBadge') : undefined,
  })),
)

/**
 * The backend country catalog, as picker options.
 *
 * `terms` carries the code and every synonym so the search still matches "hk",
 * "香港" and "Hong Kong" alike, while the visible label stays "Hong Kong (HK)".
 */
const countryOptions = computed<PickerOption[]>(() =>
  keywords.value.map((kw) => ({
    value: kw.code,
    label: `${kw.label} (${kw.code})`,
    title: kw.synonyms.join(' · '),
    terms: [kw.code, kw.label, ...kw.synonyms],
  })),
)

function startCreateFilter() {
  editingFilterId.value = ''
  filterForm.name = ''
  filterForm.outbound_type = 'urltest'
  filterForm.priority = (filters.value.filter((f) => !f.is_fallback).length + 1) * 10
  filterForm.matchers = []
  filterForm.excludes = []
  filterForm.test_url = URLTEST_DEFAULTS.test_url
  filterForm.test_interval = URLTEST_DEFAULTS.test_interval
  filterForm.test_tolerance = URLTEST_DEFAULTS.test_tolerance
}

function startEditFilter(f: Filter) {
  editingFilterId.value = f.id
  filterForm.name = f.name
  filterForm.outbound_type = f.outbound_type
  filterForm.priority = f.priority
  filterForm.matchers = f.matchers.map((m) => ({ ...m }))
  filterForm.excludes = (f.excludes ?? []).map((m) => ({ ...m }))
  filterForm.test_url = f.test_url || URLTEST_DEFAULTS.test_url
  filterForm.test_interval = f.test_interval || URLTEST_DEFAULTS.test_interval
  filterForm.test_tolerance = f.test_tolerance || URLTEST_DEFAULTS.test_tolerance
}

function cancelFilterEdit() {
  editingFilterId.value = null
}

// Matcher edits replace the array (immutable update) rather than mutating it.
function addMatcher(type: MatcherType = 'keyword', value = '') {
  filterForm.matchers = [...filterForm.matchers, { type, value }]
}

function removeMatcher(idx: number) {
  filterForm.matchers = filterForm.matchers.filter((_, i) => i !== idx)
}

// Match a specific node by its exact tag (keyword match). De-dupes so picking
// the same node twice is a no-op.
function addMatcherNode(tag: string) {
  if (!tag) return
  if (filterForm.matchers.some((m) => m.type === 'keyword' && m.value === tag)) return
  filterForm.matchers = [...filterForm.matchers, { type: 'keyword', value: tag }]
}

function addCodeMatcher(code: string) {
  if (!code) return
  if (filterForm.matchers.some((m) => m.type === 'code' && m.value === code)) return
  filterForm.matchers = [...filterForm.matchers, { type: 'code', value: code }]
}

// Exclude (deny-list) edits — same immutable pattern as matchers.
function addExclude(type: MatcherType = 'keyword', value = '') {
  filterForm.excludes = [...filterForm.excludes, { type, value }]
}

function removeExclude(idx: number) {
  filterForm.excludes = filterForm.excludes.filter((_, i) => i !== idx)
}

// Exclude a specific node by its exact tag (keyword match). De-dupes so picking
// the same node twice is a no-op.
function addExcludeNode(tag: string) {
  if (!tag) return
  if (filterForm.excludes.some((m) => m.type === 'keyword' && m.value === tag)) return
  filterForm.excludes = [...filterForm.excludes, { type: 'keyword', value: tag }]
}

async function saveFilter() {
  notice.value = ''
  const input = {
    name: filterForm.name.trim(),
    outbound_type: filterForm.outbound_type,
    priority: filterForm.priority,
    matchers: filterForm.matchers.filter((m) => m.value.trim() !== ''),
    excludes: filterForm.excludes.filter((m) => m.value.trim() !== ''),
    test_url: filterForm.test_url.trim(),
    test_interval: filterForm.test_interval.trim(),
    test_tolerance: Number(filterForm.test_tolerance) || 0,
  }
  if (!input.name) {
    notice.value = t('nodeRules.nameRequired')
    return
  }
  try {
    if (editingFilterId.value) {
      await store.updateFilter(editingFilterId.value, input)
    } else {
      await store.createFilter(input)
    }
    editingFilterId.value = null
  } catch (e) {
    notice.value = e instanceof Error ? e.message : String(e)
  }
}

async function removeFilter(f: Filter) {
  if (f.is_fallback) return
  try {
    await store.deleteFilter(f.id)
  } catch (e) {
    notice.value = e instanceof Error ? e.message : String(e)
  }
}

// ---- Group editor state ----
const editingGroupId = ref<string | null>(null)
const groupForm = reactive<{
  name: string
  priority: number
  filter_ids: string[]
  extra_tags: string[]
}>({
  name: '',
  priority: 0,
  filter_ids: [],
  extra_tags: [],
})

function startCreateGroup() {
  editingGroupId.value = ''
  groupForm.name = ''
  groupForm.priority = (groups.value.length + 1) * 10
  groupForm.filter_ids = []
  groupForm.extra_tags = []
}

function startEditGroup(g: Group) {
  editingGroupId.value = g.id
  groupForm.name = g.name
  groupForm.priority = g.priority
  groupForm.filter_ids = [...g.filter_ids]
  groupForm.extra_tags = [...(g.extra_tags ?? [])]
}

// A group may also name outbounds directly — the `direct` bypass entry a
// selector usually wants next to its regions. Filters never collect `direct`
// on their own, so this picker is the only route to it.
function addGroupTag(tag: string) {
  if (!tag || groupForm.extra_tags.includes(tag)) return
  groupForm.extra_tags = [...groupForm.extra_tags, tag]
}

function removeGroupTag(tag: string) {
  groupForm.extra_tags = groupForm.extra_tags.filter((t) => t !== tag)
}

function cancelGroupEdit() {
  editingGroupId.value = null
}

function toggleGroupFilter(id: string) {
  groupForm.filter_ids = groupForm.filter_ids.includes(id)
    ? groupForm.filter_ids.filter((x) => x !== id)
    : [...groupForm.filter_ids, id]
}

async function saveGroup() {
  notice.value = ''
  const input = {
    name: groupForm.name.trim(),
    priority: groupForm.priority,
    filter_ids: groupForm.filter_ids,
    extra_tags: groupForm.extra_tags,
  }
  if (!input.name) {
    notice.value = t('nodeRules.nameRequired')
    return
  }
  try {
    if (editingGroupId.value) {
      await store.updateGroup(editingGroupId.value, input)
    } else {
      await store.createGroup(input)
    }
    editingGroupId.value = null
  } catch (e) {
    notice.value = e instanceof Error ? e.message : String(e)
  }
}

async function removeGroup(g: Group) {
  try {
    await store.deleteGroup(g.id)
  } catch (e) {
    notice.value = e instanceof Error ? e.message : String(e)
  }
}

// ---- Templates / preview / apply ----
async function addTemplate(id: string) {
  notice.value = ''
  try {
    await store.applyTemplate(id)
  } catch (e) {
    notice.value = e instanceof Error ? e.message : String(e)
  }
}

async function doApply() {
  applying.value = true
  notice.value = ''
  try {
    await store.apply()
    notice.value = t('nodeRules.applied')
    await store.runPreview()
  } catch (e) {
    notice.value = e instanceof Error ? e.message : String(e)
  } finally {
    applying.value = false
  }
}

function filterName(id: string): string {
  return filters.value.find((f) => f.id === id)?.name ?? id
}

onMounted(async () => {
  await Promise.all([store.fetchAll(), store.fetchCatalog()])
  await store.runPreview()
})
</script>

<template>
  <div class="space-y-3 animate-fade-in">
    <!-- Header: subtitle + actions -->
    <div class="flex flex-wrap items-center justify-between gap-2">
      <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('nodeRules.subtitle') }}</p>
      <div class="flex items-center gap-2">
        <!--
          Two modes, one control. A segmented switch rather than a pair of
          buttons: the pill shows which view you are in, which a button that
          merely triggers something cannot.
        -->
        <div
          class="inline-flex items-center gap-0.5 p-0.5 rounded-control bg-gray-100 dark:bg-slate-800 border border-gray-200 dark:border-gray-700"
          role="tablist"
          :aria-label="t('nodeRules.modes.switchHint')"
        >
          <button
            v-for="option in modeOptions"
            :key="option.value"
            type="button"
            role="tab"
            :aria-selected="mode === option.value"
            class="px-2.5 py-1 text-xs font-semibold rounded-control transition-colors duration-200 cursor-pointer"
            :class="mode === option.value
            ? 'bg-white dark:bg-slate-900 text-primary-700 dark:text-primary-400 shadow-surface'
            : 'text-gray-500 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white'"
            @click="option.value === 'preview' ? showPreview() : (mode = 'edit')"
          >
            {{ option.label }}
          </button>
        </div>
        <Button variant="primary" size="sm" action :disabled="applying" @click="doApply">
          {{ applying ? t('nodeRules.applying') : t('nodeRules.applyNow') }}
        </Button>
      </div>
    </div>

    <!-- Alert Messages -->
    <div v-if="notice" class="px-3 py-2 rounded-control bg-primary-500/10 border border-primary-500/20 text-primary-700 dark:text-primary-400 text-sm animate-fade-in flex items-center gap-2">
      <span class="w-1.5 h-1.5 rounded-pill bg-primary-500 animate-ping"></span>
      <span>{{ notice }}</span>
    </div>
    <div v-if="error" class="px-3 py-2 rounded-control bg-red-500/10 border border-red-500/20 text-red-700 dark:text-red-400 text-sm animate-fade-in flex items-center gap-2">
      <span class="w-1.5 h-1.5 rounded-pill bg-red-500 animate-ping"></span>
      <span>{{ error }}</span>
    </div>

    <!--
      Edit ⇄ Preview. `mode="out-in"` so the two never overlap: they are
      different shapes at different heights, and cross-fading them makes the
      page jump. The cost is a ~150ms gap, which is what the leave/enter
      durations below are tuned around.
    -->
    <Transition name="mode-swap" mode="out-in">
      <div v-if="mode === 'edit'" key="edit" class="space-y-3">
        <!-- Templates -->
        <section v-if="templates.length" class="space-y-2 bg-gray-50/50 dark:bg-slate-900/40 border border-gray-100 dark:border-gray-800/80 px-3 py-2.5 rounded-surface">
          <h2 class="text-xs font-bold text-gray-400 dark:text-gray-500 uppercase tracking-wider">{{ t('nodeRules.templates') }}</h2>
          <div class="flex flex-wrap gap-1.5">
            <button
              v-for="tpl in templates"
              :key="tpl.id"
              class="border border-primary-200 dark:border-primary-800 hover:border-primary-500 hover:bg-primary-50 dark:hover:bg-primary-950/20 text-primary-700 dark:text-primary-400 text-xs font-semibold px-2 py-1 rounded-control transition-all cursor-pointer"
              :title="tpl.description"
              @click="addTemplate(tpl.id)"
            >
              + {{ tpl.name }}
            </button>
          </div>
        </section>

        <!-- Filters & Groups Grid -->
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-3">
          <!-- Filters panel -->
          <section class="space-y-2 flex flex-col">
            <div class="flex items-center justify-between">
              <h2 class="text-sm font-bold text-gray-900 dark:text-white flex items-center gap-1.5">
                <AdjustmentsHorizontalIcon class="h-4 w-4 text-primary-500" />
                {{ t('nodeRules.filters') }}
              </h2>
              <button
                class="node-rule-primary-button bg-primary-600 hover:bg-primary-500 text-white text-xs font-bold px-2 py-1 rounded-control transition-colors cursor-pointer flex items-center gap-1"
                @click="startCreateFilter"
              >
                <PlusIcon class="h-3.5 w-3.5" />
                {{ t('nodeRules.addFilter') }}
              </button>
            </div>

            <!-- Filter list -->
            <ul class="space-y-1.5">
              <li
                v-for="f in filters"
                :key="f.id"
                class="node-rule-card rounded-control border border-gray-200 dark:border-gray-800 px-2.5 py-2 flex items-start justify-between gap-2 bg-white dark:bg-slate-900 transition-colors duration-200"
              >
                <div class="min-w-0 space-y-1">
                  <div class="flex items-center gap-2 flex-wrap">
                    <span class="font-bold text-sm text-gray-900 dark:text-white truncate">{{ f.name }}</span>
                    <span class="px-2 py-0.5 rounded-pill text-[10px] font-bold uppercase tracking-wider bg-primary-100 dark:bg-primary-950/40 text-primary-700 dark:text-primary-400">
                      {{ f.outbound_type }}
                    </span>
                    <span v-if="f.is_fallback" class="px-2 py-0.5 rounded-pill text-[10px] font-bold bg-amber-100 dark:bg-amber-950/40 text-amber-700 dark:text-amber-400">
                      {{ t('nodeRules.fallback') }}
                    </span>
                    <span class="text-xs text-gray-400 font-mono">P{{ f.priority }}</span>
                  </div>
                  <div v-if="!f.is_fallback" class="flex flex-wrap gap-1">
                    <span v-for="(m, i) in f.matchers" :key="i" class="px-2 py-0.5 rounded-surface text-[11px] bg-white dark:bg-slate-800 border border-gray-200 dark:border-gray-700 text-gray-700 dark:text-gray-300 font-medium">
                      {{ m.value }}
                    </span>
                    <span v-if="!f.matchers.length" class="text-xs text-gray-400 italic">{{ t('nodeRules.noMatchers') }}</span>
                    <span
                      v-for="(m, i) in (f.excludes ?? [])"
                      :key="`x${i}`"
                      class="px-2 py-0.5 rounded-control text-[11px] bg-red-50/50 dark:bg-red-950/10 border border-red-200/40 dark:border-red-950/20 text-red-600 dark:text-red-400 font-medium"
                      :title="t('nodeRules.excludeBadgeTitle')"
                    >
                      − {{ m.value }}
                    </span>
                  </div>
                  <div v-else class="text-xs text-gray-400">{{ t('nodeRules.fallbackHint') }}</div>
                </div>
                <div class="flex gap-0.5 shrink-0">
                  <button
                    @click="startEditFilter(f)"
                    class="p-1 rounded-control text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors cursor-pointer"
                    :title="t('nodeRules.edit')"
                  >
                    <PencilIcon class="h-3.5 w-3.5" />
                  </button>
                  <PopConfirm
                    v-if="!f.is_fallback"
                    :message="t('nodeRules.confirmDeleteFilter', { name: f.name })"
                    :confirm-label="t('nodeRules.delete')"
                    tone="danger"
                    triggerClass="p-1 rounded-control text-red-500 hover:bg-red-500/10 transition-colors cursor-pointer inline-flex items-center justify-center"
                    @confirm="removeFilter(f)"
                  >
                    <TrashIcon class="h-3.5 w-3.5" />
                  </PopConfirm>
                </div>
              </li>
            </ul>
          </section>

          <!-- Groups panel -->
          <section class="space-y-2 flex flex-col">
            <div class="flex items-center justify-between">
              <h2 class="text-sm font-bold text-gray-900 dark:text-white flex items-center gap-1.5">
                <SparklesIcon class="h-4 w-4 text-primary-500" />
                {{ t('nodeRules.groups') }}
              </h2>
              <button
                class="node-rule-primary-button bg-primary-600 hover:bg-primary-500 text-white text-xs font-bold px-2 py-1 rounded-control transition-colors cursor-pointer flex items-center gap-1"
                @click="startCreateGroup"
              >
                <PlusIcon class="h-3.5 w-3.5" />
                {{ t('nodeRules.addGroup') }}
              </button>
            </div>

            <!-- Group list -->
            <ul class="space-y-1.5">
              <li
                v-for="g in groups"
                :key="g.id"
                class="node-rule-card rounded-control border border-gray-200 dark:border-gray-800 px-2.5 py-2 flex items-start justify-between gap-2 bg-white dark:bg-slate-900 transition-colors duration-200"
              >
                <div class="min-w-0 space-y-1">
                  <div class="flex items-center gap-2">
                    <span class="font-bold text-sm text-gray-900 dark:text-white truncate">{{ g.name }}</span>
                    <span class="text-xs text-gray-400 font-mono">P{{ g.priority }}</span>
                  </div>
                  <div class="flex flex-wrap gap-1">
                    <span v-for="fid in g.filter_ids" :key="fid" class="px-2 py-0.5 rounded-surface text-[11px] bg-white dark:bg-slate-800 border border-gray-200 dark:border-gray-700 text-gray-700 dark:text-gray-300 font-medium">
                      {{ filterName(fid) }}
                    </span>
                    <span
                      v-for="tag in (g.extra_tags ?? [])"
                      :key="`t${tag}`"
                      class="px-2 py-0.5 rounded-surface text-[11px] bg-emerald-50/60 dark:bg-emerald-950/10 border border-emerald-200/50 dark:border-emerald-900/30 text-emerald-700 dark:text-emerald-400 font-medium"
                      :title="t('nodeRules.groupTagBadgeTitle')"
                    >
                      {{ tag }}
                    </span>
                    <span v-if="!g.filter_ids.length && !(g.extra_tags ?? []).length" class="text-xs text-gray-400 italic">{{ t('nodeRules.noFilters') }}</span>
                  </div>
                </div>
                <div class="flex gap-0.5 shrink-0">
                  <button
                    @click="startEditGroup(g)"
                    class="p-1 rounded-control text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors cursor-pointer"
                    :title="t('nodeRules.edit')"
                  >
                    <PencilIcon class="h-3.5 w-3.5" />
                  </button>
                  <PopConfirm
                    :message="t('nodeRules.confirmDeleteGroup', { name: g.name })"
                    :confirm-label="t('nodeRules.delete')"
                    tone="danger"
                    triggerClass="p-1 rounded-control text-red-500 hover:bg-red-500/10 transition-colors cursor-pointer inline-flex items-center justify-center"
                    @confirm="removeGroup(g)"
                  >
                    <TrashIcon class="h-3.5 w-3.5" />
                  </PopConfirm>
                </div>
              </li>
              <li v-if="!groups.length" class="text-xs text-gray-400 text-center py-4 bg-gray-50/20 dark:bg-slate-900/20 border border-dashed border-gray-200 dark:border-gray-800 rounded-control">
                {{ t('nodeRules.noGroups') }}
              </li>
            </ul>
          </section>
        </div>
      </div>

      <div v-else key="preview" class="space-y-3">
        <!--
          Preview never re-renders the editor's data, so a run that returns
          nothing has to say so — an empty page here reads as a broken button.
        -->
        <div v-if="!preview" class="text-center py-10 bg-gray-50/40 dark:bg-slate-900/20 border border-dashed border-gray-200 dark:border-gray-800 rounded-surface space-y-2">
          <p class="text-sm font-semibold text-gray-500 dark:text-gray-400">{{ t('nodeRules.previewEmpty') }}</p>
          <p class="text-xs text-gray-400 max-w-md mx-auto">{{ t('nodeRules.previewEmptyHint') }}</p>
          <Button variant="secondary" size="sm" action :disabled="loading" @click="store.runPreview()">
            {{ t('nodeRules.previewRun') }}
          </Button>
        </div>

        <!-- Preview Section -->
        <section v-if="preview" class="space-y-2">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <h2 class="text-sm font-bold text-gray-900 dark:text-white flex items-center gap-1.5">
              <InformationCircleIcon class="h-4 w-4 text-primary-500" />
              {{ t('nodeRules.previewResult') }}
              <span class="text-xs font-normal text-gray-400">({{ t('nodeRules.endpoints', { n: preview.endpoints }) }})</span>
            </h2>
            <!-- Entering the mode already re-runs it; this is for the second look
              after an edit made in a dialog, which does not change the mode. -->
            <Button variant="secondary" size="sm" action :disabled="loading" @click="store.runPreview()">
              {{ loading ? t('nodeRules.applying') : t('nodeRules.previewRefresh') }}
            </Button>
          </div>
          <p class="text-xs text-gray-400">{{ t('nodeRules.modes.switchHint') }}</p>

          <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-2">
            <div v-for="pf in preview.filters" :key="pf.id" class="node-rule-card rounded-control border border-gray-200 dark:border-gray-800 bg-white dark:bg-slate-900 px-2.5 py-2 self-start transition-colors duration-200">
              <!--
                No collapse. The whole point of preview mode is reading which
                nodes landed where, and a card that has to be opened first hides
                exactly that behind a click per filter. The list scrolls inside
                its own `max-h-44` instead, so a filter holding 60 nodes still
                costs one card's worth of page.
              -->
              <div class="w-full flex items-center justify-between gap-2">
                <span class="font-bold text-sm text-gray-900 dark:text-white truncate flex items-center gap-1.5 min-w-0">
                  <span class="truncate">{{ pf.name }}</span>
                  <span v-if="pf.is_fallback" class="px-2 py-0.5 rounded-pill text-[9px] font-bold bg-amber-100 dark:bg-amber-950/40 text-amber-700 dark:text-amber-400 shrink-0">{{ t('nodeRules.fallback') }}</span>
                </span>
                <span class="px-2 py-0.5 rounded-control text-xs font-bold shrink-0" :class="pf.member_count ? 'bg-primary-100 dark:bg-primary-950 text-primary-700 dark:text-primary-400' : 'bg-gray-100 dark:bg-slate-800 text-gray-400'">
                  {{ pf.member_count }}
                </span>
              </div>
              <div v-if="pf.member_count" class="mt-2 max-h-44 overflow-y-auto space-y-1 border-t border-gray-100 dark:border-gray-800 pt-2 text-xs font-mono">
                <div
                  v-for="(tag, i) in pf.members"
                  :key="i"
                  class="text-gray-600 dark:text-gray-350 truncate hover:text-gray-900 dark:hover:text-white py-0.5"
                  :title="tag"
                >
                  {{ tag }}
                </div>
              </div>
            </div>

            <!--
              Unmatched nodes — the last cell of the same grid, not a full-width
              band under it. It is one more bucket ("what no filter claimed"),
              so it reads as a peer of the filter cards; stretched across the
              row it looked like a separate section, and its own row of
              whitespace was the widest thing on the page.
            -->
            <div v-if="preview.unmatched.length" class="node-rule-card rounded-control border border-amber-200 dark:border-amber-900/40 bg-white dark:bg-slate-900 px-2.5 py-2 self-start transition-colors duration-200">
              <div class="w-full flex items-center justify-between gap-2">
                <span class="font-bold text-sm text-amber-700 dark:text-amber-400 truncate">
                  {{ t('nodeRules.unmatchedTitle') }}
                </span>
                <span class="px-2 py-0.5 rounded-control text-xs font-bold shrink-0 bg-amber-100 dark:bg-amber-950/40 text-amber-700 dark:text-amber-400">
                  {{ preview.unmatched.length }}
                </span>
              </div>
              <p class="mt-0.5 text-[11px] text-amber-600/80 dark:text-amber-500/70">
                {{ t('nodeRules.unmatchedHint') }}
              </p>
              <div class="mt-2 max-h-44 overflow-y-auto space-y-1 border-t border-amber-200/50 dark:border-amber-950/20 pt-2 text-xs font-mono">
                <div
                  v-for="(tag, i) in preview.unmatched"
                  :key="i"
                  class="text-gray-600 dark:text-gray-350 truncate py-0.5 hover:text-gray-900 dark:hover:text-white"
                  :title="tag"
                >
                  {{ tag }}
                </div>
              </div>
            </div>
          </div>
        </section>
      </div>
    </Transition>

    <!-- Create/Edit Filter Modal -->
    <Dialog
      :visible="editingFilterId !== null"
      @update:visible="(v: boolean) => { if (!v) cancelFilterEdit() }"
      modal
      class="w-full max-w-2xl"
    >
      <!-- The header slot rather than `:header`, so the section icon survives
        the move onto the shared dialog. -->
      <template #header>
        <h3 class="text-sm font-bold text-gray-900 dark:text-white flex items-center gap-2">
          <AdjustmentsHorizontalIcon class="h-5 w-5 text-primary-500" />
          {{ editingFilterId === '' ? $t('nodeRules.modal.addFilter') : $t('nodeRules.modal.editFilter') }}
        </h3>
      </template>

      <div class="space-y-3">
        <!-- Basic Settings Section -->
        <div class="space-y-2.5">
          <h4 class="text-xs font-bold text-gray-400 dark:text-gray-500 uppercase tracking-wider">Basic Settings</h4>
          <div class="grid grid-cols-1 sm:grid-cols-3 gap-2.5">
            <div class="sm:col-span-2">
              <label class="block text-xs font-semibold text-gray-500 dark:text-gray-400 mb-1.5">Filter Name</label>
              <Input
                v-model="filterForm.name"
                :placeholder="t('nodeRules.filterName')"
              />
            </div>
            <div>
              <label class="block text-xs font-semibold text-gray-500 dark:text-gray-400 mb-1.5">Outbound Type</label>
              <Select
                class="w-full"
                v-model="filterForm.outbound_type"
                :options="outboundTypeOptions"
                optionLabel="label"
                optionValue="value"
              />
            </div>
          </div>

          <div>
            <label class="block text-xs font-semibold text-gray-500 dark:text-gray-400 mb-1.5 flex justify-between">
              <span>Priority</span>
              <span class="text-gray-400 font-normal">Filters execute in descending order (higher first)</span>
            </label>
            <Input
              v-model.number="filterForm.priority"
              type="number"
            />
          </div>
        </div>

        <!-- Rule Matchers Section -->
        <div class="space-y-2.5 border-t border-gray-100 dark:border-gray-800 pt-3">
          <h4 class="text-xs font-bold text-gray-400 dark:text-gray-500 uppercase tracking-wider flex items-center gap-1.5">
            Rule Matchers
            <span class="text-[11px] font-normal text-gray-400 normal-case">(Matches node names by keywords, country codes, or emoji)</span>
          </h4>

          <!-- Existing Matchers list -->
          <div class="space-y-2 max-h-40 overflow-y-auto">
            <div v-for="(m, idx) in filterForm.matchers" :key="idx" class="flex gap-2 items-center bg-gray-50 dark:bg-slate-800/40 p-1.5 rounded-control border border-gray-100 dark:border-gray-800/50">
              <Select
                class="w-32 shrink-0"
                v-model="m.type"
                :options="matcherTypeOptions"
                optionLabel="label"
                optionValue="value"
              />
              <input
                v-model="m.value"
                class="bg-transparent flex-1 text-sm text-gray-900 dark:text-white focus:outline-none border-none py-1 px-2"
                :placeholder="t('nodeRules.matcherValue')"
              />
              <button @click="removeMatcher(idx)" class="text-gray-400 hover:text-red-500 p-1 cursor-pointer">
                <XMarkIcon class="h-4 w-4" />
              </button>
            </div>
            <div v-if="!filterForm.matchers.length" class="text-xs text-gray-400 text-center py-2 bg-gray-50/40 dark:bg-slate-800/10 rounded-control border border-dashed border-gray-200 dark:border-gray-800">
              No matchers configured. This filter will not match any nodes.
            </div>
          </div>

          <!-- Add Matcher Buttons -->
          <div class="flex flex-wrap gap-2 items-center">
            <button
              type="button"
              @click="addMatcher('keyword')"
              class="border border-primary-200 dark:border-primary-800 hover:bg-primary-50 dark:hover:bg-primary-950/20 text-primary-600 dark:text-primary-400 text-xs font-semibold px-2 py-1 rounded-control transition-all cursor-pointer"
            >
              + Add Keyword
            </button>
            <button
              type="button"
              @click="addMatcher('emoji')"
              class="border border-primary-200 dark:border-primary-800 hover:bg-primary-50 dark:hover:bg-primary-950/20 text-primary-600 dark:text-primary-400 text-xs font-semibold px-2 py-1 rounded-control transition-all cursor-pointer"
            >
              + Add Emoji
            </button>

            <!--
              Node Picker: adds an exact-tag keyword matcher. This is how a
              `direct` outbound joins a filter — the matcher never collects one
              automatically, so without an explicit pick it is unreachable.
            -->
            <SearchPicker
              v-if="endpointTags.length"
              :options="nodeOptions"
              :placeholder="t('nodeRules.includeNode')"
              :empty-message="t('nodeRules.noNodeMatches')"
              input-class="w-48"
              panel-class="w-64"
              @select="addMatcherNode"
            />

            <!-- Country Picker Searchable Input -->
            <!-- The whole catalog is offered uncapped (`:limit="0"`): it is a
              fixed ~200 countries, not a subscription that grows. -->
            <SearchPicker
              v-if="keywords.length"
              :options="countryOptions"
              :limit="0"
              :placeholder="t('nodeRules.addCountry')"
              :empty-message="t('nodeRules.noCodeMatches')"
              input-class="w-48"
              panel-class="w-60"
              @select="addCodeMatcher"
            />
          </div>
        </div>

        <!-- Rule Excludes Section -->
        <div class="space-y-2.5 border-t border-gray-100 dark:border-gray-800 pt-3">
          <h4 class="text-xs font-bold text-gray-400 dark:text-gray-500 uppercase tracking-wider flex items-center gap-1.5">
            Exclude Rules (Deny-list)
            <span class="text-[11px] font-normal text-gray-400 normal-case">(Keep nodes OUT even if they match)</span>
          </h4>

          <!-- Existing Excludes list -->
          <div class="space-y-2 max-h-40 overflow-y-auto">
            <div v-for="(m, idx) in filterForm.excludes" :key="idx" class="flex gap-2 items-center bg-red-50/20 dark:bg-red-950/5 p-1.5 rounded-control border border-red-100/50 dark:border-red-950/20">
              <Select
                class="w-32 shrink-0"
                v-model="m.type"
                :options="matcherTypeOptions"
                optionLabel="label"
                optionValue="value"
              />
              <input
                v-model="m.value"
                class="bg-transparent flex-1 text-sm text-gray-900 dark:text-white focus:outline-none border-none py-1 px-2"
                :placeholder="t('nodeRules.matcherValue')"
              />
              <button @click="removeExclude(idx)" class="text-gray-400 hover:text-red-500 p-1 cursor-pointer">
                <XMarkIcon class="h-4 w-4" />
              </button>
            </div>
            <div v-if="!filterForm.excludes.length" class="text-xs text-gray-400 text-center py-2 bg-gray-50/40 dark:bg-slate-800/10 rounded-control border border-dashed border-gray-200 dark:border-gray-800">
              No exclusion rules configured.
            </div>
          </div>

          <!-- Add Exclude Buttons -->
          <div class="flex flex-wrap gap-2 items-center">
            <button
              type="button"
              @click="addExclude('keyword')"
              class="border border-red-200 dark:border-red-900/50 hover:bg-red-50/10 dark:hover:bg-red-950/20 text-red-600 dark:text-red-400 text-xs font-semibold px-2 py-1 rounded-control transition-all cursor-pointer"
            >
              + Add Exclude Keyword
            </button>

            <!-- Node Picker Searchable Input -->
            <SearchPicker
              v-if="endpointTags.length"
              :options="nodeOptions"
              tone="danger"
              :placeholder="t('nodeRules.excludeNode')"
              :empty-message="t('nodeRules.noNodeMatches')"
              input-class="w-48"
              panel-class="w-64"
              @select="addExcludeNode"
            />
          </div>
        </div>

        <!-- urltest Advanced Settings Section -->
        <div v-if="filterForm.outbound_type === 'urltest'" class="space-y-2.5 border-t border-gray-100 dark:border-gray-800 pt-3">
          <h4 class="text-xs font-bold text-gray-400 dark:text-gray-500 uppercase tracking-wider">Health Check Settings (urltest)</h4>
          <div class="grid grid-cols-1 gap-2.5">
            <div>
              <label class="block text-xs font-semibold text-gray-500 dark:text-gray-400 mb-1.5">Test URL</label>
              <Input
                v-model="filterForm.test_url"
                placeholder="http://www.gstatic.com/generate_204"
              />
            </div>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-2.5">
              <div>
                <label class="block text-xs font-semibold text-gray-500 dark:text-gray-400 mb-1.5">Test Interval</label>
                <Input
                  v-model="filterForm.test_interval"
                  placeholder="3m"
                />
                <!--
                  Every member is dialled on each tick, so a short interval on
                  a large filter is a sustained flood. The server clamps the
                  rate regardless; this explains why a typed value may not be
                  what ends up in the config.
                -->
                <p class="mt-1 text-[11px] text-gray-500 dark:text-gray-400">
                  {{ $t('nodeRules.intervalHint') }}
                </p>
              </div>
              <div>
                <label class="block text-xs font-semibold text-gray-500 dark:text-gray-400 mb-1.5">Tolerance (ms)</label>
                <Input
                  v-model.number="filterForm.test_tolerance"
                  type="number"
                  min="0"
                  placeholder="200"
                />
              </div>
            </div>
          </div>
        </div>
      </div>

      <template #footer>
        <Button variant="secondary" size="sm" action @click="cancelFilterEdit">
          {{ t('nodeRules.cancel') }}
        </Button>
        <Button variant="primary" size="sm" action @click="saveFilter">
          {{ t('nodeRules.save') }}
        </Button>
      </template>
    </Dialog>

    <!-- Create/Edit Group Modal -->
    <Dialog
      :visible="editingGroupId !== null"
      @update:visible="(v: boolean) => { if (!v) cancelGroupEdit() }"
      modal
      class="w-full max-w-md"
    >
      <!-- The header slot rather than `:header`, so the section icon survives
        the move onto the shared dialog. -->
      <template #header>
        <h3 class="text-sm font-bold text-gray-900 dark:text-white flex items-center gap-2">
          <SparklesIcon class="h-5 w-5 text-primary-500" />
          {{ editingGroupId === '' ? $t('nodeRules.modal.addGroup') : $t('nodeRules.modal.editGroup') }}
        </h3>
      </template>

      <div class="space-y-3">
        <div>
          <label class="block text-xs font-semibold text-gray-500 dark:text-gray-400 mb-1.5">Group Name</label>
          <Input
            v-model="groupForm.name"
            :placeholder="t('nodeRules.groupName')"
          />
        </div>

        <div>
          <label class="block text-xs font-semibold text-gray-500 dark:text-gray-400 mb-1.5 flex justify-between">
            <span>Priority</span>
            <span class="text-gray-400 font-normal">lower number = matched first</span>
          </label>
          <Input
            v-model.number="groupForm.priority"
            type="number"
          />
        </div>

        <div class="space-y-2">
          <label class="block text-xs font-semibold text-gray-500 dark:text-gray-400">Select Filters to Include</label>
          <div class="flex flex-wrap gap-2 pt-1">
            <button
              v-for="f in filters"
              :key="f.id"
              type="button"
              @click="toggleGroupFilter(f.id)"
              class="px-2 py-1 text-xs font-semibold rounded-control border transition-all cursor-pointer"
              :class="groupForm.filter_ids.includes(f.id)
              ? 'bg-primary-600 border-primary-600 text-white'
              : 'bg-gray-50 dark:bg-slate-800 border-gray-200 dark:border-gray-700 text-gray-750 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-slate-700'"
            >
              {{ f.name }}
            </button>
          </div>
          <div v-if="!filters.length" class="text-xs text-gray-400 py-1">
            No filters available. Please add a filter first.
          </div>
        </div>

        <!--
          Outbounds named directly. Separate from the filter chips above
          because they are a different kind of member: a `direct` outbound is
          never collected by a filter's matchers, so it can only reach a group
          through here.
        -->
        <div class="space-y-2 border-t border-gray-100 dark:border-gray-800 pt-3">
          <label class="block text-xs font-semibold text-gray-500 dark:text-gray-400">
            {{ t('nodeRules.groupExtraTags') }}
            <span class="font-normal text-gray-400">{{ t('nodeRules.groupExtraTagsHint') }}</span>
          </label>

          <div v-if="groupForm.extra_tags.length" class="flex flex-wrap gap-1">
            <span
              v-for="tag in groupForm.extra_tags"
              :key="tag"
              class="inline-flex items-center gap-1 px-2 py-0.5 rounded-control text-[11px] bg-emerald-50/60 dark:bg-emerald-950/10 border border-emerald-200/50 dark:border-emerald-900/30 text-emerald-700 dark:text-emerald-400 font-medium"
            >
              {{ tag }}
              <button type="button" class="text-emerald-500 hover:text-red-500 cursor-pointer" @click="removeGroupTag(tag)">
                <XMarkIcon class="h-3 w-3" />
              </button>
            </span>
          </div>

          <SearchPicker
            v-if="endpointTags.length"
            :options="nodeOptions"
            :placeholder="t('nodeRules.groupAddTag')"
            :empty-message="t('nodeRules.noNodeMatches')"
            @select="addGroupTag"
          />
          <div v-else class="text-xs text-gray-400 py-1">{{ t('nodeRules.groupTagsEmpty') }}</div>
        </div>
      </div>

      <template #footer>
        <Button variant="secondary" size="sm" action @click="cancelGroupEdit">
          {{ t('nodeRules.cancel') }}
        </Button>
        <Button variant="primary" size="sm" action @click="saveGroup">
          {{ t('nodeRules.save') }}
        </Button>
      </template>
    </Dialog>
  </div>
</template>

<style scoped>
/*
 * Edit ⇄ Preview.
 *
 * 220ms in / 150ms out on the same curve the select panel and the nav pill use
 * — below ~150ms a transition stops reading as motion and just looks like a
 * flicker (DESIGN.md §8). The two directions travel opposite ways so the swap
 * reads as one view moving aside for the other rather than two unrelated fades.
 */
.mode-swap-enter-active {
  transition:
    opacity 0.22s ease-out,
    transform 0.22s cubic-bezier(0.32, 0.72, 0, 1);
}

.mode-swap-leave-active {
  transition:
    opacity 0.15s ease-in,
    transform 0.15s ease-in;
}

.mode-swap-enter-from {
  opacity: 0;
  transform: translateY(8px);
}

.mode-swap-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}

/* Same treatment the sidebar gives its pill: keep the cross-fade, drop the
   travel, and shorten it. */
@media (prefers-reduced-motion: reduce) {
  .mode-swap-enter-active,
  .mode-swap-leave-active {
    transition: opacity 0.12s linear;
  }

  .mode-swap-enter-from,
  .mode-swap-leave-to {
    transform: none;
  }
}

.node-rule-card {
  box-shadow:
    0 8px 22px rgba(15, 23, 42, 0.045),
    inset 0 1px 0 rgba(255, 255, 255, 0.42);
}

.node-rule-card:hover {
  box-shadow:
    0 12px 28px rgba(15, 23, 42, 0.065),
    inset 0 1px 0 rgba(255, 255, 255, 0.48);
}

.node-rule-primary-button {
  box-shadow: none;
}
</style>
