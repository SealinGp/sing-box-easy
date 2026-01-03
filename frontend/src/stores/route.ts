import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { RuleSet } from '../types/api'
import { routeService } from '../services'

export const useRouteStore = defineStore('route', () => {
  // Shared state - Rule Sets are needed by multiple components (DNSRules, RuleSets, etc.)
  const ruleSets = ref<RuleSet[]>([])
  const loading = ref(false)

  // Actions for Rule Sets (shared across components)
  const fetchRuleSets = async () => {
    loading.value = true
    try {
      const { data } = await routeService.getRuleSets()
      ruleSets.value = data.rule_sets || []
      return data.rule_sets
    } catch (err: any) {
      console.error('Failed to fetch rule sets:', err)
      throw err
    } finally {
      loading.value = false
    }
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

  const deleteRuleSet = async (tag: string) => {
    loading.value = true
    try {
      await routeService.deleteRuleSet(tag)
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
    loading,

    // Actions
    fetchRuleSets,
    addRuleSet,
    updateRuleSet,
    deleteRuleSet
  }
})