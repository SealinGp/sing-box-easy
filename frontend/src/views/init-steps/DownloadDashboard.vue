<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { apiService } from '../../services/api'
import type { DashboardTask } from '../../types/api'
import { Button, Alert, Card, Loading } from '../../components'
import { CheckCircleIcon, XCircleIcon, ArrowLeftIcon } from '@heroicons/vue/24/outline'

const emit = defineEmits<{
  next: []
  prev: []
}>()

const downloading = ref(false)
const checkingConfig = ref(true)
const checkingInstalled = ref(false)
const error = ref('')

const clashAPIConfigured = ref(false)
const externalUIPath = ref('')
const alreadyInstalled = ref(false)
const downloadTask = ref<DashboardTask | null>(null)

let pollInterval: ReturnType<typeof setInterval> | null = null

onMounted(async () => {
  await checkClashAPIConfig()
})

const checkClashAPIConfig = async () => {
  checkingConfig.value = true
  error.value = ''

  try {
    // 检查 Clash API 配置
    const clashAPI = await apiService.getClashAPI()

    if (clashAPI?.external_ui) {
      clashAPIConfigured.value = true
      externalUIPath.value = clashAPI.external_ui

      // 检查是否已安装
      await checkDashboardInstalled()
    } else {
      clashAPIConfigured.value = false
    }
  } catch (err: any) {
    console.log('No Clash API config found')
    clashAPIConfigured.value = false
  } finally {
    checkingConfig.value = false
  }
}

const checkDashboardInstalled = async () => {
  checkingInstalled.value = true

  try {
    const status = await apiService.getDashboardStatus()
    alreadyInstalled.value = status.installed || false
  } catch (err: any) {
    console.log('Failed to check dashboard status:', err)
    alreadyInstalled.value = false
  } finally {
    checkingInstalled.value = false
  }
}

const startDownload = async () => {
  error.value = ''
  downloading.value = true

  try {
    const response = await apiService.downloadDashboard(externalUIPath.value)

    // Initialize task with pending status
    downloadTask.value = {
      id: response.task_id,
      status: 'running',
      message: response.message,
    }

    // 开始轮询任务状态
    pollTaskStatus(response.task_id)
  } catch (err: any) {
    error.value = err.response?.data?.error || err.message || 'Failed to start download'
    downloading.value = false
  }
}

const pollTaskStatus = (taskId: string) => {
  pollInterval = setInterval(async () => {
    try {
      const task = await apiService.getDashboardTask(taskId)
      downloadTask.value = task

      if (task.status === 'completed') {
        stopPolling()
        downloading.value = false
        alreadyInstalled.value = true
      } else if (task.status === 'failed') {
        stopPolling()
        downloading.value = false
        error.value = task.error || 'Download failed'
      }
    } catch (err: any) {
      console.error('Failed to poll task status:', err)
      stopPolling()
      downloading.value = false
      error.value = 'Failed to check download status'
    }
  }, 2000) // 每2秒轮询一次
}

const stopPolling = () => {
  if (pollInterval) {
    clearInterval(pollInterval)
    pollInterval = null
  }
}

const handleNext = () => {
  stopPolling()
  emit('next')
}

const handleSkip = () => {
  stopPolling()
  emit('next')
}

const handlePrev = () => {
  stopPolling()
  emit('prev')
}

onUnmounted(() => {
  stopPolling()
})
</script>

