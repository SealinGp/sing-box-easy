import Popover from '../volt/Popover.vue'
import { useWorkspaceActive, useWorkspaceDirty } from './workspaceContext'
import { trafficPolicyService, type ManagedPolicy } from '../services/trafficPolicy'
import { ref, onMounted, computed, onUnmounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { configService, serviceControlService } from '../services'
import { type SingBoxConfig, type ConfigVersion } from '../types/api'
import { useNotify } from '../composables/useNotify'
import { useConfirm } from '../composables/useConfirm'

export function useConfigEditor() {

  const { t, locale } = useI18n()

  const configContent = ref('')
  const originalContent = ref('')
  const loading = ref(false)
  const saving = ref(false)
  const rawImpactPanel = ref<InstanceType<typeof Popover>>()
  const saveAnchor = ref<HTMLElement>()
  const rawReview = ref<{revision:string;policies:ManagedPolicy[];content:string}>()
  async function saveReviewed() {
    if (!rawReview.value || saving.value) return
    if (rawReview.value.content !== configContent.value) {
      rawImpactPanel.value?.hide()
      rawReview.value = undefined
      await saveConfig()
      return
    }
    const submitted = rawReview.value
    saving.value = true
    try {
      await configService.saveConfig(JSON.parse(submitted.content), submitted.revision)
      originalContent.value = submitted.content
      hasChanges.value = configContent.value !== submitted.content
      rawImpactPanel.value?.hide()
      rawReview.value = undefined
      notify.success(t('config.toast.savedOk'))
    } catch (error) {
      notify.apiError(error, t('config.toast.saveFailed'))
      rawReview.value = undefined
      rawImpactPanel.value?.hide()
    } finally {
      saving.value = false
    }
  }
  const validating = ref(false)
  const restarting = ref(false)
  const hasChanges = ref(false)
  useWorkspaceDirty(() => hasChanges.value)
  const workspaceActive = useWorkspaceActive()
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

  watch(workspaceActive, active => {
    if (active && !hasChanges.value && !saving.value) void loadConfig()
  })

  const handleContentChange = (value: string) => {
    configContent.value = value
    hasChanges.value = value !== originalContent.value
  }

  const saveConfig = async () => {
    if (saving.value || loading.value) return
    const submitted = configContent.value
    saving.value = true
    try {
      const config = JSON.parse(submitted) as SingBoxConfig
      const latest = await configService.getConfig()
      if (JSON.stringify(latest.data, null, 2) !== originalContent.value && !await confirm({
        message: t('nav.configChanged'), confirmLabel: t('config.save'), tone: 'danger',
      })) return
      const impact = await trafficPolicyService.impact(config)
      if (impact.policies.length) {
        rawReview.value={...impact,content:submitted}
        if (saveAnchor.value) rawImpactPanel.value?.showAt(saveAnchor.value)
        return
      }
      await configService.saveConfig(config,impact.revision)
      originalContent.value = submitted
      hasChanges.value = configContent.value !== submitted
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
      action: 'delete',
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
      action: 'delete',
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

  const resetChanges = async () => {
    if (saving.value || !(await confirm({ message: t('config.confirm.reset'), confirmLabel: t('config.reset'), tone: 'danger' }))) return
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
    if (saving.value) return
    try {
      const parsed = JSON.parse(configContent.value)
      handleContentChange(JSON.stringify(parsed, null, 2))
    } catch {
      notify.error(t('config.toast.cannotFormat'), t('config.toast.invalidJson'))
    }
  }

  // Keyboard shortcut
  const handleKeyDown = (e: KeyboardEvent) => {
    if (!workspaceActive.value) return
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

  return {
    t, configContent, loading, saving, rawImpactPanel, saveAnchor, rawReview, saveReviewed,
    validating, restarting, hasChanges, isFullscreen, isSplit, editorTheme, editorOptions,
    showVersions, versions, versionsLoading, versionRetentionDays, selectedVersionIds,
    batchDeleting, allVersionsSelected, showDiff, diffVersionId, diffOriginal, diffModified,
    handleContentChange, saveConfig, validateConfig, openVersions, closeVersions,
    toggleSelectAllVersions, toggleSelectVersion, batchDeleteVersions, openDiff, rollbackTo,
    deleteVersion, restartService, formatTime, formatRelative, formatBytes, resetChanges,
    toggleFullscreen, toggleTheme, toggleSplit, formatDocument,
  }
}
