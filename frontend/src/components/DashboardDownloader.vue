<script setup lang="ts">
/**
 * Downloads the selected dashboard into sing-box's `external_ui` directory.
 *
 * Download or upload an archive beside the dashboard URL picker, then poll
 * its installation task until it completes.
 *
 * The target directory and URL are passed in from the live form values rather
 * than read from the saved config — the user may have just changed them, and
 * downloading what they see is the least surprising behaviour.
 */
import { computed, onMounted, onUnmounted, ref, useId } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  ArrowDownTrayIcon,
  CheckCircleIcon,
  ExclamationTriangleIcon,
} from '@heroicons/vue/24/outline'
import type { DashboardTask } from '../types/api'
import { dashboardService } from '../services'
import Button from './Button.vue'
import Input from './Input.vue'
import { ContextPanel } from '../volt'

interface Props {
  /** sing-box `external_ui` path; blank lets the backend fall back. */
  targetDir?: string
  /** Archive URL to fetch. Required — the button stays disabled without it. */
  downloadUrl?: string
  disabled?: boolean
}

/**
 * Emitted once the files are on disk. The backend writes `external_ui` into
 * config.json itself, so the form would otherwise keep showing the stale (often
 * empty) path until a manual reload.
 */
const emit = defineEmits<{ installed: [path: string] }>()

const props = withDefaults(defineProps<Props>(), {
  targetDir: '',
  downloadUrl: '',
  disabled: false,
})

const { t } = useI18n()

/** How often to poll the download task. Shared by download and upload. */
const POLL_INTERVAL_MS = 2000

const downloading = ref(false)
const task = ref<DashboardTask | null>(null)
const errorMessage = ref('')
const installedPath = ref('')
const showProxy = ref(false)
const proxy = ref('')
const uploadOpen = ref(false)
const uploadFile = ref<File | null>(null)
const folderName = ref('')
const uploadId = useId()

let pollTimer: ReturnType<typeof setInterval> | null = null

