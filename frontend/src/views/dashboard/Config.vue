<script setup lang="ts">
import { ref, onMounted, computed, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { configService, serviceControlService } from '../../services'
import { type SingBoxConfig, type ConfigVersion } from '../../types/api'
import { useNotify } from '../../composables/useNotify'
import { useConfirm } from '../../composables/useConfirm'
import MonacoEditor from '../../components/MonacoEditor.vue'
import MonacoDiffEditor from '../../components/MonacoDiffEditor.vue'

const { t, locale } = useI18n()

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
// Retention window for auto-deletion of old versions. Mirrors the backend
// constant configversion.DefaultMaxAge (60 days); the daily cron removes
// anything older.
const versionRetentionDays = 60
// Multi-select for batch delete.
const selectedVersionIds = ref<Set<number>>(new Set())
const batchDeleting = ref(false)
const allVersionsSelected = computed(
  () => versions.value.length > 0 && versions.value.every((v) => selectedVersionIds.value.has(v.id)),
)
// `now` ticks while the versions modal is open so the relative "x ago" labels
// stay fresh without a full reload.
const now = ref(Date.now())
let nowTimer: ReturnType<typeof setInterval> | undefined
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
  selectedVersionIds.value = new Set()
  now.value = Date.now()
  if (!nowTimer) nowTimer = setInterval(() => (now.value = Date.now()), 30000)
  await loadVersions()
}

const closeVersions = () => {
  showVersions.value = false
  showDiff.value = false
  selectedVersionIds.value = new Set()
  if (nowTimer) {
    clearInterval(nowTimer)
    nowTimer = undefined
  }
}

const loadVersions = async () => {
  versionsLoading.value = true
  try {
    const { data } = await configService.listVersions()
    versions.value = data.versions || []
    // Drop any selected ids that no longer exist.
    const present = new Set(versions.value.map((v) => v.id))
    selectedVersionIds.value = new Set([...selectedVersionIds.value].filter((id) => present.has(id)))
  } catch (err) {
    notify.apiError(err, t('config.toast.loadVersionsFailed'))
  } finally {
    versionsLoading.value = false
  }
}

