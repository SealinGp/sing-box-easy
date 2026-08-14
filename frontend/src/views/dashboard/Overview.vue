<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { type ServiceStatus } from '../../types/api'
import Button from '../../components/Button.vue'
import { serviceControlService } from '../../services'
import { useNotify } from '../../composables/useNotify'
import { formatRelativeTime } from '../../utils/relativeTime'
import SubscriptionsOverviewCard from '../../components/SubscriptionsOverviewCard.vue'
import DnsProbeCard from '../../components/DnsProbeCard.vue'
import { DocumentTextIcon, CommandLineIcon } from '@heroicons/vue/24/outline'

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
    const {data} = await serviceControlService.getServiceStatus()
    status.value = data
  } catch (err) {
    notify.apiError(err, t('overview.toast.fetchFailed'))
  } finally {
    loading.value = false
  }
}

const handleStart = async () => {
  actionLoading.value = true
  try {
    await serviceControlService.startService()
    notify.success(t('overview.toast.startedOk'))
    await fetchStatus()
  } catch (err) {
    notify.apiError(err, t('overview.toast.startFailed'))
  } finally {
    actionLoading.value = false
  }
}

const handleStop = async () => {
  actionLoading.value = true
  try {
    await serviceControlService.stopService()
    notify.success(t('overview.toast.stoppedOk'))
    await fetchStatus()
  } catch (err) {
    notify.apiError(err, t('overview.toast.stopFailed'))
  } finally {
    actionLoading.value = false
  }
}

const handleRestart = async () => {
  actionLoading.value = true
  try {
    await serviceControlService.restartService()
    notify.success(t('overview.toast.restartedOk'))
    await fetchStatus()
  } catch (err) {
    notify.apiError(err, t('overview.toast.restartFailed'))
  } finally {
    actionLoading.value = false
  }
}

onMounted(fetchStatus)
</script>

<template>
  <div class="p-8">
    <h2 class="text-3xl font-bold text-gray-900 dark:text-gray-100 mb-6">{{ $t('overview.title') }}</h2>

    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 items-start">
      <!-- Service Status Card -->
      <div class="bg-white dark:bg-slate-800 p-6 rounded-surface shadow dark:shadow-float dark:shadow-slate-700/50">
        <h3 class="text-lg font-semibold text-gray-700 dark:text-gray-300 mb-4">{{ $t('overview.serviceStatus') }}</h3>

        <div v-if="loading" class="flex items-center justify-center py-4">
          <div class="animate-spin rounded-pill h-8 w-8 border-b-2 border-primary-600"></div>
        </div>

        <div v-else-if="status" class="space-y-4">
          <div class="flex items-center gap-3">
            <span :class="statusColor" class="text-3xl">{{ statusIcon }}</span>
            <span :class="statusColor" class="text-2xl font-bold capitalize">
              {{ statusLabel }}
            </span>
          </div>

          <div v-if="status.pid" class="text-sm text-gray-600 dark:text-gray-400">
            <p><span class="font-semibold">{{ $t('overview.pid') }}:</span> {{ status.pid }}</p>
          </div>

          <div v-if="status.uptime" class="text-sm text-gray-600 dark:text-gray-400">
            <p><span class="font-semibold">{{ $t('overview.uptime') }}:</span> {{ status.uptime }}</p>
          </div>

          <div v-if="lastStartedRelative" class="text-sm text-gray-600 dark:text-gray-400">
            <p>
              <span class="font-semibold">{{ $t('overview.lastStarted') }}:</span>
              <span :title="lastStartedAbsolute"> {{ lastStartedRelative }}</span>
            </p>
          </div>

          <div v-if="status.version" class="text-sm text-gray-600 dark:text-gray-400">
            <p><span class="font-semibold">{{ $t('overview.version') }}:</span> {{ status.version }}</p>
          </div>

          <div class="pt-4 border-t border-gray-200 dark:border-gray-700">
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

          <div class="pt-2">
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

          <div class="pt-1 flex items-center gap-4">
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
          </div>
        </div>
      </div>

      <!-- Subscriptions: quota, expiry and freshness at a glance -->
      <SubscriptionsOverviewCard />

      <!-- "Where does this domain actually go?" without leaving the dashboard -->
      <DnsProbeCard />
    </div>
  </div>
</template>
