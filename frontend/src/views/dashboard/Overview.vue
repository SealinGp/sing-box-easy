<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import type { ServiceStatus } from '../../types/api'
import Alert from '../../components/Alert.vue'
import Button from '../../components/Button.vue'
import { serviceControlService } from '../../services'

const status = ref<ServiceStatus | null>(null)
const loading = ref(false)
const actionLoading = ref(false)
const error = ref<string | null>(null)
const successMessage = ref<string | null>(null)

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
  error.value = null
  try {    
    const {data} = await serviceControlService.getServiceStatus()
    status.value = data
  } catch (err) {
    console.error('Failed to get service status:', err)
    error.value = 'Failed to fetch service status'
  } finally {
    loading.value = false
  }
}

const handleStart = async () => {
  actionLoading.value = true
  error.value = null
  successMessage.value = null
  try {
    await serviceControlService.startService()
    successMessage.value = 'Service started successfully'
    await fetchStatus()
  } catch (err: any) {
    console.error('Failed to start service:', err)
    error.value = err.response?.data?.error || 'Failed to start service'
  } finally {
    actionLoading.value = false
  }
}

const handleStop = async () => {
  actionLoading.value = true
  error.value = null
  successMessage.value = null
  try {
    await serviceControlService.stopService()
    successMessage.value = 'Service stopped successfully'
    await fetchStatus()
  } catch (err: any) {
    console.error('Failed to stop service:', err)
    error.value = err.response?.data?.error || 'Failed to stop service'
  } finally {
    actionLoading.value = false
  }
}

const handleRestart = async () => {
  actionLoading.value = true
  error.value = null
  successMessage.value = null
  try {
    await serviceControlService.restartService()
    successMessage.value = 'Service restarted successfully'
    await fetchStatus()
  } catch (err: any) {
    console.error('Failed to restart service:', err)
    error.value = err.response?.data?.error || 'Failed to restart service'
  } finally {
    actionLoading.value = false
  }
}

onMounted(fetchStatus)
</script>

<template>
  <div class="p-8">
    <h2 class="text-3xl font-bold text-gray-900 mb-6">Dashboard Overview</h2>

    <Alert v-if="error" type="error" closable @close="error = null" class="mb-6">
      {{ error }}
    </Alert>
    <Alert v-if="successMessage" type="success" closable @close="successMessage = null" class="mb-6">
      {{ successMessage }}
    </Alert>

    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
      <!-- Service Status Card -->
      <div class="bg-white p-6 rounded-lg shadow">
        <h3 class="text-lg font-semibold text-gray-700 mb-4">Service Status</h3>

        <div v-if="loading" class="flex items-center justify-center py-4">
          <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
        </div>

        <div v-else-if="status" class="space-y-4">
          <div class="flex items-center gap-3">
            <span :class="statusColor" class="text-3xl">{{ statusIcon }}</span>
            <span :class="statusColor" class="text-2xl font-bold capitalize">
              {{ status.status }}
            </span>
          </div>

          <div v-if="status.pid" class="text-sm text-gray-600">
            <p><span class="font-semibold">PID:</span> {{ status.pid }}</p>
          </div>

          <div v-if="status.uptime" class="text-sm text-gray-600">
            <p><span class="font-semibold">Uptime:</span> {{ status.uptime }}</p>
          </div>

          <div class="pt-4 border-t border-gray-200">
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