<template>
  <div class="space-y-6">
    <!-- 检查配置状态 -->
    <div v-if="checkingConfig" class="flex justify-center py-8">
      <Loading size="lg" text="Checking Clash API configuration..." />
    </div>

    <!-- 未配置 External UI -->
    <div v-else-if="!clashAPIConfigured" class="space-y-6">
      <Alert type="warning" title="External UI Not Configured">
        You haven't configured an External UI path in the Clash API settings. Please go back to the previous step and configure it.
      </Alert>

      <Card>
        <div class="text-center py-8">
          <p class="text-gray-600 mb-6">
            The dashboard requires an External UI path to be configured in Clash API settings.
          </p>
          <div class="flex justify-center gap-3">
            <Button variant="secondary" @click="handlePrev">
              <ArrowLeftIcon class="h-4 w-4 mr-2 inline-block" />
              Go Back to Configure
            </Button>
            <Button variant="ghost" @click="handleSkip">
              Skip Dashboard Download
            </Button>
          </div>
        </div>
      </Card>
    </div>

    <!-- 已配置 External UI -->
    <div v-else class="space-y-6">
      <!-- 已安装提示 -->
      <Alert v-if="alreadyInstalled && !downloading" type="success" title="Dashboard Already Installed">
        Dashboard is already installed at: <code class="font-mono">{{ externalUIPath }}</code>
      </Alert>

      <!-- 错误提示 -->
      <Alert v-if="error" type="error" closable @close="error = ''">
        {{ error }}
      </Alert>

      <!-- 下载卡片 -->
      <Card>
        <div class="space-y-4">
          <div>
            <h3 class="text-lg font-semibold text-gray-900 mb-2">Download Dashboard</h3>
            <p class="text-sm text-gray-600">
              Download and install the zashboard web UI for managing sing-box through your browser.
            </p>
          </div>

          <!-- 配置信息 -->
          <div class="bg-gray-50 rounded-lg p-4">
            <div class="text-sm">
              <div class="flex justify-between mb-2">
                <span class="text-gray-600">Dashboard:</span>
                <span class="font-medium text-gray-900">zashboard</span>
              </div>
              <div class="flex justify-between">
                <span class="text-gray-600">Install Path:</span>
                <code class="font-mono text-sm text-gray-900">{{ externalUIPath }}</code>
              </div>
            </div>
          </div>

          <!-- 下载进度 -->
          <Card v-if="downloading && downloadTask" padding="sm" class="bg-blue-50 border-blue-200">
            <div class="space-y-3">
              <div class="flex items-center space-x-2">
                <Loading size="sm" />
                <p class="text-sm font-semibold text-blue-900">Downloading Dashboard...</p>
              </div>
              <div class="bg-gray-900 text-green-400 p-3 rounded font-mono text-xs">
                <pre class="whitespace-pre-wrap break-words">{{ downloadTask.message || 'Preparing download...' }}</pre>
              </div>
              <p class="text-xs text-blue-600">This may take a few minutes. Please wait...</p>
            </div>
          </Card>

          <!-- 下载成功 -->
          <Card
            v-if="!downloading && downloadTask?.status === 'completed'"
            padding="sm"
            class="bg-green-50 border-green-200"
          >
            <div class="flex items-start space-x-3">
              <CheckCircleIcon class="h-5 w-5 text-green-600 flex-shrink-0 mt-0.5" />
              <div class="flex-1">
                <p class="text-sm font-medium text-green-900">Download completed successfully!</p>
                <p class="text-xs text-green-600 mt-1">{{ downloadTask.message }}</p>
              </div>
            </div>
          </Card>

          <!-- 下载失败 -->
          <Card
            v-if="!downloading && downloadTask?.status === 'failed'"
            padding="sm"
            class="bg-red-50 border-red-200"
          >
            <div class="flex items-start space-x-3">
              <XCircleIcon class="h-5 w-5 text-red-600 flex-shrink-0 mt-0.5" />
              <div class="flex-1">
                <p class="text-sm font-medium text-red-900">Download failed</p>
                <p class="text-xs text-red-600 mt-1">{{ downloadTask.error || downloadTask.message }}</p>
              </div>
            </div>
          </Card>

          <!-- 操作按钮 -->
          <div class="flex gap-3 pt-4 border-t">
            <Button
              v-if="!alreadyInstalled"
              variant="primary"
              :loading="downloading"
              :disabled="downloading"
              @click="startDownload"
            >
              {{ downloading ? 'Downloading...' : 'Download Dashboard' }}
            </Button>

            <Button
              v-if="alreadyInstalled"
              variant="secondary"
              :disabled="downloading"
              @click="startDownload"
            >
              Re-download
            </Button>

            <Button
              v-if="alreadyInstalled || downloadTask?.status === 'completed'"
              variant="primary"
              @click="handleNext"
            >
              Continue to Next Step
            </Button>

            <Button
              v-if="!downloading"
              variant="ghost"
              @click="handleSkip"
            >
              Skip this step
            </Button>
          </div>
        </div>
      </Card>

      <!-- 说明信息 -->
      <Card padding="sm" class="bg-gray-50">
        <div class="text-sm text-gray-600 space-y-2">
          <p class="font-medium text-gray-900">About zashboard:</p>
          <ul class="list-disc list-inside space-y-1 ml-2">
            <li>Modern web-based dashboard for sing-box</li>
            <li>Supports Clash API for real-time control</li>
            <li>View connections, switch proxies, update rules</li>
            <li>Access via http://127.0.0.1:9090/ui after installation</li>
          </ul>
        </div>
      </Card>
    </div>
  </div>
</template>
