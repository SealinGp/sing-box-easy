<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { apiService } from '../../services/api'
import type { InstallTask } from '../../types/api'
import { Button, Input, Alert, Card, Loading } from '../../components'
import { CheckCircleIcon, XCircleIcon } from '@heroicons/vue/24/outline'

const emit = defineEmits<{
  next: []
  prev: []
}>()

const version = ref('1.13.0')
const beta = ref(false)
const installing = ref(false)
const installTask = ref<InstallTask | null>(null)
const error = ref('')
const checkingExisting = ref(true)
const alreadyInstalled = ref(false)

let pollInterval: ReturnType<typeof setInterval> | null = null

onMounted(async () => {
  // 检查是否已经安装
  try {
    const initStatus = await apiService.getInitStatus()
    if (initStatus.singbox_installed) {
      alreadyInstalled.value = true
    }
  } catch (err: any) {
    console.error('Failed to check install status:', err)
  } finally {
    checkingExisting.value = false
  }
})

const startInstall = async () => {
  if (!version.value.trim()) {
    error.value = 'Please enter a version'
    return
  }

  error.value = ''
  installing.value = true

  try {
    const task = await apiService.installSingBox(
      version.value.trim(),
      beta.value
    )
    installTask.value = task

    // 开始轮询任务状态
    pollTaskStatus(task.id)
  } catch (err: any) {
    error.value = err.response?.data?.error || err.message || 'Failed to start installation'
    installing.value = false
  }
}

const pollTaskStatus = (taskId: string) => {
  pollInterval = setInterval(async () => {
    try {
      const task = await apiService.getInstallStatus(taskId)
      installTask.value = task

      if (task.status === 'completed') {
        stopPolling()
        installing.value = false
        alreadyInstalled.value = true
      } else if (task.status === 'failed') {
        stopPolling()
        installing.value = false
        error.value = task.error || 'Installation failed'
      }
    } catch (err: any) {
      console.error('Failed to poll task status:', err)
      stopPolling()
      installing.value = false
      error.value = 'Failed to check installation status'
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

// 组件卸载时清理定时器
import { onUnmounted } from 'vue'
onUnmounted(() => {
  stopPolling()
})
</script>

<template>
  <div class="space-y-6">
    <!-- 检查现有安装状态 -->
    <div v-if="checkingExisting" class="flex justify-center py-8">
      <Loading size="lg" text="Checking installation status..." />
    </div>

    <!-- 已安装提示 -->
    <Alert v-else-if="alreadyInstalled" type="success" title="sing-box Already Installed">
      sing-box is already installed on your system. You can proceed to the next step or reinstall if needed.
    </Alert>

    <!-- 安装表单 -->
    <Card v-if="!checkingExisting">
      <div class="space-y-4">
        <div>
          <h3 class="text-lg font-semibold text-gray-900 mb-2">Install sing-box</h3>
          <p class="text-sm text-gray-600">
            sing-box is a universal proxy platform. Please specify the version you want to install.
          </p>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Input
            v-model="version"
            label="Version"
            placeholder="e.g., 1.13.0"
            :disabled="true"
            required
          />

          <div class="flex items-end">
            <label class="flex items-center space-x-2 cursor-pointer">
              <input
                v-model="beta"
                type="checkbox"
                class="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
                :disabled="installing || alreadyInstalled"
              />
              <span class="text-sm font-medium text-gray-700">Install beta version</span>
            </label>
          </div>
        </div>

        <!-- 错误提示 -->
        <Alert v-if="error" type="error" closable @close="error = ''">
          {{ error }}
        </Alert>

        <!-- 安装进度 -->
        <Card v-if="installing && installTask" padding="sm" class="bg-blue-50 border-blue-200">
          <div class="flex items-start space-x-3">
            <Loading size="sm" />
            <div class="flex-1">
              <p class="text-sm font-medium text-blue-900">{{ installTask.message }}</p>
              <p class="text-xs text-blue-600 mt-1">This may take a few minutes...</p>
            </div>
          </div>
        </Card>

        <!-- 安装成功 -->
        <Card
          v-if="!installing && installTask?.status === 'completed'"
          padding="sm"
          class="bg-green-50 border-green-200"
        >
          <div class="flex items-start space-x-3">
            <CheckCircleIcon class="h-5 w-5 text-green-600 flex-shrink-0 mt-0.5" />
            <div class="flex-1">
              <p class="text-sm font-medium text-green-900">Installation completed successfully!</p>
              <p class="text-xs text-green-600 mt-1">{{ installTask.message }}</p>
            </div>
          </div>
        </Card>

        <!-- 安装失败 -->
        <Card
          v-if="!installing && installTask?.status === 'failed'"
          padding="sm"
          class="bg-red-50 border-red-200"
        >
          <div class="flex items-start space-x-3">
            <XCircleIcon class="h-5 w-5 text-red-600 flex-shrink-0 mt-0.5" />
            <div class="flex-1">
              <p class="text-sm font-medium text-red-900">Installation failed</p>
              <p class="text-xs text-red-600 mt-1">{{ installTask.error || installTask.message }}</p>
            </div>
          </div>
        </Card>

        <!-- 操作按钮 -->
        <div class="flex gap-3">
          <Button
            v-if="!alreadyInstalled"
            variant="primary"
            :loading="installing"
            :disabled="installing"
            @click="startInstall"
          >
            {{ installing ? 'Installing...' : 'Install sing-box' }}
          </Button>

          <Button
            v-if="alreadyInstalled"
            variant="secondary"
            :disabled="installing"
            @click="startInstall"
          >
            Reinstall
          </Button>

          <Button
            v-if="alreadyInstalled || installTask?.status === 'completed'"
            variant="primary"
            @click="handleNext"
          >
            Continue to Next Step
          </Button>

          <Button
            v-if="!alreadyInstalled && !installing"
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
        <p class="font-medium text-gray-900">Notes:</p>
        <ul class="list-disc list-inside space-y-1 ml-2">
          <li>The installation uses the official sing-box installation script</li>
          <li>Default version is 1.13.0 (recommended)</li>
          <li>Beta versions may contain experimental features</li>
          <li>You can skip this step if sing-box is already installed manually</li>
        </ul>
      </div>
    </Card>
  </div>
</template>
