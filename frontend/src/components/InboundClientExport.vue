<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { DocumentDuplicateIcon } from '@heroicons/vue/24/outline'
import { ContextPanel, Select } from '../volt'
import Button from './Button.vue'
import Input from './Input.vue'
import CopyIcon from './CopyIcon.vue'
import Tooltip from './Tooltip.vue'
import SaveFeedback from './SaveFeedback.vue'
import { apiService } from '../services/api'
import { apiErrorMessage } from '../utils/apiErrorMessage'
import type { BasicResponse, Inbound } from '../types/api'
const props = defineProps<{ inbound: Inbound }>()
const { t, te } = useI18n()
const trigger = ref<HTMLElement>()
const visible = ref(false)
const busy = ref(false)
const error = ref('')
const server = ref('')
const user = ref(0)
let generation = 0
const result = ref<{ node: Record<string, unknown>; warnings: string[] }>()
const users = computed(() => ((props.inbound as unknown as { users?: { name?: string; username?: string }[] }).users ?? []).map((u, index) => ({ value: index, label: u.name || u.username || `${index + 1}` })))
const text = computed(() => result.value ? JSON.stringify(result.value.node, null, 2) : '')
watch([server, user, () => props.inbound], () => { generation++; result.value = undefined; error.value = '' }, { deep: true, flush: 'sync' })
async function generate() {
  if (busy.value) return
  busy.value = true; error.value = ''; result.value = undefined
  const requestGeneration = generation
  try {
    const response = (await apiService.post<BasicResponse<{ node: Record<string, unknown>; warnings: string[] }>>('/nodes/from-inbound', { inbound: props.inbound, server: server.value, user: user.value })).data.data
    if (requestGeneration === generation) result.value = response
  } catch (err) {
    const message = apiErrorMessage(err, t('common.error'))
    if (requestGeneration === generation) error.value = te(message) ? t(message) : message
  } finally { busy.value = false }
}
async function open() {
  if (busy.value) return
  const listen = (props.inbound as unknown as { listen?: string }).listen
  server.value = !listen || ['::', '0.0.0.0', '127.0.0.1', '::1'].includes(listen) ? window.location.hostname.replace(/^\[|\]$/g, '') : listen
  user.value = 0; visible.value = true
  await generate()
}
</script>

<template>
  <button ref="trigger" class="icon-action ui-control" :aria-label="t('inbounds.tooltip.copyConfig')" :disabled="busy" @click.stop="open"><DocumentDuplicateIcon class="h-4 w-4" /></button>
  <Tooltip :target="trigger"><template #content>{{ t('inbounds.tooltip.copyConfig') }}</template></Tooltip>
  <ContextPanel :visible="visible" :header="t('clientExport.title')" class="max-w-xl" :closable="!busy" :dismissable="!busy" :close-on-escape="!busy" @update:visible="value => { if (!busy) visible = value }">
    <fieldset :disabled="busy" :inert="busy" class="min-w-0 space-y-3">
      <label class="block text-sm">{{ t('clientExport.server') }}<Input v-model="server" class="mt-1" /></label>
      <label v-if="users.length > 1" class="block text-sm">{{ t('clientExport.user') }}<Select v-model="user" :options="users" option-label="label" option-value="value" class="mt-1 w-full" /></label>
      <p class="text-xs text-text-secondary">{{ t('clientExport.addressHint') }}</p>
      <template v-if="result">
        <p v-for="warning in result.warnings" :key="warning" class="text-sm text-amber-700 dark:text-amber-300">{{ t(warning) }}</p>
        <pre class="max-h-72 overflow-auto whitespace-pre-wrap break-all rounded-control bg-bg-secondary p-3 text-xs">{{ text }}</pre>
        <p class="text-xs text-text-secondary">{{ t('clientExport.pasteHint') }}</p>
      </template>
      <SaveFeedback :error="error" />
    </fieldset>
    <template #footer>
      <Button v-if="!result" :loading="busy" @click="generate">{{ t('clientExport.generate') }}</Button>
      <div v-else class="flex items-center gap-2 text-sm">{{ t('clientExport.copy') }}<CopyIcon :text="text" :label="t('clientExport.copy')" /></div>
    </template>
  </ContextPanel>
</template>
