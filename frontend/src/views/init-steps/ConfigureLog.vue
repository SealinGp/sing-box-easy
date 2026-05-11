<script setup lang="ts">
import { ref, onMounted } from 'vue'
import type { LogConfig } from '../../types/api'
import { Button, Input, Select, Alert, Card, Loading } from '../../components'
import { InformationCircleIcon } from '@heroicons/vue/24/outline'
import { logService } from '../../services'

const emit = defineEmits<{
  next: []
  prev: []
}>()

const loading = ref(false)
const saving = ref(false)
const error = ref('')
const success = ref(false)

// 日志配置
const logConfig = ref<LogConfig>({
  disabled: false,
  level: 'info',
  output: '',
  timestamp: true,
})

// 日志级别选项
const logLevelOptions = [
  { label: 'Trace (Most Verbose)', value: 'trace' },
  { label: 'Debug', value: 'debug' },
  { label: 'Info (Recommended)', value: 'info' },
  { label: 'Warn', value: 'warn' },
  { label: 'Error (Least Verbose)', value: 'error' },
]

onMounted(async () => {
  await loadLogConfig()
})

const loadLogConfig = async () => {
  loading.value = true
  error.value = ''

  try {
    const {data} = await logService.getLog()
    const config = data    
    if (config) {
      logConfig.value = {
        disabled: config.disabled || false,
        level: config.level || 'info',
        output: config.output || '',
        timestamp: config.timestamp !== undefined ? config.timestamp : true,
      }
    }
  } catch (err: any) {
    // 如果配置不存在，使用默认值
    console.log('No existing log config, using defaults')
  } finally {
    loading.value = false
  }
}

const saveLogConfig = async () => {
  saving.value = true
  error.value = ''
  success.value = false

  try {
    await logService.updateLog(logConfig.value)
    success.value = true

    // 2秒后自动进入下一步
    setTimeout(() => {
      emit('next')
    }, 2000)
  } catch (err: any) {
    error.value = err.message || 'Failed to save log configuration'
  } finally {
    saving.value = false
  }
}

const handleNext = () => {
  if (!success.value) {
    saveLogConfig()
  } else {
    emit('next')
  }
}

const handleSkip = () => {
  emit('next')
}
</script>

<template>
  <div class="space-y-6">
    <!-- 加载状态 -->
    <div v-if="loading" class="flex justify-center py-8">
      <Loading size="lg" text="Loading log configuration..." />
    </div>

    <!-- 配置表单 -->
    <div v-else class="space-y-6">
      <!-- 成功提示 -->
      <Alert v-if="success" type="success" title="Configuration Saved">
        Log configuration has been saved successfully. Proceeding to next step...
      </Alert>

      <!-- 错误提示 -->
      <Alert v-if="error" type="error" closable @close="error = ''">
        {{ error }}
      </Alert>

      <Card>
        <div class="space-y-4">
          <div>
            <h3 class="text-lg font-semibold text-gray-900 mb-2">Log Configuration</h3>
            <p class="text-sm text-gray-600">
              Configure logging behavior for sing-box. Logs help troubleshoot issues and monitor system activity.
            </p>
          </div>

          <!-- 启用/禁用日志 -->
          <div class="flex items-center space-x-3 p-4 bg-gray-50 rounded-lg">
            <input
              v-model="logConfig.disabled"
              type="checkbox"
              id="disable-log"
              class="w-4 h-4 text-violet-600 border-gray-300 rounded focus:ring-violet-500"
            />
            <label for="disable-log" class="flex-1 cursor-pointer">
              <span class="text-sm font-medium text-gray-900">Disable Logging</span>
              <p class="text-xs text-gray-500 mt-0.5">
                Turn off all logging (not recommended for initial setup)
              </p>
            </label>
          </div>

          <!-- 日志级别 -->
          <Select
            v-model="logConfig.level"
            :options="logLevelOptions"
            label="Log Level"
            :disabled="logConfig.disabled"
            placeholder="Select log level"
          />

          <!-- 日志输出路径 -->
          <Input
            v-model="logConfig.output"
            label="Log Output Path"
            placeholder="e.g., /var/log/sing-box/sing-box.log (leave empty for stdout), Will not write log to console after enable."
            :disabled="logConfig.disabled"
          />

          <!-- 时间戳 -->
          <div class="flex items-center space-x-3 p-4 bg-gray-50 rounded-lg">
            <input
              v-model="logConfig.timestamp"
              type="checkbox"
              id="enable-timestamp"
              class="w-4 h-4 text-violet-600 border-gray-300 rounded focus:ring-violet-500"
              :disabled="logConfig.disabled"
            />
            <label for="enable-timestamp" class="flex-1 cursor-pointer">
              <span class="text-sm font-medium text-gray-900">Include Timestamps</span>
              <p class="text-xs text-gray-500 mt-0.5">
                Add timestamp to each log entry
              </p>
            </label>
          </div>

          <!-- 操作按钮 -->
          <div class="flex gap-3 pt-4 border-t">
            <Button
              variant="primary"
              :loading="saving"
              :disabled="saving || success"
              @click="handleNext"
            >
              {{ success ? 'Saved' : saving ? 'Saving...' : 'Save & Continue' }}
            </Button>

            <Button
              variant="ghost"
              :disabled="saving || success"
              @click="handleSkip"
            >
              Skip this step
            </Button>
          </div>
        </div>
      </Card>

      <!-- 配置说明 -->
      <Card padding="sm" class="bg-violet-50 border-violet-200">
        <div class="flex items-start space-x-3">
          <InformationCircleIcon class="h-5 w-5 text-violet-600 flex-shrink-0 mt-0.5" />
          <div class="text-sm text-violet-900 space-y-2">
            <p class="font-medium">Log Level Guide:</p>
            <ul class="list-disc list-inside space-y-1 ml-2 text-violet-800">
              <li><strong>Trace:</strong> Most detailed, includes all debug info (high disk usage)</li>
              <li><strong>Debug:</strong> Detailed information for debugging</li>
              <li><strong>Info:</strong> General operational messages (recommended for production)</li>
              <li><strong>Warn:</strong> Warning messages for potential issues</li>
              <li><strong>Error:</strong> Only error messages (minimal logging)</li>
            </ul>
            <p class="mt-2 text-xs text-violet-700">
              💡 Tip: Use "Info" level for normal operation and "Debug" for troubleshooting.
            </p>
          </div>
        </div>
      </Card>

      <!-- 输出路径说明 -->
      <Card padding="sm" class="bg-gray-50">
        <div class="text-sm text-gray-600 space-y-2">
          <p class="font-medium text-gray-900">Log Output Options:</p>
          <ul class="list-disc list-inside space-y-1 ml-2">
            <li><strong>Empty (stdout):</strong> Logs output to standard output (console)</li>
            <li><strong>File path:</strong> Logs written to specified file (e.g., /var/log/sing-box.log)</li>
            <li>Make sure the directory exists and sing-box has write permissions</li>
          </ul>
        </div>
      </Card>
    </div>
  </div>
</template>
