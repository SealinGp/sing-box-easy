<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { logService } from '../../services'
import { type LogConfig } from '../../types/api'
import { useNotify } from '../../composables/useNotify'
import { Select } from '../../volt'

const { t } = useI18n()

const logConfig = ref<LogConfig>({
  disabled: false,
  level: 'info',
  output: '',
  timestamp: true,
})
const loading = ref(false)
const saving = ref(false)
const notify = useNotify()

const logLevels = ['trace', 'debug', 'info', 'warn', 'error', 'fatal', 'panic']

const logLevelOptions = logLevels.map((level) => ({ value: level, label: level.toUpperCase() }))

onMounted(async () => {
  await loadLog()
})

const loadLog = async () => {
  loading.value = true
  try {
    const response = await logService.getLog()
    if (response.data) {
      logConfig.value = response.data
    }
  } catch (err) {
    notify.apiError(err, t('log.toast.loadFailed'))
  } finally {
    loading.value = false
  }
}

const saveLog = async () => {
  saving.value = true

  try {
    const response = await logService.updateLog(logConfig.value)
    notify.success(response.data?.message || t('log.toast.savedOk'))
  } catch (err) {
    notify.apiError(err, t('log.toast.saveFailed'))
  } finally {
    saving.value = false
  }
}

const resetToDefaults = () => {
  logConfig.value = {
    disabled: false,
    level: 'info',
    output: '',
    timestamp: true,
  }
}
</script>

<template>
  <div class="page-shell">
    <div class="mb-4">
      <h2 class="text-3xl font-bold text-gray-900 dark:text-gray-100">{{ $t('log.title') }}</h2>
      <p class="mt-2 text-sm text-gray-600 dark:text-gray-400">
        {{ $t('log.subtitle') }}
      </p>
    </div>

    <div v-if="loading" class="bg-white dark:bg-slate-800 p-6 rounded-surface shadow-surface">
      <div class="flex items-center justify-center">
        <div class="text-center">
          <div class="animate-spin rounded-pill h-12 w-12 border-b-2 border-primary-600 mx-auto"></div>
          <p class="mt-4 text-gray-500 dark:text-gray-400">{{ $t('log.loading') }}</p>
        </div>
      </div>
    </div>

    <div v-else class="bg-white dark:bg-slate-800 p-4 rounded-surface shadow-surface">
      <div class="space-y-4">
        <!-- Disabled -->
        <div class="flex items-center justify-between">
          <div>
            <label class="text-sm font-medium text-gray-900 dark:text-gray-100">
              {{ $t('log.disabled.label') }}
            </label>
            <p class="text-sm text-gray-500 dark:text-gray-400">
              {{ $t('log.disabled.help') }}
            </p>
          </div>
          <button
            @click="logConfig.disabled = !logConfig.disabled"
            :class="[
              'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-pill border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-600 focus:ring-offset-2',
              logConfig.disabled ? 'bg-primary-600' : 'bg-gray-200 dark:bg-gray-700'
            ]"
          >
            <span
              :class="[
                'pointer-events-none inline-block h-5 w-5 transform rounded-pill bg-white shadow ring-0 transition duration-200 ease-in-out',
                logConfig.disabled ? 'translate-x-5' : 'translate-x-0'
              ]"
            />
          </button>
        </div>

        <!-- Log Level -->
        <div>
          <label class="block text-sm font-medium text-gray-900 dark:text-gray-100 mb-2">
            {{ $t('log.level.label') }}
          </label>
          <p class="text-sm text-gray-500 dark:text-gray-400 mb-3">
            {{ $t('log.level.help') }}
          </p>
          <Select
            class="w-full"
            v-model="logConfig.level"
            :options="logLevelOptions"
            optionLabel="label"
            optionValue="value"
            filter
            :filterPlaceholder="$t('common.search')"
            :emptyFilterMessage="$t('common.noMatch')"
          />
        </div>

        <!-- Output File -->
        <div>
          <label class="block text-sm font-medium text-gray-900 dark:text-gray-100 mb-2">
            {{ $t('log.output.label') }}
          </label>
          <p class="text-sm text-gray-500 dark:text-gray-400 mb-3">
            {{ $t('log.output.help') }}
          </p>
          <input
            v-model="logConfig.output"
            type="text"
            placeholder="/var/log/sing-box.log"
            class="block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-control shadow-surface focus:outline-none focus:ring-primary-500 focus:border-primary-500 bg-white dark:bg-slate-700 text-gray-900 dark:text-gray-100 placeholder-gray-400 dark:placeholder-gray-500"
          />
        </div>

        <!-- Timestamp -->
        <div class="flex items-center justify-between">
          <div>
            <label class="text-sm font-medium text-gray-900 dark:text-gray-100">
              {{ $t('log.timestamp.label') }}
            </label>
            <p class="text-sm text-gray-500 dark:text-gray-400">
              {{ $t('log.timestamp.help') }}
            </p>
          </div>
          <button
            @click="logConfig.timestamp = !logConfig.timestamp"
            :class="[
              'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-pill border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-600 focus:ring-offset-2',
              logConfig.timestamp ? 'bg-primary-600' : 'bg-gray-200 dark:bg-gray-700'
            ]"
          >
            <span
              :class="[
                'pointer-events-none inline-block h-5 w-5 transform rounded-pill bg-white shadow ring-0 transition duration-200 ease-in-out',
                logConfig.timestamp ? 'translate-x-5' : 'translate-x-0'
              ]"
            />
          </button>
        </div>

        <!-- Action Buttons -->
        <div class="flex justify-between pt-4 border-t border-gray-200 dark:border-gray-700">
          <button
            @click="resetToDefaults"
            class="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-slate-700 border border-gray-300 dark:border-gray-600 rounded-control hover:bg-gray-50 dark:hover:bg-slate-600"
          >
            {{ $t('log.resetDefaults') }}
          </button>

          <button
            @click="saveLog"
            :disabled="saving"
            class="px-6 py-2 text-sm font-medium text-white bg-primary-600 dark:bg-primary-700 rounded-control hover:bg-primary-700 dark:hover:bg-primary-600 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {{ saving ? $t('common.saving') : $t('log.saveConfig') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
