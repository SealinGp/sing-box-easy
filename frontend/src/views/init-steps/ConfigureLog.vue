<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import type { LogConfig } from '../../types/api'
import { Button, Input, Alert, Card, Loading } from '../../components'
import { Select } from '../../volt'
import { InformationCircleIcon } from '@heroicons/vue/24/outline'
import { logService } from '../../services'

const { t } = useI18n()

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
const logLevelOptions = computed(() => [
  { label: t('init.log.levelOptions.trace'), value: 'trace' },
  { label: t('init.log.levelOptions.debug'), value: 'debug' },
  { label: t('init.log.levelOptions.info'), value: 'info' },
  { label: t('init.log.levelOptions.warn'), value: 'warn' },
  { label: t('init.log.levelOptions.error'), value: 'error' },
])

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
    error.value = err.message || t('init.log.saveFailed')
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
      <Loading size="lg" :text="$t('init.log.loading')" />
    </div>

    <!-- 配置表单 -->
    <div v-else class="space-y-6">
      <!-- 成功提示 -->
      <Alert v-if="success" type="success" :title="$t('init.log.savedTitle')">
        {{ $t('init.log.savedDesc') }}
      </Alert>

      <!-- 错误提示 -->
      <Alert v-if="error" type="error" closable @close="error = ''">
        {{ error }}
      </Alert>

      <Card>
        <div class="space-y-4">
          <div>
            <h3 class="text-lg font-semibold text-gray-900 mb-2">{{ $t('init.log.heading') }}</h3>
            <p class="text-sm text-gray-600">
              {{ $t('init.log.intro') }}
            </p>
          </div>

          <!-- 启用/禁用日志 -->
          <div class="flex items-center space-x-3 p-4 bg-gray-50 rounded-surface">
            <input
              v-model="logConfig.disabled"
              type="checkbox"
              id="disable-log"
              class="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500"
            />
            <label for="disable-log" class="flex-1 cursor-pointer">
              <span class="text-sm font-medium text-gray-900">{{ $t('init.log.disableLabel') }}</span>
              <p class="text-xs text-gray-500 mt-0.5">
                {{ $t('init.log.disableDesc') }}
              </p>
            </label>
          </div>

          <!-- 日志级别 -->
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('init.log.levelLabel') }}</label>
          <Select
            class="w-full"
            v-model="logConfig.level"
            :options="logLevelOptions"
            optionLabel="label"
            optionValue="value"
            :disabled="logConfig.disabled"
            :placeholder="$t('init.log.levelPlaceholder')"
          />

          <!-- 日志输出路径 -->
          <Input
            v-model="logConfig.output"
            :label="$t('init.log.outputLabel')"
            :placeholder="$t('init.log.outputPlaceholder')"
            :disabled="logConfig.disabled"
          />

          <!-- 时间戳 -->
          <div class="flex items-center space-x-3 p-4 bg-gray-50 rounded-surface">
            <input
              v-model="logConfig.timestamp"
              type="checkbox"
              id="enable-timestamp"
              class="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500"
              :disabled="logConfig.disabled"
            />
            <label for="enable-timestamp" class="flex-1 cursor-pointer">
              <span class="text-sm font-medium text-gray-900">{{ $t('init.log.timestampLabel') }}</span>
              <p class="text-xs text-gray-500 mt-0.5">
                {{ $t('init.log.timestampDesc') }}
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
              {{ success ? $t('init.log.savedBtn') : saving ? $t('common.saving') : $t('init.log.saveContinueBtn') }}
            </Button>

            <Button
              variant="ghost"
              :disabled="saving || success"
              @click="handleSkip"
            >
              {{ $t('init.log.skipBtn') }}
            </Button>
          </div>
        </div>
      </Card>

      <!-- 配置说明 -->
      <Card padding="sm" class="bg-primary-50 border-primary-200">
        <div class="flex items-start space-x-3">
          <InformationCircleIcon class="h-5 w-5 text-primary-600 flex-shrink-0 mt-0.5" />
          <div class="text-sm text-primary-900 space-y-2">
            <p class="font-medium">{{ $t('init.log.guideTitle') }}</p>
            <ul class="list-disc list-inside space-y-1 ml-2 text-primary-800">
              <li><strong>Trace:</strong> {{ $t('init.log.guide.trace') }}</li>
              <li><strong>Debug:</strong> {{ $t('init.log.guide.debug') }}</li>
              <li><strong>Info:</strong> {{ $t('init.log.guide.info') }}</li>
              <li><strong>Warn:</strong> {{ $t('init.log.guide.warn') }}</li>
              <li><strong>Error:</strong> {{ $t('init.log.guide.error') }}</li>
            </ul>
            <p class="mt-2 text-xs text-primary-700">
              💡 {{ $t('init.log.guideTip') }}
            </p>
          </div>
        </div>
      </Card>

      <!-- 输出路径说明 -->
      <Card padding="sm" class="bg-gray-50">
        <div class="text-sm text-gray-600 space-y-2">
          <p class="font-medium text-gray-900">{{ $t('init.log.outputTitle') }}</p>
          <ul class="list-disc list-inside space-y-1 ml-2">
            <li><strong>Empty (stdout):</strong> {{ $t('init.log.output.stdout') }}</li>
            <li><strong>File path:</strong> {{ $t('init.log.output.file') }}</li>
            <li>{{ $t('init.log.output.perms') }}</li>
          </ul>
        </div>
      </Card>
    </div>
  </div>
</template>
