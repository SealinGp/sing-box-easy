<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { storeToRefs } from 'pinia'
import { useNodeRulesStore } from '../../stores/noderules'
import PopConfirm from '../../components/PopConfirm.vue'
import type { Filter, Group, Matcher, MatcherType, FilterOutboundType } from '../../types/noderules'
import { URLTEST_DEFAULTS } from '../../types/noderules'

const { t } = useI18n()
const store = useNodeRulesStore()
const { filters, groups, keywords, templates, preview, loading, error } = storeToRefs(store)

const applying = ref(false)
const notice = ref('')

// ---- Preview expand state ----
// Vue 3 tracks Set.has/add/delete via its collection proxy, so reactive(Set)
// re-renders the template on mutation. Do NOT switch to ref<Set>() without also
// reassigning on every change — a mutated ref<Set> would not trigger updates.
const expandedPreview = reactive<Set<string>>(new Set())
const showUnmatched = ref(false)

function togglePreview(id: string) {
  if (expandedPreview.has(id)) expandedPreview.delete(id)
  else expandedPreview.add(id)
}

// ---- Filter editor state ----
const editingFilterId = ref<string | null>(null) // null = not editing; '' = creating
const filterForm = reactive<{
  name: string
  outbound_type: FilterOutboundType
  priority: number
  matchers: Matcher[]
  test_url: string
  test_interval: string
  test_tolerance: number
}>({
  name: '',
  outbound_type: 'urltest',
  priority: 0,
  matchers: [],
  test_url: URLTEST_DEFAULTS.test_url,
  test_interval: URLTEST_DEFAULTS.test_interval,
  test_tolerance: URLTEST_DEFAULTS.test_tolerance,
})

