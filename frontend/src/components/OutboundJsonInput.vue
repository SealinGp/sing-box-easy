<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Textarea from './Textarea.vue'
import { parseOutboundJson } from '../utils/outboundJson'
const emit = defineEmits<{ parsed: [node: Record<string, unknown>]; invalid: [invalid: boolean] }>()
const { t } = useI18n()
const input = ref('')
const error = ref('')
const parsed = ref(false)
watch(input, value => {
  error.value = ''; parsed.value = false
  if (!value.trim()) { emit('invalid', false); return }
  try {
    const node = parseOutboundJson(value)
    emit('parsed', node); emit('invalid', false); parsed.value = true
  } catch (err) {
    const key = err instanceof Error ? err.message : ''
    error.value = t(key.startsWith('outboundJson.') ? key : 'outboundJson.syntax')
    emit('invalid', true)
  }
})
</script>

<template>
  <details class="rounded-control border border-border p-3">
    <summary class="cursor-pointer text-sm font-medium">{{ t('outboundJson.title') }}</summary>
    <label class="mt-3 block text-sm">
      {{ t('outboundJson.label') }}
      <Textarea v-model="input" :rows="5" full-width class="mt-1 font-mono" :placeholder="t('outboundJson.placeholder')" :aria-invalid="!!error" />
    </label>
    <p v-if="error" role="alert" class="mt-2 text-sm text-red-600 dark:text-red-400">{{ error }}</p>
    <p v-else-if="parsed" role="status" class="mt-2 text-sm text-text-secondary">{{ t('outboundJson.ready') }}</p>
  </details>
</template>
