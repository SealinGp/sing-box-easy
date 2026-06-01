<script setup lang="ts">
import { ref, onMounted, computed, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { configService, serviceControlService } from '../../services'
import { type SingBoxConfig, type ConfigVersion } from '../../types/api'
import { useNotify } from '../../composables/useNotify'
import { useConfirm } from '../../composables/useConfirm'
import MonacoEditor from '../../components/MonacoEditor.vue'
import MonacoDiffEditor from '../../components/MonacoDiffEditor.vue'

const { t } = useI18n()

const configContent = ref('')
const originalContent = ref('')
const loading = ref(false)
const saving = ref(false)
const validating = ref(false)
const restarting = ref(false)
const hasChanges = ref(false)
const isFullscreen = ref(false)
const isSplit = ref(false)
const editorTheme = ref<'vs-dark' | 'vs-light'>('vs-dark')
const notify = useNotify()
const { confirm } = useConfirm()

// --- Config version history (Versions modal) ---
const showVersions = ref(false)
const versions = ref<ConfigVersion[]>([])
const versionsLoading = ref(false)
// Diff view (within the versions modal)
const showDiff = ref(false)
const diffVersionId = ref<number | null>(null)
const diffOriginal = ref('') // selected version content
const diffModified = ref('') // current editor content

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
  } catch (err) {
    notify.apiError(err, t('config.toast.loadFailed'))
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
    notify.success(t('config.toast.savedOk'))
  } catch (err) {
    if (err instanceof SyntaxError) {
      notify.error(t('config.toast.checkSyntax'), t('config.toast.invalidJson'))
    } else {
      notify.apiError(err, t('config.toast.saveFailed'))
    }
  } finally {
    saving.value = false
  }
}

const validateConfig = async () => {
  validating.value = true
  try {
    const config = JSON.parse(configContent.value) as SingBoxConfig
    await configService.validateConfig(config)
    notify.success(t('config.toast.validDetail'), t('config.toast.validTitle'))
  } catch (err) {
    if (err instanceof SyntaxError) {
      notify.error(t('config.toast.checkSyntax'), t('config.toast.invalidJson'))
    } else {
      notify.apiError(err, t('config.toast.validationFailedDetail'), t('config.toast.validationFailedTitle'))
    }
  } finally {
    validating.value = false
  }
}

// --- Versions modal ---

const openVersions = async () => {
  showVersions.value = true
  showDiff.value = false
  await loadVersions()
}

const closeVersions = () => {
  showVersions.value = false
  showDiff.value = false
}

const loadVersions = async () => {
  versionsLoading.value = true
  try {
    const { data } = await configService.listVersions()
    versions.value = data.versions || []
  } catch (err) {
    notify.apiError(err, t('config.toast.loadVersionsFailed'))
  } finally {
    versionsLoading.value = false
  }
}

const openDiff = async (v: ConfigVersion) => {
  try {
    const { data } = await configService.getVersion(v.id)
    diffOriginal.value = JSON.stringify(data, null, 2)
    diffModified.value = configContent.value
    diffVersionId.value = v.id
    showDiff.value = true
  } catch (err) {
    notify.apiError(err, t('config.toast.loadVersionFailed'))
  }
}

const rollbackTo = async (v: ConfigVersion) => {
  const ok = await confirm({
    title: t('config.versionsModal.rollback'),
    message: t('config.confirm.rollback', { id: v.id }),
    confirmLabel: t('config.versionsModal.rollback'),
  })
  if (!ok) return
  loading.value = true
  try {
    await configService.rollbackToVersion(v.id)
    await loadConfig()
    closeVersions()
    notify.success(t('config.toast.rolledBack', { id: v.id }))
  } catch (err) {
    notify.apiError(err, t('config.toast.rollbackFailed'))
  } finally {
    loading.value = false
  }
}

const deleteVersion = async (v: ConfigVersion) => {
  const ok = await confirm({
    title: t('config.versionsModal.delete'),
    message: t('config.confirm.delete', { id: v.id }),
    confirmLabel: t('common.delete'),
    tone: 'danger',
  })
  if (!ok) return
  versionsLoading.value = true
  try {
    await configService.deleteVersion(v.id)
    // If the diff view is open on the version we just removed, drop back to the list.
    if (showDiff.value && diffVersionId.value === v.id) {
      showDiff.value = false
      diffVersionId.value = null
    }
    await loadVersions()
    notify.success(t('config.toast.deleted', { id: v.id }))
  } catch (err) {
    notify.apiError(err, t('config.toast.deleteFailed'))
  } finally {
    versionsLoading.value = false
  }
}

