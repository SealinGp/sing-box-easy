<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Code, type ServiceStatus } from '../../types/api'
import Button from '../../components/Button.vue'
import { serviceControlService } from '../../services'
import { useToast } from 'primevue/usetoast'

const status = ref<ServiceStatus | null>(null)
const loading = ref(false)
const actionLoading = ref(false)
const toast = useToast()
const { t } = useI18n()

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
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: err.message || t('overview.toast.fetchFailed'),
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

const handleStart = async () => {
  actionLoading.value = true
  try {
    const resp = await serviceControlService.startService()
    if(resp.code != Code.Success) {
      toast.add({
        severity: 'error',
        summary: t('common.error'),
        detail: resp.msg,
      })
      return
    }

    toast.add({
      severity: 'success',
      summary: t('common.success'),
      detail: t('overview.toast.startedOk'),
      life: 3000
    })
    await fetchStatus()
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: err.message || t('overview.toast.startFailed'),
      life: 3000
    })
  } finally {
    actionLoading.value = false
  }
}

const handleStop = async () => {
  actionLoading.value = true
  try {
    await serviceControlService.stopService()
    toast.add({
      severity: 'success',
      summary: t('common.success'),
      detail: t('overview.toast.stoppedOk'),
      life: 3000
    })
    await fetchStatus()
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: err.message || t('overview.toast.stopFailed'),
      life: 3000
    })
  } finally {
    actionLoading.value = false
  }
}

const handleRestart = async () => {
  actionLoading.value = true
  try {
    await serviceControlService.restartService()
    toast.add({
      severity: 'success',
      summary: t('common.success'),
      detail: t('overview.toast.restartedOk'),
      life: 3000
    })
    await fetchStatus()
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: err.message || t('overview.toast.restartFailed'),
      life: 3000
    })
  } finally {
    actionLoading.value = false
  }
}

onMounted(fetchStatus)
</script>

<template>
  <div class="p-8">
    <h2 class="text-3xl font-bold text-gray-900 dark:text-gray-100 mb-6">{{ $t('overview.title') }}</h2>

    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
      <!-- Service Status Card -->
      <div class="bg-white dark:bg-slate-800 p-6 rounded-lg shadow dark:shadow-xl dark:shadow-slate-700/50">
        <h3 class="text-lg font-semibold text-gray-700 dark:text-gray-300 mb-4">{{ $t('overview.serviceStatus') }}</h3>

        <div v-if="loading" class="flex items-center justify-center py-4">
          <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-violet-600"></div>
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

          <div class="pt-4 border-t border-gray-200 dark:border-gray-700">
            <div class="grid grid-cols-3 gap-2">
              <Button
                @click="handleStart"
                :disabled="status.status === 'running' || actionLoading"
                variant="success"
                size="sm"
                class="text-xs"
              >
                {{ $t('common.start') }}
              </Button>

              <Button
                @click="handleStop"
                :disabled="status.status === 'stopped' || actionLoading"
                variant="danger"
                size="sm"
                class="text-xs"
              >
                {{ $t('common.stop') }}
              </Button>

              <Button
                @click="handleRestart"
                :disabled="status.status === 'stopped' || actionLoading"
                variant="primary"
                size="sm"
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
              class="w-full text-xs"
            >
              {{ $t('overview.refreshStatus') }}
            </Button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
