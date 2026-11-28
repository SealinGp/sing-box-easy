<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { logService } from '../../services'
import { type LogConfig } from '../../types/api'
import { useInlineSave } from '../../composables/useInlineSave'
import { Select } from '../../volt'
import Button from '../../components/Button.vue'
import Input from '../../components/Input.vue'
import FieldHelp from '../../components/FieldHelp.vue'
import Reveal from '../../components/Reveal.vue'
import SaveFeedback from '../../components/SaveFeedback.vue'

const { t } = useI18n()
const defaults = (): LogConfig => ({ disabled: false, level: 'info', output: '', timestamp: true })
const logConfig = ref<LogConfig>(defaults())
const loading = ref(false)
const loadError = ref('')
const { saving, saveError, savedMessage, persist, clearSaveFeedback } = useInlineSave()
const logLevelOptions = ['trace', 'debug', 'info', 'warn', 'error', 'fatal', 'panic']
  .map(level => ({ value: level, label: level.toUpperCase() }))
watch(logConfig, clearSaveFeedback, { deep: true })

async function loadLog() {
  loading.value = true
  loadError.value = ''
  try {
    const response = await logService.getLog()
    logConfig.value = response.data || defaults()
  } catch (error) {
    loadError.value = error instanceof Error ? error.message : t('log.toast.loadFailed')
  } finally { loading.value = false }
}
async function saveLog() {
  if (loading.value || loadError.value) return
  await persist(() => logService.updateLog({ ...logConfig.value }), t('log.toast.savedOk'), t('log.toast.saveFailed'))
}
onMounted(loadLog)
</script>

<template>
  <div class="page-shell">
    <div class="mb-4">
      <h2 class="text-3xl font-bold text-text-primary">{{ $t('log.title') }}</h2>
      <p class="mt-2 text-sm text-text-secondary">{{ $t('log.subtitle') }}</p>
    </div>
    <div class="bg-bg-secondary p-4 rounded-surface shadow-surface">
      <p v-if="loading" role="status" class="py-8 text-center text-text-secondary">{{ $t('log.loading') }}</p>
      <div v-else-if="loadError" class="space-y-3">
        <p role="alert" class="text-sm text-danger">{{ loadError }}</p>
        <Button variant="secondary" @click="loadLog">{{ $t('common.refresh') }}</Button>
      </div>
      <fieldset v-else :disabled="saving" class="min-w-0 space-y-4">
        <div class="flex items-center gap-2">
          <label class="flex items-center gap-2 text-sm font-medium">
            <input v-model="logConfig.disabled" type="checkbox" class="h-4 w-4 accent-primary" />
            {{ $t('log.disabled.label') }}
          </label>
          <FieldHelp :label="$t('log.disabled.label')" :text="$t('log.disabled.help')" />
        </div>
        <Reveal :open="!logConfig.disabled"><div class="space-y-4 pt-1">
          <div>
            <div class="mb-1 flex items-center gap-1">
              <label for="log-level" class="text-sm font-medium">{{ $t('log.level.label') }}</label>
              <FieldHelp :label="$t('log.level.label')" :text="$t('log.level.help')" />
            </div>
            <Select input-id="log-level" :aria-label="$t('log.level.label')" :disabled="saving" class="w-full" v-model="logConfig.level"
              :options="logLevelOptions" optionLabel="label" optionValue="value" />
          </div>
          <Input v-model="logConfig.output" :label="$t('log.output.label')" :hint="$t('log.output.help')" placeholder="/var/log/sing-box.log" />
          <div class="flex items-center gap-2">
            <label class="flex items-center gap-2 text-sm font-medium">
              <input v-model="logConfig.timestamp" type="checkbox" class="h-4 w-4 accent-primary" />
              {{ $t('log.timestamp.label') }}
            </label>
            <FieldHelp :label="$t('log.timestamp.label')" :text="$t('log.timestamp.help')" />
          </div>
        </div></Reveal>
        <div class="flex flex-wrap items-center gap-3 border-t border-border pt-4">
          <Button variant="secondary" @click="logConfig = defaults()">{{ $t('log.resetDefaults') }}</Button>
          <SaveFeedback :error="saveError" :message="savedMessage" />
          <Button :loading="saving" @click="saveLog">{{ $t('log.saveConfig') }}</Button>
        </div>
      </fieldset>
    </div>
  </div>
</template>
