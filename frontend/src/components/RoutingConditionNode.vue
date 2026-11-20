<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'
import FieldHelp from './FieldHelp.vue'
import type { RoutingCondition } from '../utils/routeRulePresentation'
import { humanizeFieldName } from '../utils/fieldLabels'
import { FOCUS_PAGES, focusLink } from '../utils/focusTarget'

const props = defineProps<{ node: RoutingCondition; compact?: boolean }>()
const { t, te } = useI18n()
const field = computed(() => props.node.kind === 'field' ? props.node : null)
const group = computed(() => props.node.kind === 'group' ? props.node : null)
const values = computed(() => {
  const value = field.value?.value
  if (typeof value === 'boolean') return [String(value)]
  return (Array.isArray(value) ? value : [value]).map(item =>
    typeof item === 'object' ? JSON.stringify(item) : String(item))
})
const label = computed(() => {
  const key = field.value?.key ?? ''
  const path = `route.rules.fields.${key}`
  return (te(path) ? t(path) : humanizeFieldName(key)).replace(/[:：]\s*$/, '')
})
const operator = computed(() => {
  if (!values.value.length) return ''
  if (field.value?.key === 'rule_set') return t('route.rules.sentence.matches')
  return t(values.value.length > 1 ? 'route.rules.sentence.isAny' : 'route.rules.sentence.is')
})
const groupedValues = computed(() => values.value.length > 1 || field.value?.key === 'rule_set')
const connector = computed(() => group.value?.mode === 'configured' ? ' · ' :
  t(group.value?.mode === 'or' ? 'route.rules.sentence.or' : 'route.rules.sentence.and'))
const groupLabel = computed(() => t(`route.rules.sentence.${{
  and: 'all', or: 'any', not: 'not', configured: 'configured',
}[group.value?.mode ?? 'and']}`))
</script>

<template>
  <span v-if="compact" class="routing-expression">
    <template v-if="group">
      <span v-if="group.mode === 'not'" class="font-medium text-text-secondary">{{ t('route.rules.sentence.not') }} </span>
      <template v-if="!group.children.length">{{ t('route.rules.sentence.remaining') }}</template>
      <template v-else>
        <span v-if="group.children.length > 1 || group.mode === 'not'" class="text-text-secondary">(</span>
        <template v-for="(child, index) in group.children" :key="index">
          <span v-if="index" class="mx-1 text-xs font-medium text-primary-600 dark:text-primary-400">{{ connector }}</span>
          <RoutingConditionNode :node="child" compact />
        </template>
        <span v-if="group.children.length > 1 || group.mode === 'not'" class="text-text-secondary">)</span>
      </template>
    </template>
    <template v-else-if="field">
      <span class="text-text-secondary">{{ label }}</span>
      <code class="mx-1 text-[11px] font-medium text-text-secondary">{{ operator }}</code>
      <span v-if="groupedValues" class="text-text-secondary">(</span>
      <template v-for="(value, index) in values.slice(0, 2)" :key="index">
        <span v-if="index" class="text-text-secondary">, </span>
        <span class="font-medium text-text-primary">{{ value }}</span>
      </template>
      <span v-if="values.length > 2" class="text-xs text-text-secondary">, +{{ values.length - 2 }}</span>
      <span v-if="groupedValues" class="text-text-secondary">)</span>
    </template>
  </span>
  <div v-else-if="group" class="min-w-0">
    <p v-if="!group.children.length" class="text-sm text-text-secondary">{{ t('route.rules.sentence.remaining') }}</p>
    <div v-else class="overflow-hidden rounded-control border border-border" :class="{ 'border-dashed': group.mode === 'not' }">
      <div class="flex items-center gap-1 bg-bg-tertiary px-3 py-1.5 text-xs font-medium text-text-secondary">
        {{ groupLabel }}
        <code v-if="group.mode === 'and' || group.mode === 'or'" class="ml-1 text-[10px] font-normal">{{ group.mode.toUpperCase() }}</code>
        <FieldHelp v-if="group.mode === 'configured'" :label="groupLabel" :text="t('route.rules.sentence.ruleSetCombination')" />
      </div>
      <div class="space-y-1.5 p-3">
        <template v-for="(child, index) in group.children" :key="index">
          <p v-if="index" class="text-[11px] font-medium text-text-secondary">{{ connector }}</p>
          <RoutingConditionNode :node="child" />
        </template>
      </div>
    </div>
  </div>
  <div v-else-if="field" class="flex min-w-0 flex-wrap items-baseline gap-x-2 gap-y-1 text-xs leading-relaxed">
    <span class="text-text-secondary">{{ label }}</span>
    <code class="text-[11px] font-medium text-text-secondary">{{ operator }}</code>
    <span v-if="groupedValues" class="text-text-secondary">(</span>
    <template v-for="(value, index) in values" :key="index">
      <span v-if="index" class="text-text-secondary">,</span>
      <RouterLink v-if="field.key === 'rule_set'" :to="focusLink(FOCUS_PAGES.ruleSets, value)" class="break-all text-primary-600 underline focus-visible:outline-2 focus-visible:outline-primary-500 dark:text-primary-400">{{ value }}</RouterLink>
      <span v-else class="break-all font-medium text-text-primary">{{ value }}</span>
    </template>
    <span v-if="groupedValues" class="text-text-secondary">)</span>
  </div>
</template>
