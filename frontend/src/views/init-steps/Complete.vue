<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { apiService } from '../../services/api'
import { Button, Alert, Card } from '../../components'
import { CheckCircleIcon, RocketLaunchIcon, Cog6ToothIcon } from '@heroicons/vue/24/outline'

const router = useRouter()

const completing = ref(false)
const error = ref('')

const completeSetup = async () => {
  completing.value = true
  error.value = ''

  try {
    // 标记初始化完成
    await apiService.completeInit()

    // 跳转到管理面板
    router.push('/dashboard')
  } catch (err: any) {
    error.value = err.response?.data?.error || err.message || 'Failed to complete initialization'
  } finally {
    completing.value = false
  }
}

const goToDashboard = () => {
  router.push('/dashboard')
}
</script>

<template>
  <div class="space-y-6">
    <!-- 错误提示 -->
    <Alert v-if="error" type="error" closable @close="error = ''">
      {{ error }}
    </Alert>

    <!-- 完成祝贺 -->
    <Card class="text-center py-8">
      <div class="flex justify-center mb-4">
        <div class="w-20 h-20 bg-green-100 rounded-full flex items-center justify-center">
          <CheckCircleIcon class="w-12 h-12 text-green-600" />
        </div>
      </div>
      <h2 class="text-3xl font-bold text-gray-900 mb-2">Setup Complete!</h2>
      <p class="text-gray-600 max-w-md mx-auto">
        Congratulations! You've successfully configured sing-box. Your proxy is ready to use.
      </p>
    </Card>

    <!-- 配置摘要 -->
    <Card>
      <div class="space-y-4">
        <h3 class="text-lg font-semibold text-gray-900 mb-4">Configuration Summary</h3>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <!-- sing-box 安装 -->
          <div class="flex items-start space-x-3 p-3 bg-gray-50 rounded-lg">
            <CheckCircleIcon class="w-5 h-5 text-green-600 flex-shrink-0 mt-0.5" />
            <div>
              <p class="text-sm font-medium text-gray-900">sing-box Installed</p>
              <p class="text-xs text-gray-600 mt-0.5">Core proxy service is ready</p>
            </div>
          </div>

          <!-- 日志配置 -->
          <div class="flex items-start space-x-3 p-3 bg-gray-50 rounded-lg">
            <CheckCircleIcon class="w-5 h-5 text-green-600 flex-shrink-0 mt-0.5" />
            <div>
              <p class="text-sm font-medium text-gray-900">Logging Configured</p>
              <p class="text-xs text-gray-600 mt-0.5">Logs are being recorded</p>
            </div>
          </div>

          <!-- 实验性功能 -->
          <div class="flex items-start space-x-3 p-3 bg-gray-50 rounded-lg">
            <CheckCircleIcon class="w-5 h-5 text-green-600 flex-shrink-0 mt-0.5" />
            <div>
              <p class="text-sm font-medium text-gray-900">Experimental Features</p>
              <p class="text-xs text-gray-600 mt-0.5">Clash API and cache configured</p>
            </div>
          </div>

          <!-- 出站节点 -->
          <div class="flex items-start space-x-3 p-3 bg-gray-50 rounded-lg">
            <CheckCircleIcon class="w-5 h-5 text-green-600 flex-shrink-0 mt-0.5" />
            <div>
              <p class="text-sm font-medium text-gray-900">Outbound Nodes</p>
              <p class="text-xs text-gray-600 mt-0.5">Proxy servers configured</p>
            </div>
          </div>

          <!-- 规则集 -->
          <div class="flex items-start space-x-3 p-3 bg-gray-50 rounded-lg">
            <CheckCircleIcon class="w-5 h-5 text-green-600 flex-shrink-0 mt-0.5" />
            <div>
              <p class="text-sm font-medium text-gray-900">Rule Sets</p>
              <p class="text-xs text-gray-600 mt-0.5">Routing rules configured</p>
            </div>
          </div>

          <!-- DNS -->
          <div class="flex items-start space-x-3 p-3 bg-gray-50 rounded-lg">
            <CheckCircleIcon class="w-5 h-5 text-green-600 flex-shrink-0 mt-0.5" />
            <div>
              <p class="text-sm font-medium text-gray-900">DNS Configuration</p>
              <p class="text-xs text-gray-600 mt-0.5">DNS servers configured</p>
            </div>
          </div>

          <!-- 入站 -->
          <div class="flex items-start space-x-3 p-3 bg-gray-50 rounded-lg">
            <CheckCircleIcon class="w-5 h-5 text-green-600 flex-shrink-0 mt-0.5" />
            <div>
              <p class="text-sm font-medium text-gray-900">Inbound Proxies</p>
              <p class="text-xs text-gray-600 mt-0.5">Local proxy ports configured</p>
            </div>
          </div>

          <!-- 路由规则 -->
          <div class="flex items-start space-x-3 p-3 bg-gray-50 rounded-lg">
            <CheckCircleIcon class="w-5 h-5 text-green-600 flex-shrink-0 mt-0.5" />
            <div>
              <p class="text-sm font-medium text-gray-900">Routing Rules</p>
              <p class="text-xs text-gray-600 mt-0.5">Traffic routing configured</p>
            </div>
          </div>
        </div>
      </div>
    </Card>

    <!-- 下一步操作 -->
    <Card class="bg-blue-50 border-blue-200">
      <div class="space-y-4">
        <div class="flex items-start space-x-3">
          <RocketLaunchIcon class="w-6 h-6 text-blue-600 flex-shrink-0 mt-0.5" />
          <div>
            <h3 class="text-lg font-semibold text-blue-900 mb-2">Next Steps</h3>
            <ul class="space-y-2 text-sm text-blue-800">
              <li class="flex items-start">
                <span class="mr-2">1.</span>
                <span>Start the sing-box service from the dashboard</span>
              </li>
              <li class="flex items-start">
                <span class="mr-2">2.</span>
                <span>Configure your applications to use the proxy (e.g., 127.0.0.1:7890)</span>
              </li>
              <li class="flex items-start">
                <span class="mr-2">3.</span>
                <span>Monitor connections and manage nodes from the dashboard</span>
              </li>
              <li class="flex items-start">
                <span class="mr-2">4.</span>
                <span>Access web dashboard if Clash API is enabled (http://127.0.0.1:9090/ui)</span>
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
            <h3 class="text-lg font-semibold text-gray-900 mb-2">Quick Access</h3>
            <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
              <div class="p-3 bg-gray-50 rounded-lg">
                <p class="text-sm font-medium text-gray-900">HTTP/SOCKS5 Proxy</p>
                <code class="text-xs text-gray-600 font-mono">127.0.0.1:7890</code>
              </div>
              <div class="p-3 bg-gray-50 rounded-lg">
                <p class="text-sm font-medium text-gray-900">Clash API</p>
                <code class="text-xs text-gray-600 font-mono">127.0.0.1:9090</code>
              </div>
              <div class="p-3 bg-gray-50 rounded-lg">
                <p class="text-sm font-medium text-gray-900">Web Dashboard</p>
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
        {{ completing ? 'Completing...' : 'Complete Setup & Go to Dashboard' }}
      </Button>

      <Button
        variant="secondary"
        size="lg"
        :disabled="completing"
        @click="goToDashboard"
      >
        Skip to Dashboard
      </Button>
    </div>

    <!-- 提示信息 -->
    <div class="text-center text-sm text-gray-500">
      <p>You can always reconfigure these settings later from the dashboard.</p>
    </div>
  </div>
</template>
