<script setup lang="ts">
import { watch as watchDraft } from 'vue'
import { useWorkspaceEdit } from '../composables/workspaceContext'
const markDraftEdited = useWorkspaceEdit()
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import DNSRuleConditions from './DNSRuleConditions.vue'
import Button from './Button.vue'
import IconButton from './IconButton.vue'
import { Select } from '../volt'
import { readDNSRuleDraft, invalidDNSConditionGroup, type DNSRuleDraft } from '../utils/dnsRuleDraft'

const rule = defineModel<DNSRuleDraft>({ required: true })
watchDraft(rule, () => markDraftEdited(), { deep: true })
defineProps<{ ruleSetOptions: { value: string; label: string }[] }>()
const { t } = useI18n()
const modes = computed(() => [
  { value: 'and', label: t('dns.rules.logical.all') },
  { value: 'or', label: t('dns.rules.logical.any') },
])
const removed = ref<{ index: number; rule: DNSRuleDraft } | null>(null)
const revision = ref(0)
function add(group = false) {
  rule.value.rules = [...rule.value.rules, readDNSRuleDraft(group ? { type: 'logical', mode: 'and', rules: [{}] } : {})]
}
function remove(index: number) {
  removed.value = { index, rule: rule.value.rules[index] }
  rule.value.rules = rule.value.rules.filter((_: unknown, i: number) => i !== index)
  revision.value++
}
function undo() {
  if (!removed.value) return
  rule.value.rules.splice(removed.value.index, 0, removed.value.rule)
  removed.value = null
  revision.value++
}
</script>

<template>
  <div class="min-w-0 space-y-3">
    <template v-if="rule.type === 'logical'">
      <Select v-model="rule.mode" :options="modes" option-label="label" option-value="value"
        :aria-label="t('dns.rules.logical.mode')" class="w-full" />
      <div v-for="(_, index) in rule.rules" :key="`${revision}-${index}`" class="min-w-0 rounded-control border border-border p-3">
        <div class="mb-2 flex items-center justify-between gap-2">
          <span class="text-xs font-medium text-text-secondary">{{ t('dns.rules.logical.condition', { index: index + 1 }) }}</span>
          <IconButton action="delete" size="sm" @click="remove(index)" />
        </div>
        <DNSConditionEditor v-model="rule.rules[index]" :rule-set-options="ruleSetOptions" />
        <p v-if="rule.rules[index].type !== 'logical' && invalidDNSConditionGroup(rule.rules[index], true)" class="mt-2 text-sm text-danger" role="alert">{{ t('dns.rules.logical.emptyCondition') }}</p>
      </div>
      <p v-if="!rule.rules.length" class="text-sm text-danger" role="alert">{{ t('dns.rules.logical.empty') }}</p>
      <div class="flex flex-wrap gap-2">
        <Button size="sm" variant="secondary" @click="add()">{{ t('dns.rules.logical.add') }}</Button>
        <Button size="sm" variant="ghost" @click="add(true)">{{ t('dns.rules.logical.addGroup') }}</Button>
        <Button v-if="removed" size="sm" variant="ghost" @click="undo">{{ t('dns.rules.logical.undo') }}</Button>
      </div>
    </template>
    <DNSRuleConditions v-else
      v-model:rule-set="rule.rule_set" v-model:domain="rule.domain"
      v-model:domain-suffix="rule.domain_suffix" v-model:domain-keyword="rule.domain_keyword"
      v-model:geosite="rule.geosite" v-model:query-type="rule.query_type"
      :rule-set-options="ruleSetOptions" />
    <label class="flex min-h-9 cursor-pointer items-center gap-2 text-sm text-text-secondary">
      <input v-model="rule.invert" type="checkbox" class="h-4 w-4 accent-primary-600" />
      {{ t('dns.rules.logical.invert') }}
    </label>
  </div>
</template>