const restartService = async () => {
  const ok = await confirm({
    title: t('config.restart'),
    message: t('config.confirm.restart'),
    confirmLabel: t('config.restart'),
  })
  if (!ok) return
  restarting.value = true
  try {
    await serviceControlService.restartService()
    notify.success(t('config.toast.restartedDetail'), t('config.toast.restartedTitle'))
  } catch (err) {
    notify.apiError(err, t('config.toast.restartFailed'))
  } finally {
    restarting.value = false
  }
}

const formatBytes = (n: number) => {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / 1024 / 1024).toFixed(1)} MB`
}

const formatTime = (s: string) => {
  const d = new Date(s)
  return isNaN(d.getTime()) ? s : d.toLocaleString()
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

const toggleSplit = () => {
  isSplit.value = !isSplit.value
}

const formatDocument = () => {
  try {
    const parsed = JSON.parse(configContent.value)
    configContent.value = JSON.stringify(parsed, null, 2)
  } catch {
    notify.error(t('config.toast.cannotFormat'), t('config.toast.invalidJson'))
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
  <!--
    Outer wrapper is pinned to 100vh and laid out as a single flex column so
    the editor child can claim all the leftover viewport height instead of
    waiting for a parent `h-full` chain that may or may not resolve. The
    fullscreen toggle still wins via `fixed inset-0 z-50` when active.
  -->
  <div
    class="p-4 h-screen flex flex-col overflow-hidden"
    :class="{ 'fixed inset-0 z-50 bg-gray-50 dark:bg-gray-900': isFullscreen }"
  >
      <!-- Header -->
      <div class="flex justify-between items-center mb-4 shrink-0">
        <h2 class="text-3xl font-bold text-gray-900 dark:text-gray-100">{{ $t('config.title') }}</h2>
        <div class="flex items-center gap-3">
          <!-- Theme Toggle -->
          <button
            @click="toggleTheme"
            class="p-2 text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition-colors"
            :title="$t('config.toggleTheme')"
          >
            <svg v-if="editorTheme === 'vs-dark'" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z" />
            </svg>
            <svg v-else class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z" />
            </svg>
          </button>

          <!-- Split View Toggle -->
          <button
            @click="toggleSplit"
            class="p-2 rounded-lg transition-colors"
            :class="isSplit
              ? 'text-violet-600 dark:text-violet-400 bg-violet-100 dark:bg-violet-900/30'
              : 'text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800'"
            :title="isSplit ? $t('config.splitOn') : $t('config.splitOff')"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 5a1 1 0 011-1h14a1 1 0 011 1v14a1 1 0 01-1 1H5a1 1 0 01-1-1V5zM12 4v16" />
            </svg>
          </button>

          <!-- Fullscreen Toggle -->
          <button
            @click="toggleFullscreen"
            class="p-2 text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition-colors"
            :title="isFullscreen ? $t('config.exitFullscreen') : $t('config.enterFullscreen')"
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
            {{ $t('config.format') }}
          </button>

          <button
            @click="validateConfig"
            :disabled="loading || validating || !configContent"
            class="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors disabled:opacity-50"
          >
            <span v-if="validating">{{ $t('config.validating') }}</span>
            <span v-else>{{ $t('config.validate') }}</span>
          </button>

          <button
            @click="restartService"
            :disabled="restarting"
            class="px-4 py-2 text-sm font-medium text-emerald-700 dark:text-emerald-300 bg-emerald-50 dark:bg-emerald-900/20 border border-emerald-300 dark:border-emerald-700 rounded-lg hover:bg-emerald-100 dark:hover:bg-emerald-900/40 transition-colors disabled:opacity-50"
            :title="$t('config.restartTitle')"
          >
            <span v-if="restarting">{{ $t('config.restarting') }}</span>
            <span v-else>{{ $t('config.restart') }}</span>
          </button>

          <button
            @click="openVersions"
            :disabled="loading"
            class="px-4 py-2 text-sm font-medium text-amber-700 dark:text-amber-300 bg-amber-50 dark:bg-amber-900/20 border border-amber-300 dark:border-amber-700 rounded-lg hover:bg-amber-100 dark:hover:bg-amber-900/40 transition-colors disabled:opacity-50"
            :title="$t('config.versionsTitle')"
          >
            {{ $t('config.versions') }}
          </button>

          <button
            @click="resetChanges"
            :disabled="loading || !hasChanges"
            class="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors disabled:opacity-50"
          >
            {{ $t('config.reset') }}
          </button>

          <button
            @click="saveConfig"
            :disabled="loading || saving || !hasChanges"
            class="px-4 py-2 text-sm font-medium text-white bg-violet-600 rounded-lg hover:bg-violet-700 transition-colors disabled:opacity-50 flex items-center gap-2"
          >
            <span v-if="saving">{{ $t('config.saving') }}</span>
            <span v-else>{{ $t('config.save') }}</span>
            <span v-if="!saving && hasChanges" class="text-xs opacity-75">(Ctrl+S)</span>
          </button>
        </div>
      </div>

      <!--
        Editor Container.
        `min-h-0` is critical: inside a flex column, `flex-1` children
        default to min-height:auto and refuse to shrink past their
        intrinsic content height, which breaks the "fill remaining
        space" behaviour. Setting min-h-0 lets the editor expand AND
        shrink to fit whatever viewport is left after header + status.
      -->
      <div class="flex-1 min-h-0 bg-white dark:bg-gray-800 rounded-lg shadow-lg overflow-hidden">
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
            :split="isSplit"
            class="h-full"
          />
        </div>
      </div>

      <!-- Status Bar -->
      <div class="mt-2 px-4 py-2 bg-gray-100 dark:bg-gray-800 rounded-lg flex items-center justify-between text-sm shrink-0">
        <div class="flex items-center gap-4">
          <span class="text-gray-600 dark:text-gray-400">
            {{ $t('config.jsonConfig') }}
          </span>
          <span v-if="hasChanges" class="flex items-center gap-1 text-amber-600 dark:text-amber-400">
            <span class="w-2 h-2 bg-amber-600 dark:bg-amber-400 rounded-full"></span>
            {{ $t('config.modified') }}
          </span>
          <span v-else class="flex items-center gap-1 text-green-600 dark:text-green-400">
            <span class="w-2 h-2 bg-green-600 dark:bg-green-400 rounded-full"></span>
            {{ $t('config.saved') }}
          </span>
        </div>
        <div class="text-gray-600 dark:text-gray-400">
          {{ $t('config.lines', { n: configContent.split('\n').length }) }}
        </div>
      </div>

      <!-- Versions Modal -->
      <div
        v-if="showVersions"
        class="fixed inset-0 z-[60] flex items-center justify-center bg-black/50 p-4"
        @click.self="closeVersions"
      >
        <div class="bg-white dark:bg-gray-900 rounded-xl shadow-2xl w-[90vw] h-[80vh] flex flex-col overflow-hidden">
          <!-- Modal header -->
          <div class="flex items-center justify-between px-5 py-3 border-b border-gray-200 dark:border-gray-700 shrink-0">
            <div class="flex items-center gap-3">
              <button
                v-if="showDiff"
                @click="showDiff = false"
                class="text-sm text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200"
              >
                &larr; {{ $t('common.back') }}
              </button>
              <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">
                <span v-if="!showDiff">{{ $t('config.versionsModal.title') }}</span>
                <span v-else>{{ $t('config.versionsModal.diffTitle', { id: diffVersionId }) }}</span>
              </h3>
            </div>
            <button
              @click="closeVersions"
              class="p-1.5 text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg"
              :title="$t('common.close')"
            >
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          <!-- List view -->
          <div v-if="!showDiff" class="flex-1 min-h-0 overflow-y-auto p-5">
            <div v-if="versionsLoading" class="flex items-center justify-center h-32">
              <div class="animate-spin rounded-full h-7 w-7 border-b-2 border-violet-600"></div>
            </div>
            <div v-else-if="versions.length === 0" class="text-center text-gray-500 dark:text-gray-400 py-12">
              {{ $t('config.versionsModal.empty') }}
            </div>
            <table v-else class="w-full text-sm">
              <thead>
                <tr class="text-left text-gray-500 dark:text-gray-400 border-b border-gray-200 dark:border-gray-700">
                  <th class="py-2 pr-4 font-medium">{{ $t('config.versionsModal.colVersion') }}</th>
                  <th class="py-2 pr-4 font-medium">{{ $t('config.versionsModal.colSavedAt') }}</th>
                  <th class="py-2 pr-4 font-medium">{{ $t('config.versionsModal.colSize') }}</th>
                  <th class="py-2 pr-4 font-medium text-right">{{ $t('config.versionsModal.colActions') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="(v, i) in versions"
                  :key="v.id"
                  class="border-b border-gray-100 dark:border-gray-800 text-gray-800 dark:text-gray-200"
                >
                  <td class="py-2.5 pr-4">
                    #{{ v.id }}
                    <span v-if="i === 0" class="ml-2 px-2 py-0.5 text-[10px] rounded-full bg-violet-100 dark:bg-violet-900/40 text-violet-700 dark:text-violet-300">{{ $t('config.versionsModal.latest') }}</span>
                  </td>
                  <td class="py-2.5 pr-4 text-gray-600 dark:text-gray-400">{{ formatTime(v.created_at) }}</td>
                  <td class="py-2.5 pr-4 text-gray-600 dark:text-gray-400">{{ formatBytes(v.size) }}</td>
                  <td class="py-2.5 pr-4">
                    <div class="flex items-center justify-end gap-2">
                      <button
                        @click="openDiff(v)"
                        class="px-3 py-1.5 text-xs font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
                      >
                        {{ $t('config.versionsModal.diff') }}
                      </button>
                      <button
                        @click="rollbackTo(v)"
                        :disabled="loading"
                        class="px-3 py-1.5 text-xs font-medium text-amber-700 dark:text-amber-300 bg-amber-50 dark:bg-amber-900/20 border border-amber-300 dark:border-amber-700 rounded-lg hover:bg-amber-100 dark:hover:bg-amber-900/40 transition-colors disabled:opacity-50"
                      >
                        {{ $t('config.versionsModal.rollback') }}
                      </button>
                      <button
                        @click="deleteVersion(v)"
                        :disabled="loading || versionsLoading"
                        class="px-3 py-1.5 text-xs font-medium text-red-700 dark:text-red-300 bg-red-50 dark:bg-red-900/20 border border-red-300 dark:border-red-700 rounded-lg hover:bg-red-100 dark:hover:bg-red-900/40 transition-colors disabled:opacity-50"
                        :title="$t('config.versionsModal.delete')"
                      >
                        {{ $t('config.versionsModal.delete') }}
                      </button>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <!-- Diff view -->
          <div v-else class="flex-1 min-h-0 flex flex-col">
            <div class="flex-1 min-h-0">
              <MonacoDiffEditor
                :original="diffOriginal"
                :modified="diffModified"
                language="json"
                :theme="editorTheme"
                class="h-full"
              />
            </div>
            <div class="flex justify-end gap-2 px-5 py-3 border-t border-gray-200 dark:border-gray-700 shrink-0">
              <button
                @click="showDiff = false"
                class="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
              >
                {{ $t('config.versionsModal.backToList') }}
              </button>
              <button
                v-if="diffVersionId !== null"
                @click="rollbackTo({ id: diffVersionId, size: 0, created_at: '' })"
                :disabled="loading"
                class="px-4 py-2 text-sm font-medium text-amber-700 dark:text-amber-300 bg-amber-50 dark:bg-amber-900/20 border border-amber-300 dark:border-amber-700 rounded-lg hover:bg-amber-100 dark:hover:bg-amber-900/40 transition-colors disabled:opacity-50"
              >
                {{ $t('config.versionsModal.rollbackToThis') }}
              </button>
              <button
                v-if="diffVersionId !== null"
                @click="deleteVersion({ id: diffVersionId, size: 0, created_at: '' })"
                :disabled="loading || versionsLoading"
                class="px-4 py-2 text-sm font-medium text-red-700 dark:text-red-300 bg-red-50 dark:bg-red-900/20 border border-red-300 dark:border-red-700 rounded-lg hover:bg-red-100 dark:hover:bg-red-900/40 transition-colors disabled:opacity-50"
              >
                {{ $t('config.versionsModal.delete') }}
              </button>
            </div>
          </div>
        </div>
      </div>
  </div>
</template>