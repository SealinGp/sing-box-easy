<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { outboundService, routeService } from '../../services'
import type { RouteRule, RuleSet, Outbound } from '../../types'
import RoutingRules from '../../components/RoutingRules.vue'
import RuleSets from '../../components/RuleSets.vue'
import FinalPolicy from '../../components/FinalPolicy.vue'
import { useToast } from 'primevue'

const toast = useToast()

// State
const loading = ref(false)
const activeTab = ref('rules')
const showAddRuleDialog = ref(false)
const showAddRuleSetDialog = ref(false)
const editingRule = ref<{ index: number; rule: RouteRule } | null>(null)
const editingRuleSet = ref<{ tag: string; ruleSet: RuleSet } | null>(null)

// Data
const rules = ref<RouteRule[]>([])
const ruleSets = ref<RuleSet[]>([])
const finalRoute = ref('proxy')
const outbounds = ref<Outbound[]>([])

// Form data
const ruleForm = ref<RouteRule>({
  action: 'route',
  outbound: '',
})

const ruleSetForm = ref<RuleSet>({
  tag: '',
  type: 'remote',
  format: 'source',
})

// Load data
async function loadData() {
  loading.value = true
  try {
    const [rulesRes, ruleSetsRes, finalRes, outboundsRes] = await Promise.all([
      routeService.getRouteRules(),
      routeService.getRuleSets(),
      routeService.getRouteFinal(),
      outboundService.getOutbounds(),
    ])
    

    rules.value = rulesRes.data.rules
    ruleSets.value = ruleSetsRes.data.rule_sets
    finalRoute.value = finalRes.data.final || 'proxy'    
    outbounds.value = outboundsRes.data.outbounds
  } catch (error) {
    console.error('Failed to load route data:', error)
  } finally {
    loading.value = false
  }
}

// Rule operations
async function handleAddRule(rule: RouteRule) {
  try {
    await routeService.addRouteRule(rule)
    showAddRuleDialog.value = false
    ruleForm.value = { action: 'route', outbound: '' }
    await loadData()
    toast.add({
      severity: 'success',
      summary: 'Success',
      detail: 'Add Route Rule successfully',
      life: 300000
    })
  } catch (error) {
    console.error('Failed to add rule:', error)
  }
}

async function handleEditRule(index: number, rule: RouteRule) {
  try {
    await routeService.updateRouteRule(index, rule)
    editingRule.value = null
    await loadData()
    toast.add({
      severity: 'success',
      summary: 'Success',
      detail: 'Update Route Rule successfully',
      life: 300000
    })
  } catch (error) {
    console.error('Failed to update rule:', error)
  }
}

async function handleDeleteRule(index: number) {
  if (!confirm('Are you sure you want to delete this rule?')) return

  try {
    await routeService.deleteRouteRule(index)
    await loadData()
  } catch (error) {
    console.error('Failed to delete rule:', error)
  }
}

// Rule Set operations
async function addRuleSet(ruleSet: RuleSet) {
  try {
    await routeService.addRuleSet(ruleSet)
    await loadData()
  } catch (error) {
    console.error('Failed to add rule set:', error)
  }
}

async function updateRuleSet(tag: string, ruleSet: RuleSet) {
  try {
    await routeService.updateRuleSet(tag, ruleSet)
    await loadData()
    
  } catch (error) {
    console.error('Failed to update rule set:', error)
  }
}

async function deleteRuleSet(tag: string) {
  if (!confirm(`Are you sure you want to delete rule set "${tag}"?`)) return

  try {
    await routeService.deleteRuleSet(tag)
    await loadData()
  } catch (error) {
    console.error('Failed to delete rule set:', error)
  }
}

// Final route operations
async function updateFinalRoute(value: string) {
  try {
    await routeService.updateRouteFinal(value)
  } catch (error) {
    console.error('Failed to update final route:', error)
  }
}

onMounted(loadData)
const tabs = [
   { id: 'rules', label: 'Routing Rules' },
  { id: 'ruleSets', label: 'Rule Sets' },
  { id: 'final', label: 'Final Policy' }
]
</script>

<template>
  <div class="p-8">
    <h2 class="text-3xl font-bold text-gray-900 dark:text-gray-100 mb-6">Route Configuration</h2>

    <!-- Tabs -->
    <div class="mb-6 border-b border-gray-200 dark:border-gray-700">
      <nav class="-mb-px flex space-x-8">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          @click="activeTab = tab.id"
          :class="[
            'py-2 px-1 border-b-2 font-medium text-sm transition-colors',
            activeTab === tab.id
              ? 'border-blue-500 text-blue-600 dark:text-blue-400'
              : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-300'
          ]"
        >
          {{ tab.label }}
        </button>
      </nav>
    </div>

    <!-- Routing Rules Tab -->
    <RoutingRules
      v-if="activeTab === 'rules'"
      v-model:showAddDialog="showAddRuleDialog"
      v-model:editingRule="editingRule"
      v-model:ruleForm="ruleForm"
      :rules="rules"
      :outbounds="outbounds"
      :loading="loading"
      @add-rule="handleAddRule"
      @edit-rule="handleEditRule"
      @delete-rule="handleDeleteRule"
    />

    <!-- Rule Sets Tab -->
    <RuleSets
      v-if="activeTab === 'ruleSets'"
      v-model:showAddDialog="showAddRuleSetDialog"
      v-model:editingRuleSet="editingRuleSet"
      v-model:ruleSetForm="ruleSetForm"
      :rule-sets="ruleSets"
      :loading="loading"
      @add-rule-set="addRuleSet"
      @edit-rule-set="updateRuleSet"
      @delete-rule-set="deleteRuleSet"
    />

    <!-- Final Policy Tab -->
    <FinalPolicy
      v-if="activeTab === 'final'"
      v-model:finalRoute="finalRoute"
      @update="updateFinalRoute"
    />
  </div>
</template>
