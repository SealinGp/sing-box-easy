<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { Button, Alert, Card } from '../../components'
import { CheckCircleIcon, RocketLaunchIcon, Cog6ToothIcon } from '@heroicons/vue/24/outline'
import { serviceControlService } from '../../services'

const { t } = useI18n()
const router = useRouter()

const completing = ref(false)
const error = ref('')

const completeSetup = async () => {
  completing.value = true
  error.value = ''

  try {
    // 标记初始化完成
    await serviceControlService.completeInit()

    // 跳转到管理面板
    router.push('/dashboard')
  } catch (err: any) {
    error.value = err.message || t('init.complete.completeFailed')
  } finally {
    completing.value = false
  }
}

const goToDashboard = () => {
  router.push('/dashboard')
}
</script>

<template>
  <div class="space-y-4">
    <!-- 错误提示 -->
    <Alert v-if="error" type="error" closable @close="error = ''">
      {{ error }}
    </Alert>

    <!-- 完成祝贺 -->
    <Card class="text-center py-6">
      <div class="flex justify-center mb-4">
        <div class="w-20 h-20 bg-green-100 rounded-pill flex items-center justify-center">
          <CheckCircleIcon class="w-12 h-12 text-green-600" />
        </div>
      </div>
      <h2 class="text-3xl font-bold text-gray-900 mb-2">{{ $t('init.complete.title') }}</h2>
      <p class="text-gray-600 max-w-md mx-auto">
        {{ $t('init.complete.congrats') }}
      </p>
    </Card>

    <!-- 配置摘要 -->
    <Card>
      <div class="space-y-4">
        <h3 class="text-lg font-semibold text-gray-900 mb-4">{{ $t('init.complete.summaryTitle') }}</h3>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <!-- sing-box 安装 -->
          <div class="flex items-start space-x-3 p-3 bg-gray-50 rounded-control">
            <CheckCircleIcon class="w-5 h-5 text-green-600 flex-shrink-0 mt-0.5" />
            <div>
              <p class="text-sm font-medium text-gray-900">{{ $t('init.complete.summary.installTitle') }}</p>
              <p class="text-xs text-gray-600 mt-0.5">{{ $t('init.complete.summary.installDesc') }}</p>
            </div>
          </div>

          <!-- 日志配置 -->
          <div class="flex items-start space-x-3 p-3 bg-gray-50 rounded-control">
            <CheckCircleIcon class="w-5 h-5 text-green-600 flex-shrink-0 mt-0.5" />
            <div>
              <p class="text-sm font-medium text-gray-900">{{ $t('init.complete.summary.logTitle') }}</p>
              <p class="text-xs text-gray-600 mt-0.5">{{ $t('init.complete.summary.logDesc') }}</p>
            </div>
          </div>

          <!-- 实验性功能 -->
          <div class="flex items-start space-x-3 p-3 bg-gray-50 rounded-control">
            <CheckCircleIcon class="w-5 h-5 text-green-600 flex-shrink-0 mt-0.5" />
            <div>
              <p class="text-sm font-medium text-gray-900">{{ $t('init.complete.summary.experimentalTitle') }}</p>
              <p class="text-xs text-gray-600 mt-0.5">{{ $t('init.complete.summary.experimentalDesc') }}</p>
            </div>
          </div>

          <!-- 出站节点 -->
          <div class="flex items-start space-x-3 p-3 bg-gray-50 rounded-control">
            <CheckCircleIcon class="w-5 h-5 text-green-600 flex-shrink-0 mt-0.5" />
            <div>
              <p class="text-sm font-medium text-gray-900">{{ $t('init.complete.summary.outboundTitle') }}</p>
              <p class="text-xs text-gray-600 mt-0.5">{{ $t('init.complete.summary.outboundDesc') }}</p>
            </div>
          </div>

          <!-- 规则集 -->
          <div class="flex items-start space-x-3 p-3 bg-gray-50 rounded-control">
            <CheckCircleIcon class="w-5 h-5 text-green-600 flex-shrink-0 mt-0.5" />
            <div>
              <p class="text-sm font-medium text-gray-900">{{ $t('init.complete.summary.ruleSetsTitle') }}</p>
              <p class="text-xs text-gray-600 mt-0.5">{{ $t('init.complete.summary.ruleSetsDesc') }}</p>
            </div>
          </div>

          <!-- DNS -->
          <div class="flex items-start space-x-3 p-3 bg-gray-50 rounded-control">
            <CheckCircleIcon class="w-5 h-5 text-green-600 flex-shrink-0 mt-0.5" />
            <div>
              <p class="text-sm font-medium text-gray-900">{{ $t('init.complete.summary.dnsTitle') }}</p>
              <p class="text-xs text-gray-600 mt-0.5">{{ $t('init.complete.summary.dnsDesc') }}</p>
            </div>
          </div>

          <!-- 入站 -->
          <div class="flex items-start space-x-3 p-3 bg-gray-50 rounded-control">
            <CheckCircleIcon class="w-5 h-5 text-green-600 flex-shrink-0 mt-0.5" />
            <div>
              <p class="text-sm font-medium text-gray-900">{{ $t('init.complete.summary.inboundTitle') }}</p>
              <p class="text-xs text-gray-600 mt-0.5">{{ $t('init.complete.summary.inboundDesc') }}</p>
            </div>
          </div>

          <!-- 路由规则 -->
          <div class="flex items-start space-x-3 p-3 bg-gray-50 rounded-control">
            <CheckCircleIcon class="w-5 h-5 text-green-600 flex-shrink-0 mt-0.5" />
            <div>
              <p class="text-sm font-medium text-gray-900">{{ $t('init.complete.summary.routingTitle') }}</p>
              <p class="text-xs text-gray-600 mt-0.5">{{ $t('init.complete.summary.routingDesc') }}</p>
            </div>
          </div>
        </div>
      </div>
    </Card>

    <!-- 下一步操作 -->
    <Card class="bg-primary-50 border-primary-200">
      <div class="space-y-4">
        <div class="flex items-start space-x-3">
          <RocketLaunchIcon class="w-6 h-6 text-primary-600 flex-shrink-0 mt-0.5" />
          <div>
            <h3 class="text-lg font-semibold text-primary-900 mb-2">{{ $t('init.complete.nextStepsTitle') }}</h3>
            <ul class="space-y-2 text-sm text-primary-800">
              <li class="flex items-start">
                <span class="mr-2">1.</span>
                <span>{{ $t('init.complete.nextSteps.start') }}</span>
              </li>
              <li class="flex items-start">
                <span class="mr-2">2.</span>
                <span>{{ $t('init.complete.nextSteps.configure') }}</span>
              </li>
              <li class="flex items-start">
                <span class="mr-2">3.</span>
                <span>{{ $t('init.complete.nextSteps.monitor') }}</span>
              </li>
              <li class="flex items-start">
                <span class="mr-2">4.</span>
                <span>{{ $t('init.complete.nextSteps.access') }}</span>
              </li>
            </ul>
          </div>
        </div>
      </div>
    </Card>

    <!-- 快速访问 -->
    <Card>
      <div class="space-y-4">
        <div class="flex items-start space-x-3">
          <Cog6ToothIcon class="w-6 h-6 text-gray-600 flex-shrink-0 mt-0.5" />
          <div class="flex-1">
            <h3 class="text-lg font-semibold text-gray-900 mb-2">{{ $t('init.complete.quickAccessTitle') }}</h3>
            <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
              <div class="p-3 bg-gray-50 rounded-control">
                <p class="text-sm font-medium text-gray-900">{{ $t('init.complete.quickAccess.proxy') }}</p>
                <code class="text-xs text-gray-600 font-mono">127.0.0.1:7890</code>
              </div>
              <div class="p-3 bg-gray-50 rounded-control">
                <p class="text-sm font-medium text-gray-900">{{ $t('init.complete.quickAccess.clashApi') }}</p>
                <code class="text-xs text-gray-600 font-mono">127.0.0.1:9090</code>
              </div>
              <div class="p-3 bg-gray-50 rounded-control">
                <p class="text-sm font-medium text-gray-900">{{ $t('init.complete.quickAccess.webDashboard') }}</p>
                <code class="text-xs text-gray-600 font-mono">/ui</code>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Card>

    <!-- 操作按钮 -->
    <div class="flex gap-3 justify-center">
      <Button
        variant="primary"
        size="lg"
        :loading="completing"
        :disabled="completing"
        @click="completeSetup"
      >
        {{ completing ? $t('init.complete.completingBtn') : $t('init.complete.completeBtn') }}
      </Button>

      <Button
        variant="secondary"
        size="lg"
        :disabled="completing"
        @click="goToDashboard"
      >
        {{ $t('init.complete.skipBtn') }}
      </Button>
    </div>

    <!-- 提示信息 -->
    <div class="text-center text-sm text-gray-500">
      <p>{{ $t('init.complete.reconfigureHint') }}</p>
    </div>
  </div>
</template>
