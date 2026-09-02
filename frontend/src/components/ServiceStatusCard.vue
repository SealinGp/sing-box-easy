<script setup lang="ts">
/**
 * sing-box's process state, and the controls for it, for the Overview page.
 *
 * Lifted out of Overview.vue, which owned this card's fetch, its three
 * lifecycle handlers and its formatting while the four cards beside it were
 * self-contained components. The view is now just the grid — every tile fetches
 * what it renders, so adding the next one costs a line rather than another
 * block of state at the top of the page.
 *
 * The links at the bottom are deliberately the three places you go NEXT when
 * this card says something is wrong: the log settings, the live log, and the
 * raw config the whole panel is an editor for.
 */
import { computed, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import type { ServiceStatus } from '../types/api'
import Button from './Button.vue'
import { serviceControlService } from '../services'
import { useNotify } from '../composables/useNotify'
import { formatRelativeTime } from '../utils/relativeTime'
import { CodeBracketIcon, CommandLineIcon, DocumentTextIcon } from '@heroicons/vue/24/outline'

const status = ref<ServiceStatus | null>(null)
const loading = ref(false)
const actionLoading = ref(false)
const notify = useNotify()
const { t, locale } = useI18n()

// "14 minutes ago", derived from the process start time the backend reports.
const lastStartedRelative = computed(() => {
  const ts = status.value?.started_at
  if (!ts) return ''
  return formatRelativeTime(ts * 1000, locale.value)
})

// Absolute timestamp for the tooltip on the relative label.
const lastStartedAbsolute = computed(() => {
  const ts = status.value?.started_at
  if (!ts) return ''
  return new Date(ts * 1000).toLocaleString(locale.value)
})

const statusColor = computed(() => {
  if (!status.value) return 'text-gray-500'
  switch (status.value.status) {
    case 'running':
      return 'text-green-600'
    case 'stopped':
      return 'text-red-600'
    default:
      return 'text-yellow-600'
  }
})

// Translate the known backend status words; fall back to the raw value for
// anything we don't have a key for.
const statusLabel = computed(() => {
  const s = status.value?.status
  if (s === 'running') return t('overview.status.running')
  if (s === 'stopped') return t('overview.status.stopped')
  if (s === 'unknown') return t('overview.status.unknown')
  return s ?? ''
})

// Status icon was a dead switch returning the same bullet for every case;
// keep one constant. If we want distinct glyphs per state later, restore
// per-case logic here.
const statusIcon = '●'

const fetchStatus = async () => {
  loading.value = true
  try {
    const { data } = await serviceControlService.getServiceStatus()
    status.value = data
  } catch (err) {
    notify.apiError(err, t('overview.toast.fetchFailed'))
  } finally {
    loading.value = false
  }
}

/**
 * The three lifecycle actions differ only in which call they make and what they
 * say, so they share one runner: each refetches on success, and each reports
 * through `notify` rather than leaving a failed start looking like a no-op.
 */
async function runAction(
  action: () => Promise<unknown>,
  successKey: string,
  failureKey: string,
) {
  actionLoading.value = true
  try {
    await action()
    notify.success(t(successKey))
    await fetchStatus()
  } catch (err) {
    notify.apiError(err, t(failureKey))
  } finally {
    actionLoading.value = false
  }
}

const handleStart = () =>
  runAction(
    () => serviceControlService.startService(),
    'overview.toast.startedOk',
    'overview.toast.startFailed',
  )

const handleStop = () =>
  runAction(
    () => serviceControlService.stopService(),
    'overview.toast.stoppedOk',
    'overview.toast.stopFailed',
  )

const handleRestart = () =>
  runAction(
    () => serviceControlService.restartService(),
    'overview.toast.restartedOk',
    'overview.toast.restartFailed',
  )

onMounted(fetchStatus)
</script>

<template>
  <div class="bg-white dark:bg-slate-800 p-4 rounded-surface shadow-surface">
    <h3 class="text-lg font-semibold text-gray-700 dark:text-gray-300 mb-2">
      {{ $t('overview.serviceStatus') }}
    </h3>

    <div v-if="loading" class="flex items-center justify-center py-3">
      <div class="animate-spin rounded-pill h-6 w-6 border-b-2 border-primary-600"></div>
    </div>

    <div v-else-if="status" class="space-y-2">
      <div class="flex items-center gap-2">
        <span :class="statusColor" class="text-base leading-none">{{ statusIcon }}</span>
        <span :class="statusColor" class="text-lg font-bold capitalize">
          {{ statusLabel }}
        </span>
      </div>

      <div v-if="status.pid" class="text-xs text-gray-600 dark:text-gray-400">
        <p><span class="font-semibold">{{ $t('overview.pid') }}:</span> {{ status.pid }}</p>
      </div>

      <div v-if="status.uptime" class="text-xs text-gray-600 dark:text-gray-400">
        <p><span class="font-semibold">{{ $t('overview.uptime') }}:</span> {{ status.uptime }}</p>
      </div>

      <div v-if="lastStartedRelative" class="text-xs text-gray-600 dark:text-gray-400">
        <p>
          <span class="font-semibold">{{ $t('overview.lastStarted') }}:</span>
          <span :title="lastStartedAbsolute"> {{ lastStartedRelative }}</span>
        </p>
      </div>

      <div v-if="status.version" class="text-xs text-gray-600 dark:text-gray-400">
        <p><span class="font-semibold">{{ $t('overview.version') }}:</span> {{ status.version }}</p>
      </div>

      <div class="pt-2 border-t border-gray-200 dark:border-gray-700">
        <div class="grid grid-cols-3 gap-2">
          <Button
            @click="handleStart"
            :disabled="status.status === 'running' || actionLoading"
            variant="success"
            size="sm"
            action
            class="text-xs"
          >
            {{ $t('common.start') }}
          </Button>

          <Button
            @click="handleStop"
            :disabled="status.status === 'stopped' || actionLoading"
            variant="danger"
            size="sm"
            action
            class="text-xs"
          >
            {{ $t('common.stop') }}
          </Button>

          <Button
            @click="handleRestart"
            :disabled="status.status === 'stopped' || actionLoading"
            variant="primary"
            size="sm"
            action
            class="text-xs"
          >
            {{ $t('common.restart') }}
          </Button>
        </div>
      </div>

      <div>
        <Button
          @click="fetchStatus"
          :disabled="loading || actionLoading"
          variant="secondary"
          size="sm"
          action
          class="w-full text-xs"
        >
          {{ $t('overview.refreshStatus') }}
        </Button>
      </div>

      <!-- Wraps rather than scrolls: three links do not fit one column at the
           narrow end of the grid. -->
      <div class="flex flex-wrap items-center gap-x-3 gap-y-1.5">
        <RouterLink
          :to="{ name: 'DashboardLog' }"
          class="inline-flex items-center gap-1.5 text-xs font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300 transition-colors"
        >
          <DocumentTextIcon class="h-4 w-4" />
          {{ $t('overview.logSettings') }}
        </RouterLink>
        <RouterLink
          :to="{ name: 'DashboardLogs' }"
          class="inline-flex items-center gap-1.5 text-xs font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300 transition-colors"
        >
          <CommandLineIcon class="h-4 w-4" />
          {{ $t('overview.realtimeLog') }}
        </RouterLink>
        <!--
          The raw config.json, in the Monaco editor. Reached from here because a
          service that will not start is usually answered by reading the file
          the form wrote — and because some edits (a key no form covers, a
          hand-merged block) are only possible there.
        -->
        <RouterLink
          :to="{ name: 'DashboardConfig' }"
          class="inline-flex items-center gap-1.5 text-xs font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300 transition-colors"
        >
          <CodeBracketIcon class="h-4 w-4" />
          {{ $t('overview.rawConfig') }}
        </RouterLink>
      </div>
    </div>
  </div>
</template>
