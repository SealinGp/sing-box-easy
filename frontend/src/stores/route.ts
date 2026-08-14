import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { RuleSet } from '../types/api'
import { routeService } from '../services'

export const useRouteStore = defineStore('route', () => {
  // Shared state - Rule Sets are needed by multiple components (DNSRules, RuleSets, etc.)
  const ruleSets = ref<RuleSet[]>([])
  const loading = ref(false)

  // Whether a fetch has ever succeeded. Distinguishes "no rule sets configured"
  // from "not loaded yet", which an empty array alone cannot express.
  const ruleSetsLoaded = ref(false)

  // Actions for Rule Sets (shared across components)
  const fetchRuleSets = async () => {
    loading.value = true
    try {
      const { data } = await routeService.getRuleSets()
      ruleSets.value = data.rule_sets || []
      ruleSetsLoaded.value = true
      return data.rule_sets
    } catch (err: any) {
      console.error('Failed to fetch rule sets:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  // Load once, for components that only need the list to populate a picker.
  //
  // `inFlight` collapses concurrent callers onto one request: the rule-set
  // picker mounts inside a dialog that can open several times per page visit,
  // and every mount would otherwise fire its own GET. Errors are swallowed by
  // the caller's perspective only in that they do not reject here — fetchRuleSets
  // already logs them, and an empty picker is a survivable degradation.
  let inFlight: Promise<unknown> | null = null

  const ensureRuleSets = async () => {
    if (ruleSetsLoaded.value) return ruleSets.value
    if (!inFlight) {
      inFlight = fetchRuleSets().catch(() => undefined).finally(() => {
        inFlight = null
      })
    }
    await inFlight
    return ruleSets.value
  }

  const addRuleSet = async (ruleSet: RuleSet) => {
    loading.value = true
    try {
      await routeService.addRuleSet(ruleSet)
      await fetchRuleSets()
    } catch (err: any) {
      console.error('Failed to add rule set:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  const updateRuleSet = async (tag: string, ruleSet: RuleSet) => {
    loading.value = true
    try {
      await routeService.updateRuleSet(tag, ruleSet)
      await fetchRuleSets()
    } catch (err: any) {
      console.error('Failed to update rule set:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  const deleteRuleSet = async (tag: string, opts?: { cascade?: boolean }) => {
    loading.value = true
    try {
      await routeService.deleteRuleSet(tag, opts)
      await fetchRuleSets()
    } catch (err: any) {
      console.error('Failed to delete rule set:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  return {
    // State
    ruleSets,
    ruleSetsLoaded,
    loading,

    // Actions
    fetchRuleSets,
    ensureRuleSets,
    addRuleSet,
    updateRuleSet,
    deleteRuleSet
  }
})