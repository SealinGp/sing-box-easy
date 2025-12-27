<script setup lang="ts">
import { ref, onMounted, nextTick, computed } from 'vue'
import { CodeEditor } from 'monaco-editor-vue3'
import { configService } from '../../services'
import type { SingBoxConfig } from '../../types/api'
import { useToast } from 'primevue'

const configContent = ref('')
const originalContent = ref('')
const loading = ref(false)
const saving = ref(false)
const validating = ref(false)
const hasChanges = ref(false)
const isFullscreen = ref(false)
const editorReady = ref(false)
const editorTheme = ref<'vs-dark' | 'vs-light'>('vs-dark')
const toast = useToast()

const editorOptions = computed(() => ({
  automaticLayout: true,
  formatOnType: true,
  formatOnPaste: true,
  minimap: { enabled: true },
  scrollBeyondLastLine: false,
  fontSize: 14,
  tabSize: 2,
  // Folding (collapse/expand) options
  folding: true, // Enable code folding
  showFoldingControls: 'always', // 'always' | 'mouseover' | 'never'
  foldingStrategy: 'auto', // 'auto' | 'indentation'
  foldingHighlight: true, // Highlight folded regions
  // Additional helpful options for JSON editing
  lineNumbers: 'on',
  renderLineHighlight: 'all',
  bracketPairColorization: {
    enabled: true,
  },
  wordWrap: 'on',
  theme: editorTheme.value,
}))

onMounted(async () => {
  await loadConfig()
  // Wait for the next DOM update cycle
  await nextTick()
  editorReady.value = true
})

