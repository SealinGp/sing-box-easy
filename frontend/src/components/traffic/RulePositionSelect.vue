<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Select from '../../volt/Select.vue'

const props = defineProps<{ rules: Record<string, unknown>[]; label: string }>()
const model = defineModel<number>({ required: true })
const { t } = useI18n()
function describe(rule: Record<string, unknown>) {
  const destination = rule.outbound ?? rule.server ?? rule.action ?? ''
  const conditions = Object.entries(rule)
    .filter(([key]) => !['outbound', 'server', 'action', 'tag', 'type'].includes(key))
    .map(([key, value]) => `${key} ∈ ${Array.isArray(value) ? value.map(v => typeof v === 'object' ? JSON.stringify(v) : v).join(', ') : typeof value === 'object' ? JSON.stringify(value) : value}`)
    .join(' · ')
  return [rule.tag, conditions || t('managed.allTraffic'), destination && `→ ${destination}`].filter(Boolean).join(' · ')
}
const options = computed(() => [
  ...props.rules.map((rule, value) => ({ value, label: t('managed.beforeRule', { rule: describe(rule) }) })),
  { value: props.rules.length, label: t('managed.afterAllRules') },
])
</script>

<template>
  <div class="min-w-0 space-y-2">
    <label class="block text-sm font-medium">{{ label }}</label>
    <Select v-model="model" :options="options" option-label="label" option-value="value"
      filter :aria-label="label" class="w-full" :filter-placeholder="t('common.search')">
      <template #option="{ option }"><span class="max-w-[min(70vw,40rem)] whitespace-normal break-words text-xs">{{ option.label }}</span></template>
      <template #value="{ value }"><span class="block whitespace-normal break-words text-xs">{{ options.find(option => option.value === value)?.label }}</span></template>
    </Select>
  </div>
</template>
