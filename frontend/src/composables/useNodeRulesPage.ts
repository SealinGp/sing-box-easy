import { computed, onMounted, reactive, ref } from 'vue'
import { useFocusTarget } from './useFocusTarget'
import { useI18n } from 'vue-i18n'
import { storeToRefs } from 'pinia'
import { useNodeRulesStore } from '../stores/noderules'
import type { Filter, Group, Matcher, MatcherType, FilterOutboundType } from '../types/noderules'
import { URLTEST_DEFAULTS } from '../types/noderules'
import type { PickerOption } from '../components/SearchPicker.vue'

export function useNodeRulesPage() {
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
    // `?? []` throughout: this runs on every render of the page, so one field
    // arriving as null from an older backend must cost an empty list, not the
    // whole page (#14 — a filter with no members was sent as `members: null`).
    for (const pf of p.filters ?? []) for (const tag of pf.members ?? []) seen.add(tag)
    for (const tag of p.unmatched ?? []) seen.add(tag)
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

  // A managed outbound's tag IS its Filter's or Group's name, so the name is
  // what an Outbounds/Proxies link carries here.
  const { isFocused } = useFocusTarget(() => filters.value.length + groups.value.length > 0)

  onMounted(async () => {
    await Promise.all([store.fetchAll(), store.fetchCatalog()])
    await store.runPreview()
  })
  return { t, store, filters, groups, keywords, templates, preview, loading, error, applying, notice, mode, showPreview, outboundTypeOptions, modeOptions, matcherTypeOptions, editingFilterId, filterForm, endpointTags, nodeOptions, countryOptions, startCreateFilter, startEditFilter, cancelFilterEdit, addMatcher, removeMatcher, addMatcherNode, addCodeMatcher, addExclude, removeExclude, addExcludeNode, saveFilter, removeFilter, editingGroupId, groupForm, startCreateGroup, startEditGroup, addGroupTag, removeGroupTag, cancelGroupEdit, toggleGroupFilter, saveGroup, removeGroup, addTemplate, doApply, filterName, isFocused }
}