const loadConfig = async () => {
  loading.value = true
  try {
    const response = await configService.getConfig()
    if (response.data) {
      configContent.value = JSON.stringify(response.data, null, 2)
      originalContent.value = configContent.value
      hasChanges.value = false
    } else {
      toast.add({
        severity: 'error',
        summary: 'Error',
        detail: response.msg || 'Failed to load configuration',
        life: 3000
      })
    }
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.message || 'Failed to load configuration',
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

const handleEditorChange = (value: string) => {
  configContent.value = value
  hasChanges.value = value !== originalContent.value
}

const validateConfig = async () => {
  validating.value = true

  try {
    const config: SingBoxConfig = JSON.parse(configContent.value)
    const response = await configService.validateConfig(config)

    if (response.data?.valid) {
      toast.add({
        severity: 'success',
        summary: 'Success',
        detail: 'Configuration is valid',
        life: 3000
      })
    } else {
      toast.add({
        severity: 'error',
        summary: 'Validation Failed',
        detail: response.data?.error || 'Configuration validation failed',
        life: 3000
      })
    }
  } catch (err: any) {
    if (err instanceof SyntaxError) {
      toast.add({
        severity: 'error',
        summary: 'Invalid JSON',
        detail: err.message,
        life: 3000
      })
    } else {
      toast.add({
        severity: 'error',
        summary: 'Error',
        detail: err.message || 'Validation failed',
        life: 3000
      })
    }
  } finally {
    validating.value = false
  }
}

const saveConfig = async () => {
  saving.value = true

  try {
    const config: SingBoxConfig = JSON.parse(configContent.value)
    const validateResponse = await configService.validateConfig(config)

    if (!validateResponse.data?.valid) {
      toast.add({
        severity: 'error',
        summary: 'Validation Failed',
        detail: validateResponse.data?.error || 'Configuration validation failed',
        life: 3000
      })
      saving.value = false
      return
    }

    // Note: The backend doesn't have a direct update config endpoint
    // The config is updated through specific API endpoints
    // This is a placeholder - you may need to add a PUT /config endpoint in backend
    toast.add({
      severity: 'success',
      summary: 'Success',
      detail: 'Configuration validated successfully. Use specific APIs to update config sections.',
      life: 3000
    })
    originalContent.value = configContent.value
    hasChanges.value = false
  } catch (err: any) {
    if (err instanceof SyntaxError) {
      toast.add({
        severity: 'error',
        summary: 'Invalid JSON',
        detail: err.message,
        life: 3000
      })
    } else {
      toast.add({
        severity: 'error',
        summary: 'Error',
        detail: err.message || 'Failed to save configuration',
        life: 3000
      })
    }
  } finally {
    saving.value = false
  }
}

const rollback = async () => {
  if (!confirm('Are you sure you want to rollback to the backup configuration?')) {
    return
  }

  loading.value = true

  try {
    const response = await configService.rollbackConfig()
    toast.add({
      severity: 'success',
      summary: 'Success',
      detail: response.msg || 'Configuration rolled back successfully',
      life: 3000
    })
    await loadConfig()
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.message || 'Rollback failed',
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

const resetChanges = () => {
  configContent.value = originalContent.value
  hasChanges.value = false
}

const loadBackup = async () => {
  loading.value = true

  try {
    const response = await configService.getBackupConfig()
    if (response.data) {
      configContent.value = JSON.stringify(response.data, null, 2)
      hasChanges.value = configContent.value !== originalContent.value
      toast.add({
        severity: 'success',
        summary: 'Success',
        detail: 'Backup configuration loaded',
        life: 3000
      })
    } else {
      toast.add({
        severity: 'error',
        summary: 'Error',
        detail: response.msg || 'Failed to load backup configuration',
        life: 3000
      })
    }
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.message || 'Failed to load backup configuration',
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

const toggleFullscreen = () => {
  isFullscreen.value = !isFullscreen.value
}

const toggleTheme = () => {
  editorTheme.value = editorTheme.value === 'vs-dark' ? 'vs-light' : 'vs-dark'
}
</script>

<template>
  <div class="p-8">
    <div class="mb-6">
      <h2 class="text-3xl font-bold text-gray-900 dark:text-gray-100">Configuration Editor</h2>
      <p class="mt-2 text-sm text-gray-600 dark:text-gray-400">
        Edit the raw sing-box configuration file. Changes are validated before saving.
      </p>
    </div>

    <div
      :class="[
        'bg-white dark:bg-slate-800 rounded-lg shadow dark:shadow-xl dark:shadow-slate-700/50',
        isFullscreen ? 'fixed inset-0 z-50 rounded-none' : ''
      ]"
    >
      <div class="p-2 border-b border-gray-200 dark:border-gray-700 flex justify-between items-center">
        <div class="flex items-center space-x-2">
          <span v-if="hasChanges" class="text-sm text-orange-600 dark:text-orange-400 font-medium">
            Unsaved changes
          </span>
          <span v-else class="text-sm text-gray-500 dark:text-gray-400">
            No changes
          </span>
        </div>

        <div class="flex space-x-2">
          <button
            @click="loadBackup"
            :disabled="loading"
            class="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-slate-700 border border-gray-300 dark:border-gray-600 rounded-md hover:bg-gray-50 dark:hover:bg-slate-600 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Load Backup
          </button>

          <button
            @click="rollback"
            :disabled="loading"
            class="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-slate-700 border border-gray-300 dark:border-gray-600 rounded-md hover:bg-gray-50 dark:hover:bg-slate-600 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Rollback
          </button>

          <button
            @click="resetChanges"
            :disabled="!hasChanges || loading"
            class="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-slate-700 border border-gray-300 dark:border-gray-600 rounded-md hover:bg-gray-50 dark:hover:bg-slate-600 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Reset
          </button>

          <button
            @click="validateConfig"
            :disabled="validating || loading"
            class="px-4 py-2 text-sm font-medium text-white bg-violet-600 dark:bg-violet-700 rounded-md hover:bg-violet-700 dark:hover:bg-violet-600 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {{ validating ? 'Validating...' : 'Validate' }}
          </button>

          <button
            @click="saveConfig"
            :disabled="!hasChanges || saving || loading"
            class="px-4 py-2 text-sm font-medium text-white bg-green-600 dark:bg-green-700 rounded-md hover:bg-green-700 dark:hover:bg-green-600 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {{ saving ? 'Saving...' : 'Save' }}
          </button>

          <label class="flex items-center cursor-pointer gap-2">
            <svg class="w-4 h-4 text-gray-600 dark:text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z" />
            </svg>
            <input
              type="checkbox"
              class="toggle toggle-sm"
              :checked="editorTheme === 'vs-dark'"
              @change="toggleTheme"
            />
            <svg class="w-4 h-4 text-gray-600 dark:text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z" />
            </svg>
          </label>

          <button
            @click="toggleFullscreen"
            class="p-2 text-gray-700 dark:text-gray-300 bg-white dark:bg-slate-700 border border-gray-300 dark:border-gray-600 rounded-md hover:bg-gray-50 dark:hover:bg-slate-600"
            :title="isFullscreen ? 'Exit fullscreen' : 'Enter fullscreen'"
          >
            <svg v-if="!isFullscreen" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4" />
            </svg>
            <svg v-else class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
      </div>

      <div :class="[
        isFullscreen ? 'h-[calc(100vh-73px)]' : 'h-[calc(100vh-200px)]'
      ]">
        <div v-if="loading" class="h-full flex items-center justify-center">
          <div class="text-center">
            <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-violet-600 mx-auto"></div>
            <p class="mt-4 text-gray-500 dark:text-gray-400">Loading configuration...</p>
          </div>
        </div>

        <CodeEditor
          v-else-if="editorReady"
          v-model:value="configContent"
          language="json"
          :options="editorOptions"
          @change="handleEditorChange"
          class="h-full"
        />
      </div>
    </div>
  </div>
</template>
