<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { Code, type ServiceStatus } from '../../types/api'
import Button from '../../components/Button.vue'
import { serviceControlService } from '../../services'
import { useToast } from 'primevue/usetoast'

const status = ref<ServiceStatus | null>(null)
const loading = ref(false)
const actionLoading = ref(false)
const toast = useToast()

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

const statusIcon = computed(() => {
  if (!status.value) return '●'
  switch (status.value.status) {
    case 'running':
      return '●'
    case 'stopped':
      return '●'
    default:
      return '●'
  }
})

const fetchStatus = async () => {
  loading.value = true
  try {
    const {data} = await serviceControlService.getServiceStatus()
    status.value = data
  } catch (err: any) {
    console.error('Failed to get service status:', err)
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.response?.data?.error || 'Failed to fetch service status',
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
        summary: 'Error',
        detail: resp.msg,
      })
      return
    }

    toast.add({
      severity: 'success',
      summary: 'Success',
      detail: 'Service started successfully',
      life: 3000
    })
    await fetchStatus()
  } catch (err: any) {
    console.error('Failed to start service:', err)
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.response?.data?.error || 'Failed to start service',
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
      summary: 'Success',
      detail: 'Service stopped successfully',
      life: 3000
    })
    await fetchStatus()
  } catch (err: any) {
    console.error('Failed to stop service:', err)
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.response?.data?.error || 'Failed to stop service',
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
      summary: 'Success',
      detail: 'Service restarted successfully',
      life: 3000
    })
    await fetchStatus()
  } catch (err: any) {
    console.error('Failed to restart service:', err)
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.response?.data?.error || 'Failed to restart service',
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
    <h2 class="text-3xl font-bold text-gray-900 dark:text-gray-100 mb-6">Dashboard Overview</h2>

    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
      <!-- Service Status Card -->
      <div class="bg-white dark:bg-slate-800 p-6 rounded-lg shadow dark:shadow-xl dark:shadow-slate-700/50">
        <h3 class="text-lg font-semibold text-gray-700 dark:text-gray-300 mb-4">Service Status</h3>

        <div v-if="loading" class="flex items-center justify-center py-4">
          <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-violet-600"></div>
        </div>

        <div v-else-if="status" class="space-y-4">
          <div class="flex items-center gap-3">
            <span :class="statusColor" class="text-3xl">{{ statusIcon }}</span>
            <span :class="statusColor" class="text-2xl font-bold capitalize">
              {{ status.status }}
            </span>
          </div>

          <div v-if="status.pid" class="text-sm text-gray-600 dark:text-gray-400">
            <p><span class="font-semibold">PID:</span> {{ status.pid }}</p>
          </div>

          <div v-if="status.uptime" class="text-sm text-gray-600 dark:text-gray-400">
            <p><span class="font-semibold">Uptime:</span> {{ status.uptime }}</p>
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
                Start
              </Button>

              <Button
                @click="handleStop"
                :disabled="status.status === 'stopped' || actionLoading"
                variant="danger"
                size="sm"
                class="text-xs"
              >
                Stop
              </Button>

              <Button
                @click="handleRestart"
                :disabled="status.status === 'stopped' || actionLoading"
                variant="primary"
                size="sm"
                class="text-xs"
              >
                Restart
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
              Refresh Status
            </Button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