function startCreateFilter() {
  editingFilterId.value = ''
  filterForm.name = ''
  filterForm.outbound_type = 'urltest'
  filterForm.priority = (filters.value.filter((f) => !f.is_fallback).length + 1) * 10
  filterForm.matchers = []
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

function addCodeMatcher(code: string) {
  if (!code) return
  if (filterForm.matchers.some((m) => m.type === 'code' && m.value === code)) return
  filterForm.matchers = [...filterForm.matchers, { type: 'code', value: code }]
}

async function saveFilter() {
  notice.value = ''
  const input = {
    name: filterForm.name.trim(),
    outbound_type: filterForm.outbound_type,
    priority: filterForm.priority,
    matchers: filterForm.matchers.filter((m) => m.value.trim() !== ''),
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
const groupForm = reactive<{ name: string; priority: number; filter_ids: string[] }>({
  name: '',
  priority: 0,
  filter_ids: [],
})

function startCreateGroup() {
  editingGroupId.value = ''
  groupForm.name = ''
  groupForm.priority = (groups.value.length + 1) * 10
  groupForm.filter_ids = []
}

function startEditGroup(g: Group) {
  editingGroupId.value = g.id
  groupForm.name = g.name
  groupForm.priority = g.priority
  groupForm.filter_ids = [...g.filter_ids]
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
  const input = { name: groupForm.name.trim(), priority: groupForm.priority, filter_ids: groupForm.filter_ids }
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
  <div class="space-y-6">
    <!-- Header: subtitle + actions (page title is owned by the Outbounds TabNav) -->
    <div class="flex flex-wrap items-center justify-between gap-3">
      <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('nodeRules.subtitle') }}</p>
      <div class="flex gap-2">
        <button class="btn btn-sm" :disabled="loading" @click="store.runPreview()">
          {{ t('nodeRules.preview') }}
        </button>
        <button class="btn btn-sm btn-primary" :disabled="applying" @click="doApply">
          {{ applying ? t('nodeRules.applying') : t('nodeRules.applyNow') }}
        </button>
      </div>
    </div>

    <div v-if="notice" class="alert alert-info text-sm">{{ notice }}</div>
    <div v-if="error" class="alert alert-error text-sm">{{ error }}</div>

    <!-- Templates -->
    <section v-if="templates.length" class="space-y-2">
      <h2 class="text-sm font-semibold text-gray-700 dark:text-gray-300">{{ t('nodeRules.templates') }}</h2>
      <div class="flex flex-wrap gap-2">
        <button
          v-for="tpl in templates"
          :key="tpl.id"
          class="badge badge-outline gap-1 cursor-pointer hover:badge-primary"
          :title="tpl.description"
          @click="addTemplate(tpl.id)"
        >
          + {{ tpl.name }}
        </button>
      </div>
    </section>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <!-- Filters panel -->
      <section class="card bg-base-100 shadow-sm border border-base-200 overflow-visible">
        <div class="card-body p-4 space-y-3">
          <div class="flex items-center justify-between">
            <h2 class="card-title text-base">{{ t('nodeRules.filters') }}</h2>
            <button class="btn btn-xs btn-primary" @click="startCreateFilter">+ {{ t('nodeRules.addFilter') }}</button>
          </div>

          <!-- Filter create/edit form -->
          <div v-if="editingFilterId !== null" class="rounded-lg border border-base-300 p-3 space-y-2 bg-base-200/40">
            <div class="flex gap-2">
              <input v-model="filterForm.name" :placeholder="t('nodeRules.filterName')" class="input input-sm input-bordered flex-1" />
              <select v-model="filterForm.outbound_type" class="select select-sm select-bordered">
                <option value="urltest">urltest</option>
                <option value="selector">selector</option>
              </select>
            </div>

            <!-- Priority (with a label explaining the number) -->
            <label class="flex items-center gap-2">
              <span class="text-xs text-gray-500 dark:text-gray-400 w-28 shrink-0">{{ t('nodeRules.priorityLabel') }}</span>
              <input v-model.number="filterForm.priority" type="number" class="input input-xs input-bordered w-24" />
              <span class="text-xs text-gray-400">{{ t('nodeRules.priorityHint') }}</span>
            </label>

            <!-- urltest health-check settings (only relevant for urltest) -->
            <div v-if="filterForm.outbound_type === 'urltest'" class="space-y-1 rounded-md bg-base-100/60 p-2 border border-base-200">
              <div class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('nodeRules.urltestSettings') }}</div>
              <label class="flex items-center gap-2">
                <span class="text-xs text-gray-500 w-28 shrink-0">{{ t('nodeRules.testUrl') }}</span>
                <input v-model="filterForm.test_url" class="input input-xs input-bordered flex-1" placeholder="http://www.gstatic.com/generate_204" />
              </label>
              <div class="flex gap-2">
                <label class="flex items-center gap-2 flex-1">
                  <span class="text-xs text-gray-500 w-28 shrink-0">{{ t('nodeRules.testInterval') }}</span>
                  <input v-model="filterForm.test_interval" class="input input-xs input-bordered w-24" placeholder="10s" />
                </label>
                <label class="flex items-center gap-2 flex-1">
                  <span class="text-xs text-gray-500 shrink-0">{{ t('nodeRules.testTolerance') }}</span>
                  <input v-model.number="filterForm.test_tolerance" type="number" min="0" class="input input-xs input-bordered w-24" placeholder="200" />
                </label>
              </div>
            </div>

            <!-- How matching works (collapsible explainer) -->
            <details class="rounded-md border border-base-200 bg-base-100/60 text-xs">
              <summary class="cursor-pointer select-none px-2 py-1.5 font-medium text-gray-600 dark:text-gray-300">
                ⓘ {{ t('nodeRules.help.toggle') }}
              </summary>
              <div class="px-2 pb-2 pt-1 space-y-2 text-gray-600 dark:text-gray-300">
                <p>{{ t('nodeRules.help.intro') }}</p>
                <ul class="space-y-1.5">
                  <li>
                    <code class="badge badge-ghost badge-xs">keyword</code>
                    <span class="font-medium">{{ t('nodeRules.help.keywordTitle') }}</span>
                    <span class="block text-gray-500 dark:text-gray-400">{{ t('nodeRules.help.keywordDesc') }}</span>
                  </li>
                  <li>
                    <code class="badge badge-ghost badge-xs">code</code>
                    <span class="font-medium">{{ t('nodeRules.help.codeTitle') }}</span>
                    <span class="block text-gray-500 dark:text-gray-400">{{ t('nodeRules.help.codeDesc') }}</span>
                  </li>
                  <li>
                    <code class="badge badge-ghost badge-xs">emoji</code>
                    <span class="font-medium">{{ t('nodeRules.help.emojiTitle') }}</span>
                    <span class="block text-gray-500 dark:text-gray-400">{{ t('nodeRules.help.emojiDesc') }}</span>
                  </li>
                </ul>
                <p class="text-gray-500 dark:text-gray-400">{{ t('nodeRules.help.multi') }}</p>

                <!-- Supported region codes (catalog from the backend) -->
                <div v-if="keywords.length" class="space-y-1 border-t border-base-200 pt-2">
                  <div class="font-medium text-gray-600 dark:text-gray-300">{{ t('nodeRules.help.codesTitle') }}</div>
                  <div class="flex flex-wrap gap-1">
                    <button
                      v-for="kw in keywords"
                      :key="kw.code"
                      type="button"
                      class="badge badge-ghost badge-xs cursor-pointer hover:badge-primary"
                      :title="kw.synonyms.join(' · ')"
                      @click="addCodeMatcher(kw.code)"
                    >
                      {{ kw.code }} · {{ kw.label }}
                    </button>
                  </div>
                  <div class="text-gray-400">{{ t('nodeRules.help.codesHint') }}</div>
                </div>
              </div>
            </details>

            <!-- Matchers -->
            <div class="space-y-1">
              <div v-for="(m, idx) in filterForm.matchers" :key="idx" class="flex gap-2 items-center">
                <select v-model="m.type" class="select select-xs select-bordered">
                  <option value="keyword">keyword</option>
                  <option value="code">code</option>
                  <option value="emoji">emoji</option>
                </select>
                <input v-model="m.value" class="input input-xs input-bordered flex-1" :placeholder="t('nodeRules.matcherValue')" />
                <button class="btn btn-xs btn-ghost" @click="removeMatcher(idx)">✕</button>
              </div>
              <div class="flex flex-wrap gap-2 items-center pt-1">
                <button class="btn btn-xs" @click="addMatcher('keyword')">+ {{ t('nodeRules.addMatcher') }}</button>
                <select class="select select-xs select-bordered" @change="(e) => { addCodeMatcher((e.target as HTMLSelectElement).value); (e.target as HTMLSelectElement).value = '' }">
                  <option value="">{{ t('nodeRules.addCountry') }}</option>
                  <option v-for="kw in keywords" :key="kw.code" :value="kw.code">{{ kw.label }} ({{ kw.code }})</option>
                </select>
              </div>
            </div>

            <div class="flex justify-end gap-2 pt-1">
              <button class="btn btn-xs btn-ghost" @click="cancelFilterEdit">{{ t('nodeRules.cancel') }}</button>
              <button class="btn btn-xs btn-primary" @click="saveFilter">{{ t('nodeRules.save') }}</button>
            </div>
          </div>

          <!-- Filter list -->
          <ul class="space-y-2">
            <li
              v-for="f in filters"
              :key="f.id"
              class="rounded-lg border border-base-200 p-2 flex items-start justify-between gap-2"
            >
              <div class="min-w-0">
                <div class="flex items-center gap-2 flex-wrap">
                  <span class="font-medium truncate">{{ f.name }}</span>
                  <span class="badge badge-xs">{{ f.outbound_type }}</span>
                  <span v-if="f.is_fallback" class="badge badge-xs badge-warning">{{ t('nodeRules.fallback') }}</span>
                  <span class="text-xs text-gray-400">P{{ f.priority }}</span>
                </div>
                <div v-if="!f.is_fallback" class="flex flex-wrap gap-1 mt-1">
                  <span v-for="(m, i) in f.matchers" :key="i" class="badge badge-ghost badge-xs">{{ m.value }}</span>
                  <span v-if="!f.matchers.length" class="text-xs text-gray-400">{{ t('nodeRules.noMatchers') }}</span>
                </div>
                <div v-else class="text-xs text-gray-400 mt-1">{{ t('nodeRules.fallbackHint') }}</div>
              </div>
              <div class="flex gap-1 shrink-0">
                <button class="btn btn-xs btn-ghost" @click="startEditFilter(f)">{{ t('nodeRules.edit') }}</button>
                <PopConfirm
                  v-if="!f.is_fallback"
                  :message="t('nodeRules.confirmDeleteFilter', { name: f.name })"
                  :confirm-label="t('nodeRules.delete')"
                  tone="danger"
                  @confirm="removeFilter(f)"
                >
                  {{ t('nodeRules.delete') }}
                </PopConfirm>
              </div>
            </li>
          </ul>
        </div>
      </section>

      <!-- Groups panel -->
      <section class="card bg-base-100 shadow-sm border border-base-200 overflow-visible">
        <div class="card-body p-4 space-y-3">
          <div class="flex items-center justify-between">
            <h2 class="card-title text-base">{{ t('nodeRules.groups') }}</h2>
            <button class="btn btn-xs btn-primary" @click="startCreateGroup">+ {{ t('nodeRules.addGroup') }}</button>
          </div>

          <!-- Group create/edit form -->
          <div v-if="editingGroupId !== null" class="rounded-lg border border-base-300 p-3 space-y-2 bg-base-200/40">
            <input v-model="groupForm.name" :placeholder="t('nodeRules.groupName')" class="input input-sm input-bordered w-full" />
            <label class="flex items-center gap-2">
              <span class="text-xs text-gray-500 dark:text-gray-400 w-28 shrink-0">{{ t('nodeRules.priorityLabel') }}</span>
              <input v-model.number="groupForm.priority" type="number" class="input input-xs input-bordered w-24" />
              <span class="text-xs text-gray-400">{{ t('nodeRules.priorityHint') }}</span>
            </label>
            <div class="flex flex-wrap gap-2">
              <label
                v-for="f in filters"
                :key="f.id"
                class="badge gap-1 cursor-pointer"
                :class="groupForm.filter_ids.includes(f.id) ? 'badge-primary' : 'badge-outline'"
                @click="toggleGroupFilter(f.id)"
              >
                {{ f.name }}
              </label>
            </div>
            <div class="flex justify-end gap-2 pt-1">
              <button class="btn btn-xs btn-ghost" @click="cancelGroupEdit">{{ t('nodeRules.cancel') }}</button>
              <button class="btn btn-xs btn-primary" @click="saveGroup">{{ t('nodeRules.save') }}</button>
            </div>
          </div>

          <!-- Group list -->
          <ul class="space-y-2">
            <li v-for="g in groups" :key="g.id" class="rounded-lg border border-base-200 p-2 flex items-start justify-between gap-2">
              <div class="min-w-0">
                <div class="flex items-center gap-2">
                  <span class="font-medium truncate">{{ g.name }}</span>
                  <span class="text-xs text-gray-400">P{{ g.priority }}</span>
                </div>
                <div class="flex flex-wrap gap-1 mt-1">
                  <span v-for="fid in g.filter_ids" :key="fid" class="badge badge-ghost badge-xs">{{ filterName(fid) }}</span>
                  <span v-if="!g.filter_ids.length" class="text-xs text-gray-400">{{ t('nodeRules.noFilters') }}</span>
                </div>
              </div>
              <div class="flex gap-1 shrink-0">
                <button class="btn btn-xs btn-ghost" @click="startEditGroup(g)">{{ t('nodeRules.edit') }}</button>
                <PopConfirm
                  :message="t('nodeRules.confirmDeleteGroup', { name: g.name })"
                  :confirm-label="t('nodeRules.delete')"
                  tone="danger"
                  @confirm="removeGroup(g)"
                >
                  {{ t('nodeRules.delete') }}
                </PopConfirm>
              </div>
            </li>
            <li v-if="!groups.length" class="text-xs text-gray-400">{{ t('nodeRules.noGroups') }}</li>
          </ul>
        </div>
      </section>
    </div>

    <!-- Preview -->
    <section v-if="preview" class="card bg-base-100 shadow-sm border border-base-200">
      <div class="card-body p-4 space-y-3">
        <h2 class="card-title text-base">
          {{ t('nodeRules.previewResult') }}
          <span class="text-sm font-normal text-gray-400">({{ t('nodeRules.endpoints', { n: preview.endpoints }) }})</span>
        </h2>
        <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-2">
          <div v-for="pf in preview.filters" :key="pf.id" class="rounded-lg border border-base-200 p-2 self-start">
            <button
              class="w-full flex items-center justify-between gap-2 cursor-pointer"
              :disabled="!pf.member_count"
              @click="togglePreview(pf.id)"
            >
              <span class="font-medium truncate flex items-center gap-1 min-w-0">
                <span v-if="pf.member_count" class="text-gray-400 text-xs shrink-0">{{ expandedPreview.has(pf.id) ? '▾' : '▸' }}</span>
                <span class="truncate">{{ pf.name }}</span>
                <span v-if="pf.is_fallback" class="badge badge-xs badge-warning shrink-0">{{ t('nodeRules.fallback') }}</span>
              </span>
              <span class="badge badge-sm shrink-0" :class="pf.member_count ? 'badge-primary' : 'badge-ghost'">{{ pf.member_count }}</span>
            </button>
            <div v-if="expandedPreview.has(pf.id)" class="mt-2 max-h-44 overflow-y-auto space-y-0.5 border-t border-base-200 pt-2">
              <div
                v-for="(tag, i) in pf.members"
                :key="i"
                class="text-xs text-gray-600 dark:text-gray-300 truncate"
                :title="tag"
              >
                {{ tag }}
              </div>
            </div>
          </div>
        </div>

        <!-- Unmatched nodes (fall through to the fallback) -->
        <div v-if="preview.unmatched.length" class="rounded-lg border border-warning/40 bg-warning/5 p-2">
          <button class="w-full flex items-center gap-1 text-xs text-warning cursor-pointer" @click="showUnmatched = !showUnmatched">
            <span class="shrink-0">{{ showUnmatched ? '▾' : '▸' }}</span>
            <span>{{ t('nodeRules.unmatched', { n: preview.unmatched.length }) }}</span>
          </button>
          <div v-if="showUnmatched" class="mt-2 max-h-44 overflow-y-auto space-y-0.5 border-t border-warning/30 pt-2">
            <div
              v-for="(tag, i) in preview.unmatched"
              :key="i"
              class="text-xs text-gray-600 dark:text-gray-300 truncate"
              :title="tag"
            >
              {{ tag }}
            </div>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>
