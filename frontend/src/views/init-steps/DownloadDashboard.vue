<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import type { DashboardTask } from '../../types/api'
import { Button, Alert, Card, Loading, Input } from '../../components'
import { CheckCircleIcon, XCircleIcon, ArrowLeftIcon } from '@heroicons/vue/24/outline'
import { dashboardService, experimentalService } from '../../services'

const emit = defineEmits<{
  next: []
  prev: []
}>()

const downloading = ref(false)
const uploading = ref(false)
const checkingConfig = ref(true)
const checkingInstalled = ref(false)
const error = ref('')

const clashAPIConfigured = ref(false)
const externalUIPath = ref('')
const downloadURL = ref('')
const proxy = ref('')
const uploadFile = ref<File | null>(null)
const folderName = ref('')
const alreadyInstalled = ref(false)
const downloadTask = ref<DashboardTask | null>(null)
const uploadTask = ref<DashboardTask | null>(null)

// Tab state: 'download' or 'upload'
const activeTab = ref<'download' | 'upload'>('download')

let pollInterval: ReturnType<typeof setInterval> | null = null

onMounted(async () => {
  await checkClashAPIConfig()
})

const checkClashAPIConfig = async () => {
  checkingConfig.value = true
  error.value = ''

  try {
    // 检查 Clash API 配
    const {data} = await experimentalService.getClashAPI()
    const clashAPI = data

    if (clashAPI?.external_ui) {
      clashAPIConfigured.value = true
      externalUIPath.value = clashAPI.external_ui
      // Load download URL from config if available
      downloadURL.value = clashAPI.external_ui_download_url || ''

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
    const {data} = await dashboardService.getDashboardStatus()
    const status = data
    alreadyInstalled.value = status.installed || false
  } catch (err: any) {
    console.log('Failed to check dashboard status:', err)
    alreadyInstalled.value = false
  } finally {
    checkingInstalled.value = false
  }
}

const handleFileSelect = (event: Event) => {
  const target = event.target as HTMLInputElement
  if (target.files && target.files.length > 0) {
    const file = target.files[0]
    if(file instanceof File) {
      uploadFile.value = file
    }
    
    error.value = ''
  }
}

const startDownload = async () => {
  error.value = ''
  downloading.value = true
  uploadTask.value = null

  try {
    const {data} = await dashboardService.downloadDashboard(
      externalUIPath.value,
      downloadURL.value || undefined,
      proxy.value || undefined
    )

    // Initialize task with pending status
    downloadTask.value = {
      id: data.task_id,
      status: 'running',
      message: data.message,
    }

    // 开始轮询任务状态
    pollTaskStatus(data.task_id, 'download')
  } catch (err: any) {
    error.value = err.response?.data?.error || err.message || 'Failed to start download'
    downloading.value = false
  }
}

const startUpload = async () => {
  if (!uploadFile.value) {
    error.value = 'Please select a file to upload'
    return
  }

  error.value = ''
  uploading.value = true
  downloadTask.value = null

  try {
    const {data} = await dashboardService.uploadDashboard(
      uploadFile.value,
      externalUIPath.value,
      folderName.value || undefined
    )

    // Initialize task with pending status
    uploadTask.value = {
      id: data.task_id,
      status: 'running',
      message: data.message,
    }

    // 开始轮询任务状态
    pollTaskStatus(data.task_id, 'upload')
  } catch (err: any) {
    error.value = err.response?.data?.error || err.message || 'Failed to start upload'
    uploading.value = false
  }
}

const pollTaskStatus = (taskId: string, type: 'download' | 'upload') => {
  pollInterval = setInterval(async () => {
    try {
      const {data} = await dashboardService.getDashboardTask(taskId)
      const task = data

      // Update the appropriate task
      if (type === 'download') {
        downloadTask.value = task
      } else {
        uploadTask.value = task
      }

      if (task.status === 'completed') {
        stopPolling()
        downloading.value = false
        uploading.value = false
        alreadyInstalled.value = true
      } else if (task.status === 'failed') {
        stopPolling()
        downloading.value = false
        uploading.value = false
        error.value = task.error || `${type === 'download' ? 'Download' : 'Upload'} failed`
      }
    } catch (err: any) {
      console.error('Failed to poll task status:', err)
      stopPolling()
      downloading.value = false
      uploading.value = false
      error.value = `Failed to check ${type} status`
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
                <span class="text-gray-600">Install Path:</span>
                <code class="font-mono text-sm text-gray-900">{{ externalUIPath }}</code>
              </div>
            </div>
          </div>

          <!-- 方式选择 -->
          <div class="border-b border-gray-200 mb-4">
            <nav class="-mb-px flex space-x-8" aria-label="Tabs">
              <button
                @click="activeTab = 'download'"
                :class="[
                  activeTab === 'download'
                    ? 'border-blue-500 text-blue-600'
                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300',
                  'whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm'
                ]"
              >
                Download from URL
              </button>
              <button
                @click="activeTab = 'upload'"
                :class="[
                  activeTab === 'upload'
                    ? 'border-blue-500 text-blue-600'
                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300',
                  'whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm'
                ]"
              >
                Upload ZIP File
              </button>
            </nav>
          </div>

          <!-- 下载配置 -->
          <div v-if="activeTab === 'download'" class="space-y-4">
            <div>
              <Input
                v-model="downloadURL"
                label="Download URL (Optional)"
                placeholder="Leave empty to use configured URL from Clash API"
                :disabled="downloading || uploading"
              />
              <p class="mt-1 text-xs text-gray-500">
                Override the download URL if needed. Defaults to URL configured in Clash API settings.
              </p>
            </div>

            <div>
              <Input
                v-model="proxy"
                label="Proxy (Optional)"
                placeholder="e.g., http://127.0.0.1:7890 or socks5://127.0.0.1:1080"
                :disabled="downloading || uploading"
              />
              <p class="mt-1 text-xs text-gray-500">
                Use a proxy server if GitHub is not accessible in your region. Supports HTTP and SOCKS5 proxies.
              </p>
            </div>
          </div>

          <!-- 上传配置 -->
          <div v-if="activeTab === 'upload'" class="space-y-4">
            <div>
              <label class="block text-sm font-medium text-gray-700 mb-2">
                Dashboard ZIP File
              </label>
              <input
                type="file"
                accept=".zip"
                @change="handleFileSelect"
                :disabled="downloading || uploading"
                class="block w-full text-sm text-gray-900 border border-gray-300 rounded-lg cursor-pointer bg-gray-50 focus:outline-none"
              />
              <p class="mt-1 text-xs text-gray-500">
                Select a dashboard ZIP file (e.g., zashboard-gh-pages.zip, yacd-gh-pages.zip)
              </p>
              <div v-if="uploadFile" class="mt-2 p-2 bg-blue-50 border border-blue-200 rounded">
                <p class="text-sm text-blue-900">
                  Selected: <span class="font-medium">{{ uploadFile.name }}</span> ({{ (uploadFile.size / 1024 / 1024).toFixed(2) }} MB)
                </p>
              </div>
            </div>

            <div>
              <Input
                v-model="folderName"
                label="Folder Name (Optional)"
                placeholder="Leave empty to use folder name from ZIP"
                :disabled="downloading || uploading"
              />
              <p class="mt-1 text-xs text-gray-500">
                Specify a custom folder name if the ZIP contains a different folder structure
              </p>
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

          <!-- 上传进度 -->
          <Card v-if="uploading && uploadTask" padding="sm" class="bg-blue-50 border-blue-200">
            <div class="space-y-3">
              <div class="flex items-center space-x-2">
                <Loading size="sm" />
                <p class="text-sm font-semibold text-blue-900">Uploading Dashboard...</p>
              </div>
              <div class="bg-gray-900 text-green-400 p-3 rounded font-mono text-xs">
                <pre class="whitespace-pre-wrap break-words">{{ uploadTask.message || 'Preparing upload...' }}</pre>
              </div>
              <p class="text-xs text-blue-600">Extracting and installing dashboard...</p>
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

          <!-- 上传成功 -->
          <Card
            v-if="!uploading && uploadTask?.status === 'completed'"
            padding="sm"
            class="bg-green-50 border-green-200"
          >
            <div class="flex items-start space-x-3">
              <CheckCircleIcon class="h-5 w-5 text-green-600 flex-shrink-0 mt-0.5" />
              <div class="flex-1">
                <p class="text-sm font-medium text-green-900">Upload completed successfully!</p>
                <p class="text-xs text-green-600 mt-1">{{ uploadTask.message }}</p>
              </div>
            </div>
          </Card>

          <!-- 上传失败 -->
          <Card
            v-if="!uploading && uploadTask?.status === 'failed'"
            padding="sm"
            class="bg-red-50 border-red-200"
          >
            <div class="flex items-start space-x-3">
              <XCircleIcon class="h-5 w-5 text-red-600 flex-shrink-0 mt-0.5" />
              <div class="flex-1">
                <p class="text-sm font-medium text-red-900">Upload failed</p>
                <p class="text-xs text-red-600 mt-1">{{ uploadTask.error || uploadTask.message }}</p>
              </div>
            </div>
          </Card>

          <!-- 操作按钮 -->
          <div class="flex gap-3 pt-4 border-t">
            <!-- Download button -->
            <Button
              v-if="activeTab === 'download' && !alreadyInstalled"
              variant="primary"
              :loading="downloading"
              :disabled="downloading || uploading"
              @click="startDownload"
            >
              {{ downloading ? 'Downloading...' : 'Download Dashboard' }}
            </Button>

            <!-- Upload button -->
            <Button
              v-if="activeTab === 'upload' && !alreadyInstalled"
              variant="primary"
              :loading="uploading"
              :disabled="downloading || uploading || !uploadFile"
              @click="startUpload"
            >
              {{ uploading ? 'Uploading...' : 'Upload Dashboard' }}
            </Button>

            <!-- Re-download/Re-upload button -->
            <Button
              v-if="alreadyInstalled"
              variant="secondary"
              :disabled="downloading || uploading"
              @click="activeTab === 'upload' ? startUpload : startDownload"
            >
              {{ activeTab === 'upload' ? 'Re-upload' : 'Re-download' }}
            </Button>

            <!-- Continue button -->
            <Button
              v-if="alreadyInstalled || downloadTask?.status === 'completed' || uploadTask?.status === 'completed'"
              variant="primary"
              @click="handleNext"
            >
              Continue to Next Step
            </Button>

            <!-- Skip button -->
            <Button
              v-if="!downloading && !uploading"
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
          <p class="font-medium text-gray-900">About Dashboard:</p>
          <ul class="list-disc list-inside space-y-1 ml-2">
            <li>Modern web-based dashboard for sing-box</li>
            <li>Supports Clash API for real-time control</li>
            <li>View connections, switch proxies, update rules</li>
            <li>Access via http://127.0.0.1:9090/ui after installation (adjust port if you changed it)</li>
            <li>Download URL is configured in the previous step (Experimental Features)</li>
          </ul>
        </div>
      </Card>
    </div>
  </div>
</template>