const stopPolling = () => {
  if (pollTimer !== null) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

// A dangling interval would keep polling a dead task after navigation.
onUnmounted(stopPolling)

const canDownload = computed(
  () => !props.disabled && !downloading.value && Boolean(props.downloadUrl?.trim()),
)

const succeeded = computed(() => task.value?.status === 'completed')

const startDownload = async () => {
  if (!canDownload.value) return

  stopPolling()
  errorMessage.value = ''
  task.value = null
  downloading.value = true

  try {
    const { data } = await dashboardService.downloadDashboard(
      props.targetDir?.trim() || undefined,
      props.downloadUrl?.trim(),
      proxy.value.trim() || undefined,
    )

    task.value = {
      id: data.task_id,
      status: 'running',
      message: data.message,
    }
    pollTask(data.task_id)
  } catch (err) {
    downloading.value = false
    errorMessage.value =
      err instanceof Error && err.message
        ? err.message
        : t('experimental.clash.dashboard.downloadFailed')
  }
}

const startUpload = async () => {
  if (!uploadFile.value || downloading.value || props.disabled) return
  stopPolling()
  downloading.value = true
  errorMessage.value = ''
  task.value = null
  try {
    const { data } = await dashboardService.uploadDashboard(uploadFile.value, props.targetDir?.trim() || undefined, folderName.value.trim() || undefined)
    task.value = { id: data.task_id, status: 'running', message: data.message }
    pollTask(data.task_id)
  } catch (err) {
    downloading.value = false
    errorMessage.value = err instanceof Error ? err.message : t('workspace.archive.failed')
  }
}

const pollTask = (taskId: string) => {
  pollTimer = setInterval(async () => {
    try {
      const { data } = await dashboardService.getDashboardTask(taskId)
      task.value = data

      if (data.status === 'completed') {
        stopPolling()
        downloading.value = false
        await refreshStatus(true)
        return
      }

      if (data.status === 'failed') {
        stopPolling()
        downloading.value = false
        errorMessage.value = data.error || t('experimental.clash.dashboard.downloadFailed')
      }
    } catch (err) {
      stopPolling()
      downloading.value = false
      errorMessage.value =
        err instanceof Error && err.message
          ? err.message
          : t('experimental.clash.dashboard.statusFailed')
    }
  }, POLL_INTERVAL_MS)
}

/**
 * Refresh the "installed at ..." hint. Failure here is not worth surfacing.
 *
 * `announce` is only set after a download the user just triggered — on mount it
 * would rewrite the form field from the server on every page visit.
 */
const refreshStatus = async (announce = false) => {
  try {
    const { data } = await dashboardService.getDashboardStatus()
    installedPath.value = data.installed ? data.path : ''
    if (announce && installedPath.value) emit('installed', installedPath.value)
  } catch {
    installedPath.value = ''
  }
}

// This component only mounts once the parent's config has loaded, so mounting
// is the right moment to prime the "installed at ..." hint.
onMounted(() => refreshStatus())
</script>

<template>
  <div class="space-y-2">
    <div class="flex flex-wrap items-center gap-2">
      <!-- `action` drops the drop shadow: this sits inline among the form's
           flat inputs, and a raised pill reads as floating above the panel. -->
      <Button
        variant="secondary"
        size="sm"
        action
        :disabled="!canDownload"
        :loading="downloading"
        @click="startDownload"
      >
        <ArrowDownTrayIcon v-if="!downloading" class="mr-1.5 h-4 w-4" />
        <span>
          {{
            downloading
              ? $t('experimental.clash.dashboard.downloading')
              : $t('experimental.clash.dashboard.download')
          }}
        </span>
      </Button>

      <Button variant="secondary" size="sm" action :disabled="disabled || downloading" @click="uploadOpen = true">{{ t('workspace.archive.button') }}</Button>
      <button
        type="button"
        class="cursor-pointer text-xs text-gray-500 underline-offset-2 hover:underline dark:text-gray-400"
        @click="showProxy = !showProxy"
      >
        {{ $t('experimental.clash.dashboard.proxyToggle') }}
      </button>

      <span
        v-if="installedPath"
        class="ml-auto flex items-center gap-1 text-xs text-emerald-600 dark:text-emerald-400"
      >
        <CheckCircleIcon class="h-4 w-4 flex-shrink-0" />
        <span class="truncate font-mono">{{ installedPath }}</span>
      </span>
    </div>

    <!-- Optional proxy: GitHub is frequently unreachable from the hosts this
         runs on, which is the usual cause of a failed download. -->
    <Input
      v-if="showProxy"
      v-model="proxy"
      :placeholder="$t('experimental.clash.dashboard.proxyPlaceholder')"
    />

    <!-- Live progress -->
    <p
      v-if="downloading && task?.message"
      class="text-xs text-gray-500 dark:text-gray-400"
    >
      {{ task.message }}
    </p>

    <!-- Success -->
    <p
      v-else-if="succeeded"
      class="flex items-start gap-1.5 text-xs text-emerald-600 dark:text-emerald-400"
    >
      <CheckCircleIcon class="mt-px h-4 w-4 flex-shrink-0" />
      <span>{{ task?.message || $t('experimental.clash.dashboard.downloadOk') }}</span>
    </p>

    <!-- Failure -->
    <p
      v-else-if="errorMessage"
      class="flex items-start gap-1.5 text-xs text-red-600 dark:text-red-400"
    >
      <ExclamationTriangleIcon class="mt-px h-4 w-4 flex-shrink-0" />
      <span class="break-words">{{ errorMessage }}</span>
    </p>

    <!-- Nudge: the download uses the on-screen values, which may not be saved -->
    <p v-else class="text-xs text-gray-400 dark:text-gray-500">
      {{ $t('experimental.clash.dashboard.downloadHint') }}
    </p>
    <ContextPanel v-model:visible="uploadOpen" :header="t('workspace.archive.title')" class="w-full max-w-lg">
      <form class="space-y-4" @submit.prevent="startUpload">
        <p class="text-sm text-text-secondary">{{ t('workspace.archive.description') }}</p>
        <div>
          <label :for="uploadId" class="mb-2 block text-sm font-medium text-text-primary">{{ t('workspace.archive.file') }}</label>
          <input :id="uploadId" type="file" accept=".zip,.tar,.tar.gz,.tgz" :disabled="downloading" required class="w-full rounded-control border border-border p-2 text-sm text-text-primary" @change="uploadFile = ($event.target as HTMLInputElement).files?.[0] ?? null" />
        </div>
        <Input v-model="folderName" :label="t('workspace.archive.folder')" :disabled="downloading" />
        <p class="break-all text-xs text-text-secondary">{{ t('workspace.archive.target') }}: {{ targetDir || '—' }}</p>
        <div v-if="downloading || task || errorMessage" class="min-h-20 rounded-control border border-border bg-bg-tertiary p-3 text-sm text-text-secondary" role="status" :aria-busy="downloading">
          {{ errorMessage || task?.error || task?.message || t('common.loading') }}
        </div>
        <div class="flex justify-end gap-2">
          <Button variant="secondary" @click="uploadOpen = false">{{ t('common.close') }}</Button>
          <Button type="submit" :disabled="!uploadFile || downloading" :loading="downloading">{{ t('workspace.archive.submit') }}</Button>
        </div>
      </form>
    </ContextPanel>
  </div>
</template>
