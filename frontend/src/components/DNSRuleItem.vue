<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowRightIcon, Bars3Icon, ChevronDownIcon } from '@heroicons/vue/24/outline'
import type { DNSRule } from '../types/api'
import ListRow from './ListRow.vue'
import ListField from './ListField.vue'
import IconButton from './IconButton.vue'
import StatusIndicator from './StatusIndicator.vue'
import RoutingConditionNode from './RoutingConditionNode.vue'
import { dnsConditionTree, dnsRuleOutcome } from '../utils/dnsRulePresentation'
import { actionOf, isTerminalAction, resolveDNSRuleActionFields, RESPONSE_CONDITION_KEYS } from '../schemas/dnsRuleActionFields'
import { isFieldFilled } from '../schemas/optionSchema'
import { humanizeFieldName } from '../utils/fieldLabels'

const props = defineProps<{
  rule: DNSRule
  index: number
  reorderable: boolean
  orphan: boolean
  rowAttrs: Record<string, unknown>
  handleAttrs: Record<string, unknown>
}>()
defineEmits<{ edit: []; delete: [] }>()
const { t, te } = useI18n()
const expanded = ref(false)
const record = computed(() => props.rule as Record<string, unknown>)
const condition = computed(() => dnsConditionTree(record.value))
const action = computed(() => actionOf(record.value))
const continues = computed(() => !isTerminalAction(action.value))
const outcome = computed(() => {
  const result = dnsRuleOutcome(record.value, t)
  if (action.value !== 'predefined') return result
  const answer = record.value.answer
  const answers = Array.isArray(answer) ? answer : answer ? [answer] : []
  return answers.length ? result + ' · ' + (answers.length === 1
    ? String(answers[0]) : t('dns.rules.table.answerCount', { count: answers.length })) : result
})
const options = computed(() => resolveDNSRuleActionFields(action.value)
  .filter(field => !RESPONSE_CONDITION_KEYS.includes(field.key) && isFieldFilled(record.value[field.key]))
  .map(field => ({
    key: field.key,
    label: te(field.labelKey) ? t(field.labelKey) : humanizeFieldName(field.key),
    value: typeof record.value[field.key] === 'object'
      ? JSON.stringify(record.value[field.key]) : String(record.value[field.key]),
  })))
</script>

<template>
  <ListRow v-bind="rowAttrs">
    <template #leading>
      <span class="inline-flex min-w-8 justify-center text-sm font-semibold tabular-nums text-text-secondary"
        :aria-label="t('route.rules.priority.position', { position: index + 1 })">{{ index + 1 }}</span>
      <button v-if="reorderable" v-bind="handleAttrs" class="list-drag-handle"
        :aria-label="t('dns.rules.reorder.handle', { position: index + 1 })">
        <Bars3Icon class="h-4 w-4" aria-hidden="true" />
      </button>
    </template>
    <div class="sentence-shell col-span-2 min-w-0">
      <details :open="expanded" @toggle="expanded = ($event.target as HTMLDetailsElement).open">
        <summary class="rule-summary rounded-control focus-visible:outline-2 focus-visible:outline-primary-500">
          <span class="condition-summary min-w-0 truncate text-sm" :aria-label="t('dns.rules.table.conditions')">
            <RoutingConditionNode :node="condition" label-prefix="dns.rules.form.fields" compact />
          </span>
          <span class="outcome-summary flex min-w-0 items-start gap-2 text-sm" :aria-label="t('dns.rules.table.action')">
            <ArrowRightIcon class="mt-1 h-3.5 w-3.5 shrink-0 text-text-secondary" aria-hidden="true" />
            <span class="min-w-0 break-words font-medium text-text-primary">
              {{ outcome }}
              <span v-if="continues" class="block text-xs font-normal text-text-secondary">{{ t('route.rules.sentence.continue') }}</span>
            </span>
          </span>
          <span class="expand-control flex items-center justify-end gap-1 text-[11px] text-text-secondary">
            {{ t(expanded ? 'route.rules.sentence.collapse' : 'route.rules.sentence.expand') }}
            <ChevronDownIcon class="h-3.5 w-3.5 shrink-0" :class="{ 'rotate-180': expanded }" aria-hidden="true" />
          </span>
        </summary>
        <div class="mt-2 space-y-3 border-t border-border py-3">
          <RoutingConditionNode :node="condition" label-prefix="dns.rules.form.fields" />
          <p class="break-words text-sm font-medium">{{ outcome }}</p>
          <div v-if="options.length" class="space-y-1 break-all text-xs">
            <ListField v-for="field in options" :key="field.key" :label="field.label" :value="field.value" />
          </div>
          <p class="text-xs text-text-secondary">{{ t(continues ? 'route.rules.sentence.continue' : 'route.rules.sentence.stop') }}</p>
        </div>
      </details>
      <StatusIndicator v-if="orphan" variant="danger" class="mt-2">
        {{ t('dns.rules.parallel.orphan') }} · {{ t('dns.rules.parallel.orphanHint') }}
      </StatusIndicator>
    </div>
    <template v-if="!reorderable" #actions>
      <IconButton action="edit" @click="$emit('edit')" />
      <IconButton action="delete" @click="$emit('delete')" />
    </template>
  </ListRow>
</template>

<style scoped>
.sentence-shell { container-type: inline-size; }
.rule-summary {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(12rem, .75fr) 3.5rem;
  align-items: center;
  gap: 1rem;
  min-height: 1.75rem;
  cursor: pointer;
  list-style: none;
}
.rule-summary::-webkit-details-marker { display: none; }
@container (max-width: 600px) {
  .rule-summary { grid-template-columns: minmax(0, 1fr) 3.5rem; gap: .375rem .75rem; }
  .condition-summary { grid-column: 1; }
  .outcome-summary { grid-column: 1; grid-row: 2; }
  .expand-control { grid-column: 2; grid-row: 1 / 3; }
}
</style>
