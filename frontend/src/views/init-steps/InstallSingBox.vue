<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import type { InstallTask } from '../../types/api'
import { Button, Input, Alert, Card, Loading } from '../../components'
import { CheckCircleIcon, XCircleIcon } from '@heroicons/vue/24/outline'
import { serviceControlService } from '../../services'

const { t } = useI18n()

const emit = defineEmits<{
  next: []
  prev: []
}>()

const version = ref('1.12.12')
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
    const {data} = await serviceControlService.getInitStatus()
    const initStatus = data
    if (initStatus.steps.sing_box_installed) {
      alreadyInstalled.value = true
      // Show the actually-detected version instead of the hardcoded default.
      // Only override when the backend has a non-empty value — otherwise the
      // input would blank out and break the "reinstall" form.
      if (initStatus.sing_box_version) {
        version.value = initStatus.sing_box_version
      }
    }
  } catch (err: any) {
    console.error('Failed to check install status:', err)
  } finally {
    checkingExisting.value = false
  }
})

const startInstall = async () => {
  if (!version.value.trim()) {
    error.value = t('init.install.versionRequired')
    return
  }

  error.value = ''
  installing.value = true

  try {
    const {data} = await serviceControlService.installSingBox(
      version.value.trim(),
      beta.value
    )

    // Initialize task with pending status
    installTask.value = {
      id: data.task_id,
      status: 'running',
      message: data.message,
    }

    // 开始轮询任务状态
    pollTaskStatus(data.task_id)
  } catch (err: any) {
    error.value = err.message || t('init.install.startFailed')
    installing.value = false
  }
}

const pollTaskStatus = (taskId: string) => {
  pollInterval = setInterval(async () => {
    try {
      const {data} = await serviceControlService.getInstallTask(taskId)
      const task = data
      installTask.value = data

      if (task.status === 'completed') {
        stopPolling()
        installing.value = false
        alreadyInstalled.value = true
      } else if (task.status === 'failed') {
        stopPolling()
        installing.value = false
        // Don't set error.value here - let the Card component show the error
      }
    } catch (err: any) {
      console.error('Failed to poll task status:', err)
      stopPolling()
      installing.value = false
      error.value = t('init.install.statusFailed')
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
onUnmounted(() => {
  stopPolling()
})
</script>

<template>
  <div class="space-y-4">
    <!-- 检查现有安装状态 -->
    <div v-if="checkingExisting" class="flex justify-center py-6">
      <Loading size="lg" :text="$t('init.install.checking')" />
    </div>

    <!-- 已安装提示 -->
    <Alert v-else-if="alreadyInstalled" type="success" :title="$t('init.install.alreadyTitle')">
      {{ $t('init.install.alreadyDesc') }}
    </Alert>

    <!-- 安装表单 -->
    <Card v-if="!checkingExisting">
      <div class="space-y-4">
        <div>
          <h3 class="text-lg font-semibold text-gray-900 mb-2">{{ $t('init.install.heading') }}</h3>
          <p class="text-sm text-gray-600">
            {{ $t('init.install.intro') }}
          </p>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Input
            v-model="version"
            :label="$t('init.install.versionLabel')"
            :placeholder="$t('init.install.versionPlaceholder')"
            :disabled="true"
            required
          />

          <div class="flex items-end">
            <label class="flex items-center space-x-2 cursor-pointer">
              <input
                v-model="beta"
                type="checkbox"
                class="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500"
                :disabled="installing || alreadyInstalled"
              />
              <span class="text-sm font-medium text-gray-700">{{ $t('init.install.betaLabel') }}</span>
            </label>
          </div>
        </div>

        <!-- 错误提示 -->
        <Alert v-if="error" type="error" closable @close="error = ''">
          {{ error }}
        </Alert>

        <!-- 安装进度 -->
        <Card v-if="installing && installTask" padding="sm" class="bg-primary-50 border-primary-200">
          <div class="space-y-3">
            <div class="flex items-center space-x-2">
              <Loading size="sm" />
              <p class="text-sm font-semibold text-primary-900">{{ $t('init.install.progressTitle') }}</p>
            </div>
            <div class="bg-gray-900 text-green-400 p-3 rounded font-mono text-xs overflow-auto max-h-32">
              <pre class="whitespace-pre-wrap break-words">{{ installTask.message || $t('init.install.progressStarting') }}</pre>
            </div>
            <p class="text-xs text-primary-600">{{ $t('init.install.progressWait') }}</p>
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
              <p class="text-sm font-medium text-green-900">{{ $t('init.install.successTitle') }}</p>
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
              <p class="text-sm font-medium text-red-900">{{ $t('init.install.failedTitle') }}</p>
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
            {{ installing ? $t('init.install.installingBtn') : $t('init.install.installBtn') }}
          </Button>

          <Button
            v-if="alreadyInstalled"
            variant="secondary"
            :disabled="installing"
            @click="startInstall"
          >
            {{ $t('init.install.reinstallBtn') }}
          </Button>

          <Button
            v-if="alreadyInstalled || installTask?.status === 'completed'"
            variant="primary"
            @click="handleNext"
          >
            {{ $t('init.install.continueBtn') }}
          </Button>

          <Button
            v-if="!alreadyInstalled && !installing"
            variant="ghost"
            @click="handleSkip"
          >
            {{ $t('init.install.skipBtn') }}
          </Button>
        </div>
      </div>
    </Card>

    <!-- 说明信息 -->
    <Card padding="sm" class="bg-gray-50">
      <div class="text-sm text-gray-600 space-y-2">
        <p class="font-medium text-gray-900">{{ $t('init.install.notesTitle') }}</p>
        <ul class="list-disc list-inside space-y-1 ml-2">
          <li>{{ $t('init.install.notes.script') }}</li>
          <li>{{ $t('init.install.notes.default') }}</li>
          <li>{{ $t('init.install.notes.beta') }}</li>
          <li>{{ $t('init.install.notes.skip') }}</li>
        </ul>
      </div>
    </Card>
  </div>
</template>
