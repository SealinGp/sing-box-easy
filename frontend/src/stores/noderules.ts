import { defineStore } from 'pinia'
import { ref } from 'vue'
import { nodeRulesService } from '../services'
import type {
  Filter,
  Group,
  FilterInput,
  GroupInput,
  KeywordEntry,
  FilterTemplate,
  PreviewResult,
} from '../types/noderules'

export const useNodeRulesStore = defineStore('nodeRules', () => {
  const filters = ref<Filter[]>([])
  const groups = ref<Group[]>([])
  const keywords = ref<KeywordEntry[]>([])
  const templates = ref<FilterTemplate[]>([])
  const preview = ref<PreviewResult | null>(null)
  const loading = ref(false)
  const error = ref('')

  function setError(e: unknown) {
    error.value = e instanceof Error ? e.message : String(e)
  }

  async function fetchAll() {
    loading.value = true
    error.value = ''
    try {
      const res = await nodeRulesService.getAll()
      filters.value = res.data.filters || []
      groups.value = res.data.groups || []
    } catch (e) {
      setError(e)
    } finally {
      loading.value = false
    }
  }

  async function fetchCatalog() {
    try {
      const [kw, tpl] = await Promise.all([
        nodeRulesService.getKeywords(),
        nodeRulesService.getTemplates(),
      ])
      keywords.value = kw.data.keywords || []
      templates.value = tpl.data.templates || []
    } catch (e) {
      setError(e)
    }
  }

  async function createFilter(input: FilterInput) {
    await nodeRulesService.createFilter(input)
    await fetchAll()
  }

  async function updateFilter(id: string, input: FilterInput) {
    await nodeRulesService.updateFilter(id, input)
    await fetchAll()
  }

  async function deleteFilter(id: string) {
    await nodeRulesService.deleteFilter(id)
    await fetchAll()
  }

  async function applyTemplate(id: string) {
    await nodeRulesService.applyTemplate(id)
    await fetchAll()
  }

  async function createGroup(input: GroupInput) {
    await nodeRulesService.createGroup(input)
    await fetchAll()
  }

  async function updateGroup(id: string, input: GroupInput) {
    await nodeRulesService.updateGroup(id, input)
    await fetchAll()
  }

  async function deleteGroup(id: string) {
    await nodeRulesService.deleteGroup(id)
    await fetchAll()
  }

  async function runPreview() {
    loading.value = true
    error.value = ''
    try {
      const res = await nodeRulesService.preview()
      preview.value = res.data
    } catch (e) {
      setError(e)
    } finally {
      loading.value = false
    }
  }

  async function apply() {
    loading.value = true
    error.value = ''
    try {
      await nodeRulesService.apply()
    } catch (e) {
      setError(e)
      throw e
    } finally {
      loading.value = false
    }
  }

  return {
    filters,
    groups,
    keywords,
    templates,
    preview,
    loading,
    error,
    fetchAll,
    fetchCatalog,
    createFilter,
    updateFilter,
    deleteFilter,
    applyTemplate,
    createGroup,
    updateGroup,
    deleteGroup,
    runPreview,
    apply,
  }
})
