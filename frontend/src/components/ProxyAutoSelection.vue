<script setup lang="ts">
import { computed, ref, useId, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowPathIcon } from '@heroicons/vue/24/outline'
import type { AutoSelectionStatus, RuntimeGroup } from '../types/runtime'
import { runtimeService } from '../services'
import { apiErrorMessage } from '../utils/apiErrorMessage'
import Popover from '../volt/Popover.vue'
import Input from './Input.vue'
import Button from './Button.vue'
import FieldHelp from './FieldHelp.vue'
const props = defineProps<{ group: RuntimeGroup }>()
const emit = defineEmits<{ updated: [] }>()
const { t, locale } = useI18n()
const panel = ref<InstanceType<typeof Popover>>()
const panelId = useId()
const saved = ref<AutoSelectionStatus | null>(null)
const status = computed(() => saved.value ?? props.group.auto_selection)
const enabled = ref(false)
const minutes = ref(5)
const saving = ref(false)
const error = ref('')
const success = ref(false)
const open = ref(false)
const invalid = computed(() => enabled.value && (!Number.isInteger(minutes.value) || minutes.value < 1 || minutes.value > 1440))
watch(() => props.group.auto_selection, () => { saved.value = null })
function show() {
  open.value = true
  enabled.value = status.value?.enabled ?? false
  minutes.value = (status.value?.interval_seconds || 300) / 60
  error.value = ''; success.value = false
}
async function save() {
  if (invalid.value || saving.value) return
  saving.value = true; error.value = ''; success.value = false
  try {
    const { data } = await runtimeService.setAutoSelection(props.group.name, enabled.value, enabled.value ? Math.round(minutes.value * 60) : (status.value?.interval_seconds || 300))
    saved.value = data; success.value = true; emit('updated')
  } catch (err) { error.value = apiErrorMessage(err, t('proxies.auto.saveFailed')) }
  finally { saving.value = false }
}
const stamp = (value?: string) => value && !value.startsWith('0001-') ? new Date(value).toLocaleString(locale.value) : '—'
</script>
<template>
  <span class="inline-flex" @click.stop @keydown.enter.stop @keydown.space.stop>
    <button type="button" class="inline-flex min-h-8 items-center gap-1 rounded-control border border-border px-2 text-xs font-medium focus-visible:outline-2 focus-visible:outline-primary-500"
      :class="status?.enabled ? 'text-primary-600 dark:text-primary-400' : 'text-text-secondary'" :aria-expanded="open"
      :aria-controls="panelId" :aria-label="t('proxies.auto.open', { group: group.name })" @click="panel?.toggle($event)">
      <ArrowPathIcon class="h-3.5 w-3.5" aria-hidden="true" />{{ t(status?.enabled ? 'proxies.auto.active' : 'proxies.auto.manual') }}
    </button>
    <Popover ref="panel" :id="panelId" panel-class="w-[23rem]" :aria-label="t('proxies.auto.title')" @show="show" @hide="open = false">
      <form class="space-y-3 text-sm" :aria-busy="saving" @submit.prevent="save" @input="success = false">
        <div class="flex items-center gap-1">
          <h3 class="font-semibold text-text-primary">{{ t('proxies.auto.title') }}</h3>
          <FieldHelp :label="t('proxies.auto.title')" :text="t('proxies.auto.help')" />
        </div>
        <p class="break-all text-xs text-text-secondary">{{ group.name }}</p>
        <label class="flex min-h-10 cursor-pointer items-center gap-2 text-text-primary">
          <input v-model="enabled" type="checkbox" :disabled="saving" class="h-4 w-4 accent-primary-600" />{{ t('proxies.auto.enable') }}
        </label>
        <Input v-if="enabled" v-model="minutes" type="number" :min="1" :max="1440" required :disabled="saving"
          :label="t('proxies.auto.interval')" :error="invalid ? t('proxies.auto.intervalError') : ''" />
        <p v-if="enabled" class="text-xs text-text-secondary">{{ t('proxies.auto.manualOverride') }}</p>
        <p v-if="status?.enabled" class="text-xs text-text-secondary" role="status">{{ t(`proxies.auto.reasons.${status.reason}`) }}</p>
        <dl v-if="status?.enabled" class="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs text-text-secondary">
          <dt>{{ t('proxies.auto.checked') }}</dt><dd>{{ stamp(status.last_checked) }}</dd>
          <dt>{{ t('proxies.auto.next') }}</dt><dd>{{ stamp(status.next_check) }}</dd>
          <dt>{{ t('proxies.auto.switched') }}</dt><dd>{{ stamp(status.last_switched) }}</dd>
        </dl>
        <p v-if="status?.error" class="break-words text-xs text-red-600">{{ status.error }}</p>
        <RouterLink to="/dashboard/settings#node-quality" class="inline-block text-xs text-primary-600 underline" @click="panel?.hide()">{{ t('proxies.auto.standards') }}</RouterLink>
        <p v-if="error" role="alert" class="text-xs text-red-600">{{ error }}</p>
        <p v-if="success" role="status" class="text-xs text-green-600">{{ t('proxies.auto.saved') }}</p>
        <div class="flex gap-2">
          <Button type="submit" size="sm" :loading="saving" :disabled="saving || invalid">{{ t('common.save') }}</Button>
          <Button type="button" size="sm" variant="ghost" :disabled="saving" @click="panel?.hide()">{{ t('common.close') }}</Button>
        </div>
      </form>
    </Popover>
  </span>
</template>
