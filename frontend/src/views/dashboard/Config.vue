<script setup lang="ts">
import { ref, onMounted, computed, onUnmounted } from 'vue'
import { configService } from '../../services'
import { Code, type SingBoxConfig } from '../../types/api'
import { useToast } from 'primevue'
import MonacoEditor from '../../components/MonacoEditor.vue'

const configContent = ref('')
const originalContent = ref('')
const loading = ref(false)
const saving = ref(false)
const validating = ref(false)
const hasChanges = ref(false)
const isFullscreen = ref(false)
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
})

const loadConfig = async () => {
  loading.value = true
  try {
    const { data } = await configService.getConfig()
    const jsonString = JSON.stringify(data, null, 2)
    configContent.value = jsonString
    originalContent.value = jsonString
    hasChanges.value = false
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.response?.data?.error || 'Failed to load configuration',
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

const handleContentChange = (value: string) => {
  configContent.value = value
  hasChanges.value = value !== originalContent.value
}

const saveConfig = async () => {
  saving.value = true
  try {
    const config = JSON.parse(configContent.value) as SingBoxConfig
    await configService.saveConfig(config)
    originalContent.value = configContent.value
    hasChanges.value = false
    toast.add({
      severity: 'success',
      summary: 'Success',
      detail: 'Configuration saved successfully',
      life: 3000
    })
  } catch (err: any) {
    if (err instanceof SyntaxError) {
      toast.add({
        severity: 'error',
        summary: 'Invalid JSON',
        detail: 'Please check your JSON syntax',
        life: 3000
      })
    } else {
      toast.add({
        severity: 'error',
        summary: 'Error',
        detail: err.response?.data?.error || 'Failed to save configuration',
        life: 3000
      })
    }
  } finally {
    saving.value = false
  }
}

const validateConfig = async () => {
  validating.value = true
  try {
    const config = JSON.parse(configContent.value) as SingBoxConfig
    const resp = await configService.validateConfig(config)
    if(resp.code != Code.Success) {
      toast.add({
        severity: 'error',
        summary: 'Error',
        detail: resp.msg,
      })
      return
    }


    toast.add({
      severity: 'success',
      summary: 'Valid Configuration',
      detail: 'Configuration is valid and can be applied',
      life: 3000
    })
  } catch (err: any) {
    if (err instanceof SyntaxError) {
      toast.add({
        severity: 'error',
        summary: 'Invalid JSON',
        detail: 'Please check your JSON syntax',
        life: 3000
      })
    } else {
      toast.add({
        severity: 'error',
        summary: 'Validation Failed',
        detail: err.response?.data?.error || 'Configuration validation failed',
        life: 3000
      })
    }
  } finally {
    validating.value = false
  }
}

const rollbackConfig = async () => {
  loading.value = true
  try {
    await configService.rollbackConfig()
    await loadConfig()
    toast.add({
      severity: 'success',
      summary: 'Success',
      detail: 'Configuration rolled back successfully',
      life: 3000
    })
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.response?.data?.error || 'Failed to rollback configuration',
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

const toggleFullscreen = () => {
  isFullscreen.value = !isFullscreen.value
}

const toggleTheme = () => {
  editorTheme.value = editorTheme.value === 'vs-dark' ? 'vs-light' : 'vs-dark'
}

const formatDocument = () => {
  try {
    const parsed = JSON.parse(configContent.value)
    configContent.value = JSON.stringify(parsed, null, 2)
  } catch (err) {
    toast.add({
      severity: 'error',
      summary: 'Invalid JSON',
      detail: 'Cannot format invalid JSON',
      life: 3000
    })
  }
}

// Keyboard shortcut
const handleKeyDown = (e: KeyboardEvent) => {
  // Ctrl+S or Cmd+S to save
  if ((e.ctrlKey || e.metaKey) && e.key === 's') {
    e.preventDefault()
    if (hasChanges.value && !saving.value) {
      saveConfig()
    }
  }
}

onMounted(() => {
  document.addEventListener('keydown', handleKeyDown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeyDown)
})
</script>

<template>
  <div class="p-8" :class="{ 'fixed inset-0 z-50 bg-gray-50 dark:bg-gray-900': isFullscreen }">
    <div class="flex flex-col h-full">
      <!-- Header -->
      <div class="flex justify-between items-center mb-6">
        <h2 class="text-3xl font-bold text-gray-900 dark:text-gray-100">Configuration Editor</h2>
        <div class="flex items-center gap-3">
          <!-- Theme Toggle -->
          <button
            @click="toggleTheme"
            class="p-2 text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition-colors"
            title="Toggle theme"
          >
            <svg v-if="editorTheme === 'vs-dark'" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z" />
            </svg>
            <svg v-else class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z" />
            </svg>
          </button>

          <!-- Fullscreen Toggle -->
          <button
            @click="toggleFullscreen"
            class="p-2 text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition-colors"
            :title="isFullscreen ? 'Exit fullscreen' : 'Enter fullscreen'"
          >
            <svg v-if="!isFullscreen" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4" />
            </svg>
            <svg v-else class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 9V4m0 0H4m5 0l-5 5m11-5h5m0 0v5m0-5l-5 5m-5 6v5m0 0h5m-5 0l5-5m5 5v-5m0 5h-5m5 0l-5-5" />
            </svg>
          </button>

          <!-- Actions -->
          <button
            @click="formatDocument"
            :disabled="loading"
            class="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors disabled:opacity-50"
          >
            Format
          </button>

          <button
            @click="validateConfig"
            :disabled="loading || validating || !configContent"
            class="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors disabled:opacity-50"
          >
            <span v-if="validating">Validating...</span>
            <span v-else>Validate</span>
          </button>

          <button
            @click="rollbackConfig"
            :disabled="loading"
            class="px-4 py-2 text-sm font-medium text-amber-700 dark:text-amber-300 bg-amber-50 dark:bg-amber-900/20 border border-amber-300 dark:border-amber-700 rounded-lg hover:bg-amber-100 dark:hover:bg-amber-900/40 transition-colors disabled:opacity-50"
          >
            Rollback
          </button>

          <button
            @click="resetChanges"
            :disabled="loading || !hasChanges"
            class="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors disabled:opacity-50"
          >
            Reset
          </button>

          <button
            @click="saveConfig"
            :disabled="loading || saving || !hasChanges"
            class="px-4 py-2 text-sm font-medium text-white bg-violet-600 rounded-lg hover:bg-violet-700 transition-colors disabled:opacity-50 flex items-center gap-2"
          >
            <span v-if="saving">Saving...</span>
            <span v-else>Save</span>
            <span v-if="!saving && hasChanges" class="text-xs opacity-75">(Ctrl+S)</span>
          </button>
        </div>
      </div>

      <!-- Editor Container -->
      <div class="flex-1 bg-white dark:bg-gray-800 rounded-lg shadow-lg overflow-hidden">
        <div v-if="loading" class="flex items-center justify-center h-full">
          <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-violet-600"></div>
        </div>
        <div v-else class="h-full">
          <MonacoEditor
            v-model="configContent"
            @update:modelValue="handleContentChange"
            language="json"
            :theme="editorTheme"
            :options="editorOptions"
            class="h-full"
          />
        </div>
      </div>

      <!-- Status Bar -->
      <div class="mt-4 px-4 py-2 bg-gray-100 dark:bg-gray-800 rounded-lg flex items-center justify-between text-sm">
        <div class="flex items-center gap-4">
          <span class="text-gray-600 dark:text-gray-400">
            JSON Configuration
          </span>
          <span v-if="hasChanges" class="flex items-center gap-1 text-amber-600 dark:text-amber-400">
            <span class="w-2 h-2 bg-amber-600 dark:bg-amber-400 rounded-full"></span>
            Modified
          </span>
          <span v-else class="flex items-center gap-1 text-green-600 dark:text-green-400">
            <span class="w-2 h-2 bg-green-600 dark:bg-green-400 rounded-full"></span>
            Saved
          </span>
        </div>
        <div class="text-gray-600 dark:text-gray-400">
          Lines: {{ configContent.split('\n').length }}
        </div>
      </div>
    </div>
  </div>
</template>