const toggleSelectVersion = (id: number) => {
  const next = new Set(selectedVersionIds.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  selectedVersionIds.value = next
}

const toggleSelectAllVersions = () => {
  if (allVersionsSelected.value) {
    selectedVersionIds.value = new Set()
  } else {
    selectedVersionIds.value = new Set(versions.value.map((v) => v.id))
  }
}

const batchDeleteVersions = async () => {
  const ids = [...selectedVersionIds.value]
  if (ids.length === 0) return
  const ok = await confirm({
    title: t('config.versionsModal.deleteSelected'),
    message: t('config.confirm.deleteBatch', { n: ids.length }),
    confirmLabel: t('common.delete'),
    tone: 'danger',
  })
  if (!ok) return
  batchDeleting.value = true
  versionsLoading.value = true
  try {
    await configService.deleteVersionsBatch(ids)
    // If the diff view is open on a deleted version, drop back to the list.
    if (showDiff.value && diffVersionId.value !== null && ids.includes(diffVersionId.value)) {
      showDiff.value = false
      diffVersionId.value = null
    }
    selectedVersionIds.value = new Set()
    await loadVersions()
    notify.success(t('config.toast.deletedBatch', { n: ids.length }))
  } catch (err) {
    notify.apiError(err, t('config.toast.deleteFailed'))
  } finally {
    batchDeleting.value = false
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

// formatRelative returns a locale-aware "x ago" string relative to `now` (e.g.
// "5 minutes ago" / "5 分钟前"). Returns '' for unparseable timestamps.
const formatRelative = (s: string): string => {
  const ts = new Date(s).getTime()
  if (isNaN(ts)) return ''
  const diffSec = Math.round((ts - now.value) / 1000) // negative = in the past
  const abs = Math.abs(diffSec)
  let value: number
  let unit: Intl.RelativeTimeFormatUnit
  if (abs < 60) {
    value = diffSec
    unit = 'second'
  } else if (abs < 3600) {
    value = Math.round(diffSec / 60)
    unit = 'minute'
  } else if (abs < 86400) {
    value = Math.round(diffSec / 3600)
    unit = 'hour'
  } else if (abs < 2592000) {
    value = Math.round(diffSec / 86400)
    unit = 'day'
  } else if (abs < 31536000) {
    value = Math.round(diffSec / 2592000)
    unit = 'month'
  } else {
    value = Math.round(diffSec / 31536000)
    unit = 'year'
  }
  return new Intl.RelativeTimeFormat(locale.value, { numeric: 'auto' }).format(value, unit)
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
  if (nowTimer) clearInterval(nowTimer)
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
            <!-- Retention tip -->
            <div class="mb-4 flex items-start gap-2 rounded-lg border border-blue-200 dark:border-blue-800 bg-blue-50 dark:bg-blue-900/20 px-3 py-2 text-xs text-blue-700 dark:text-blue-300">
              <svg class="w-4 h-4 mt-0.5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <span>{{ $t('config.versionsModal.retentionTip', { days: versionRetentionDays }) }}</span>
            </div>

            <div v-if="versionsLoading" class="flex items-center justify-center h-32">
              <div class="animate-spin rounded-full h-7 w-7 border-b-2 border-violet-600"></div>
            </div>
            <div v-else-if="versions.length === 0" class="text-center text-gray-500 dark:text-gray-400 py-12">
              {{ $t('config.versionsModal.empty') }}
            </div>
            <div v-else>
              <!-- Batch toolbar -->
              <div class="flex items-center justify-between mb-3">
                <span class="text-xs text-gray-500 dark:text-gray-400">
                  <template v-if="selectedVersionIds.size">{{ $t('config.versionsModal.selected', { n: selectedVersionIds.size }) }}</template>
                </span>
                <button
                  @click="batchDeleteVersions"
                  :disabled="selectedVersionIds.size === 0 || batchDeleting || versionsLoading"
                  class="px-3 py-1.5 text-xs font-medium text-red-700 dark:text-red-300 bg-red-50 dark:bg-red-900/20 border border-red-300 dark:border-red-700 rounded-lg hover:bg-red-100 dark:hover:bg-red-900/40 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  {{ $t('config.versionsModal.deleteSelected') }}
                  <span v-if="selectedVersionIds.size">({{ selectedVersionIds.size }})</span>
                </button>
              </div>
              <table class="w-full text-sm">
              <thead>
                <tr class="text-left text-gray-500 dark:text-gray-400 border-b border-gray-200 dark:border-gray-700">
                  <th class="py-2 pr-3 font-medium w-8">
                    <input
                      type="checkbox"
                      class="cursor-pointer"
                      :checked="allVersionsSelected"
                      @change="toggleSelectAllVersions"
                      :aria-label="$t('config.versionsModal.selectAll')"
                      :title="$t('config.versionsModal.selectAll')"
                    />
                  </th>
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
                  :class="selectedVersionIds.has(v.id) ? 'bg-violet-50/60 dark:bg-violet-900/10' : ''"
                >
                  <td class="py-2.5 pr-3">
                    <input
                      type="checkbox"
                      class="cursor-pointer"
                      :checked="selectedVersionIds.has(v.id)"
                      @change="toggleSelectVersion(v.id)"
                      :aria-label="$t('config.versionsModal.selectOne', { id: v.id })"
                    />
                  </td>
                  <td class="py-2.5 pr-4">
                    #{{ v.id }}
                    <span v-if="i === 0" class="ml-2 px-2 py-0.5 text-[10px] rounded-full bg-violet-100 dark:bg-violet-900/40 text-violet-700 dark:text-violet-300">{{ $t('config.versionsModal.latest') }}</span>
                  </td>
                  <td class="py-2.5 pr-4 text-gray-600 dark:text-gray-400">
                    {{ formatTime(v.created_at) }}
                    <span class="ml-1 text-xs text-gray-400 dark:text-gray-500">({{ formatRelative(v.created_at) }})</span>
                  </td>
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
          </div>

          <!-- Diff view -->
          <div v-else class="flex-1 min-h-0 flex flex-col">
            <!-- Left/right legend so it's clear which side is which (and that
                 rollback restores the left). -->
            <div class="flex items-center justify-between px-5 py-2 text-xs border-b border-gray-200 dark:border-gray-700 shrink-0">
              <span class="flex items-center gap-1.5 font-medium text-amber-700 dark:text-amber-300">
                <span class="w-2 h-2 rounded-full bg-amber-500"></span>
                {{ $t('config.versionsModal.diffLeft', { id: diffVersionId }) }}
              </span>
              <span class="flex items-center gap-1.5 font-medium text-gray-600 dark:text-gray-400">
                {{ $t('config.versionsModal.diffRight') }}
                <span class="w-2 h-2 rounded-full bg-gray-400"></span>
              </span>
            </div>
            <div class="flex-1 min-h-0">
              <MonacoDiffEditor
                :original="diffOriginal"
                :modified="diffModified"
                language="json"
                :theme="editorTheme"
                class="h-full"
              />
            </div>
            <div class="flex items-center justify-between gap-2 px-5 py-3 border-t border-gray-200 dark:border-gray-700 shrink-0">
              <!-- Rollback sits on the LEFT, aligned with the left pane it restores. -->
              <div class="flex items-center gap-3">
                <button
                  v-if="diffVersionId !== null"
                  @click="rollbackTo({ id: diffVersionId, size: 0, created_at: '' })"
                  :disabled="loading"
                  class="px-4 py-2 text-sm font-medium text-amber-700 dark:text-amber-300 bg-amber-50 dark:bg-amber-900/20 border border-amber-300 dark:border-amber-700 rounded-lg hover:bg-amber-100 dark:hover:bg-amber-900/40 transition-colors disabled:opacity-50"
                >
                  {{ $t('config.versionsModal.rollbackToLeft', { id: diffVersionId }) }}
                </button>
                <span class="text-xs text-gray-500 dark:text-gray-400 hidden sm:inline">
                  {{ $t('config.versionsModal.diffRollbackHint', { id: diffVersionId }) }}
                </span>
              </div>
              <button
                @click="showDiff = false"
                class="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
              >
                {{ $t('config.versionsModal.backToList') }}
              </button>
            </div>
          </div>
        </div>
      </div>
  </div>
</template>