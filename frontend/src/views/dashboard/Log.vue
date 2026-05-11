<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { logService } from '../../services'
import { Code, type LogConfig } from '../../types/api'
import { useToast } from 'primevue'

const logConfig = ref<LogConfig>({
  disabled: false,
  level: 'info',
  output: '',
  timestamp: true,
})
const loading = ref(false)
const saving = ref(false)
const toast = useToast()

const logLevels = ['trace', 'debug', 'info', 'warn', 'error', 'fatal', 'panic']

onMounted(async () => {
  await loadLog()
})

const loadLog = async () => {
  loading.value = true
  try {
    const response = await logService.getLog()
    if (response.code === Code.Success && response.data) {
      logConfig.value = response.data
    } else {
      toast.add({
        severity: 'error',
        summary: 'Error',
        detail: response.msg || 'Failed to load log configuration',
        life: 3000
      })
    }
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.message || 'Failed to load log configuration',
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

const saveLog = async () => {
  saving.value = true

  try {
    const response = await logService.updateLog(logConfig.value)
    if (response.code === Code.Success) {
      toast.add({
        severity: 'success',
        summary: 'Success',
        detail: response.data?.message || 'Log configuration saved successfully',
        life: 3000
      })
    } else {
      toast.add({
        severity: 'error',
        summary: 'Error',
        detail: response.msg || 'Failed to save log configuration',
        life: 3000
      })
    }
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.message || 'Failed to save log configuration',
      life: 3000
    })
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
  <div class="p-8">
    <div class="mb-6">
      <h2 class="text-3xl font-bold text-gray-900 dark:text-gray-100">Log Configuration</h2>
      <p class="mt-2 text-sm text-gray-600 dark:text-gray-400">
        Configure sing-box logging settings
      </p>
    </div>

    <div v-if="loading" class="bg-white dark:bg-slate-800 p-8 rounded-lg shadow dark:shadow-xl dark:shadow-slate-700/50">
      <div class="flex items-center justify-center">
        <div class="text-center">
          <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-violet-600 mx-auto"></div>
          <p class="mt-4 text-gray-500 dark:text-gray-400">Loading log configuration...</p>
        </div>
      </div>
    </div>

    <div v-else class="bg-white dark:bg-slate-800 p-6 rounded-lg shadow dark:shadow-xl dark:shadow-slate-700/50">
      <div class="space-y-6">
        <!-- Disabled -->
        <div class="flex items-center justify-between">
          <div>
            <label class="text-sm font-medium text-gray-900 dark:text-gray-100">
              Disable Logging
            </label>
            <p class="text-sm text-gray-500 dark:text-gray-400">
              Turn off all logging output
            </p>
          </div>
          <button
            @click="logConfig.disabled = !logConfig.disabled"
            :class="[
              'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-violet-600 focus:ring-offset-2',
              logConfig.disabled ? 'bg-violet-600' : 'bg-gray-200 dark:bg-gray-700'
            ]"
          >
            <span
              :class="[
                'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                logConfig.disabled ? 'translate-x-5' : 'translate-x-0'
              ]"
            />
          </button>
        </div>

        <!-- Log Level -->
        <div>
          <label class="block text-sm font-medium text-gray-900 dark:text-gray-100 mb-2">
            Log Level
          </label>
          <p class="text-sm text-gray-500 dark:text-gray-400 mb-3">
            Set the minimum log level to display
          </p>
          <select
            v-model="logConfig.level"
            class="block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm focus:outline-none focus:ring-violet-500 focus:border-violet-500 bg-white dark:bg-slate-700 text-gray-900 dark:text-gray-100"
          >
            <option v-for="level in logLevels" :key="level" :value="level">
              {{ level.toUpperCase() }}
            </option>
          </select>
        </div>

        <!-- Output File -->
        <div>
          <label class="block text-sm font-medium text-gray-900 dark:text-gray-100 mb-2">
            Output File
          </label>
          <p class="text-sm text-gray-500 dark:text-gray-400 mb-3">
            Path to log file (leave empty for stdout)
          </p>
          <input
            v-model="logConfig.output"
            type="text"
            placeholder="/var/log/sing-box.log"
            class="block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm focus:outline-none focus:ring-violet-500 focus:border-violet-500 bg-white dark:bg-slate-700 text-gray-900 dark:text-gray-100 placeholder-gray-400 dark:placeholder-gray-500"
          />
        </div>

        <!-- Timestamp -->
        <div class="flex items-center justify-between">
          <div>
            <label class="text-sm font-medium text-gray-900 dark:text-gray-100">
              Include Timestamp
            </label>
            <p class="text-sm text-gray-500 dark:text-gray-400">
              Add timestamps to log entries
            </p>
          </div>
          <button
            @click="logConfig.timestamp = !logConfig.timestamp"
            :class="[
              'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-violet-600 focus:ring-offset-2',
              logConfig.timestamp ? 'bg-violet-600' : 'bg-gray-200 dark:bg-gray-700'
            ]"
          >
            <span
              :class="[
                'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                logConfig.timestamp ? 'translate-x-5' : 'translate-x-0'
              ]"
            />
          </button>
        </div>

        <!-- Action Buttons -->
        <div class="flex justify-between pt-4 border-t border-gray-200 dark:border-gray-700">
          <button
            @click="resetToDefaults"
            class="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-slate-700 border border-gray-300 dark:border-gray-600 rounded-md hover:bg-gray-50 dark:hover:bg-slate-600"
          >
            Reset to Defaults
          </button>

          <button
            @click="saveLog"
            :disabled="saving"
            class="px-6 py-2 text-sm font-medium text-white bg-violet-600 dark:bg-violet-700 rounded-md hover:bg-violet-700 dark:hover:bg-violet-600 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {{ saving ? 'Saving...' : 'Save Configuration' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
