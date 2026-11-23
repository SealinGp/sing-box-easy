<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import MultiSelect from '../../volt/MultiSelect.vue'
import { matchValues } from '../../lab/trafficPolicyConfig'

const props = defineProps<{ options: string[]; label: string }>()
const model = defineModel<string>({ required: true })
const { t } = useI18n()
const skipped = ref<string[]>([])
const selected = computed(() => matchValues(model.value))
const choices = computed(() => [...new Set([...props.options, ...selected.value])])

function pasteBatch(event: ClipboardEvent) {
  const text = event.clipboardData?.getData('text/plain') ?? ''
  if (!/[\r\n,]/.test(text)) return
  event.preventDefault()
  const tags = matchValues(text)
  const available = new Set(props.options)
  skipped.value = tags.filter(tag => !available.has(tag))
  model.value = [...new Set([...selected.value, ...tags.filter(tag => available.has(tag))])].join('\n')
  const input = event.target
  if (input instanceof HTMLInputElement && input.getAttribute('role') === 'searchbox' && input.value) {
    input.value = ''
    input.dispatchEvent(new Event('input', { bubbles: true }))
  }
}
</script>

<template>
  <div @paste="pasteBatch">
    <MultiSelect :model-value="selected" :options="choices" :aria-label="label"
      filter display="chip" :filter-placeholder="t('managed.ruleSetSearch')"
      :filter-paste="pasteBatch" :show-toggle-all="false" reset-filter-on-hide class="w-full"
      @update:model-value="model = $event.join('\n'); skipped = []" />
    <p v-if="skipped.length" role="status" class="mt-1 break-words text-xs text-muted">
      {{ t('managed.ruleSetSkipped', { tags: skipped.join(', ') }) }}
    </p>
  </div>
</template>
