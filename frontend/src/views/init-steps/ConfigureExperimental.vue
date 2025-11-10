<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { apiService } from '../../services/api'
import type { ClashAPI, CacheFile } from '../../types/api'
import { Button, Input, Select, Alert, Card, Loading } from '../../components'
import { InformationCircleIcon } from '@heroicons/vue/24/outline'

const emit = defineEmits<{
  next: []
  prev: []
}>()

const loading = ref(false)
const saving = ref(false)
const error = ref('')
const success = ref(false)

// Clash API 配置
const enableClashAPI = ref(false)
const clashAPIConfig = ref<ClashAPI>({
  external_controller: '127.0.0.1:9090',
  external_ui: '',
  secret: '',
  default_mode: 'rule',
})

// Cache File 配置
const enableCacheFile = ref(false)
const cacheFileConfig = ref<CacheFile>({
  enabled: false,
  path: '',
  cache_id: '',
  store_fakeip: false,
})

// 默认模式选项
const defaultModeOptions = [
  { label: 'Rule (Rules-based routing)', value: 'rule' },
  { label: 'Global (Route all traffic through proxy)', value: 'global' },
  { label: 'Direct (Direct connection)', value: 'direct' },
]

// 是否需要下载 Dashboard
const needsDashboard = computed(() => {
  return enableClashAPI.value && clashAPIConfig.value.external_ui
})

onMounted(async () => {
  await loadConfigs()
})

const loadConfigs = async () => {
  loading.value = true
  error.value = ''

  try {
    // 加载 Clash API 配置
    const clashAPI = await apiService.getClashAPI()
    if (clashAPI && clashAPI.external_controller) {
      enableClashAPI.value = true
      clashAPIConfig.value = {
        external_controller: clashAPI.external_controller || '127.0.0.1:9090',
        external_ui: clashAPI.external_ui || '',
        external_ui_download_url: clashAPI.external_ui_download_url,
        external_ui_download_detour: clashAPI.external_ui_download_detour,
        secret: clashAPI.secret || '',
        default_mode: clashAPI.default_mode || 'rule',
      }
    }

    // 加载 Cache File 配置
    const cacheFile = await apiService.getCacheFile()
    if (cacheFile && cacheFile.enabled) {
      enableCacheFile.value = true
      cacheFileConfig.value = {
        enabled: true,
        path: cacheFile.path || '',
        cache_id: cacheFile.cache_id || '',
        store_fakeip: cacheFile.store_fakeip || false,
        store_rdrc: cacheFile.store_rdrc || false,
        rdrc_timeout: cacheFile.rdrc_timeout || '',
      }
    }
  } catch (err: any) {
    console.log('No existing experimental config, using defaults')
  } finally {
    loading.value = false
  }
}

const saveConfigs = async () => {
  saving.value = true
  error.value = ''
  success.value = false

  try {
    // 保存 Clash API 配置
    if (enableClashAPI.value) {
      await apiService.updateClashAPI(clashAPIConfig.value)
    } else {
      // 如果禁用，发送空配置
      await apiService.updateClashAPI({})
    }

    // 保存 Cache File 配置
    if (enableCacheFile.value) {
      await apiService.updateCacheFile({
        ...cacheFileConfig.value,
        enabled: true,
      })
    } else {
      // 如果禁用，发送禁用配置
      await apiService.updateCacheFile({ enabled: false })
    }

    success.value = true

    // 2秒后自动进入下一步
    setTimeout(() => {
      emit('next')
    }, 2000)
  } catch (err: any) {
    error.value = err.response?.data?.error || err.message || 'Failed to save experimental configuration'
  } finally {
    saving.value = false
  }
}

const handleNext = () => {
  if (!success.value) {
    saveConfigs()
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
      <Loading size="lg" text="Loading experimental configuration..." />
    </div>

    <!-- 配置表单 -->
    <div v-else class="space-y-6">
      <!-- 成功提示 -->
      <Alert v-if="success" type="success" title="Configuration Saved">
        Experimental configuration has been saved successfully. Proceeding to next step...
      </Alert>

      <!-- 错误提示 -->
      <Alert v-if="error" type="error" closable @close="error = ''">
        {{ error }}
      </Alert>

      <!-- Clash API 配置 -->
      <Card>
        <div class="space-y-4">
          <div>
            <h3 class="text-lg font-semibold text-gray-900 mb-2">Clash API</h3>
            <p class="text-sm text-gray-600">
              Enable Clash-compatible RESTful API for external control and web dashboard.
            </p>
          </div>

          <!-- 启用 Clash API -->
          <div class="flex items-center space-x-3 p-4 bg-gray-50 rounded-lg">
            <input
              v-model="enableClashAPI"
              type="checkbox"
              id="enable-clash-api"
              class="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
            />
            <label for="enable-clash-api" class="flex-1 cursor-pointer">
              <span class="text-sm font-medium text-gray-900">Enable Clash API</span>
              <p class="text-xs text-gray-500 mt-0.5">
                Allows external control via HTTP API
              </p>
            </label>
          </div>

          <div v-if="enableClashAPI" class="space-y-4 pl-4 border-l-2 border-blue-200">
            <Input
              v-model="clashAPIConfig.external_controller"
              label="External Controller"
              placeholder="e.g., 127.0.0.1:9090"
              required
            />

            <Input
              v-model="clashAPIConfig.external_ui"
              label="External UI Path"
              placeholder="e.g., ./ui or /opt/clash/ui (optional)"
            />

            <Input
              v-model="clashAPIConfig.secret"
              type="password"
              label="Secret"
              placeholder="API access secret (optional)"
            />

            <Select
              v-model="clashAPIConfig.default_mode"
              :options="defaultModeOptions"
              label="Default Mode"
            />

            <!-- Dashboard 下载提示 -->
            <Alert v-if="needsDashboard" type="info" title="Dashboard Required">
              You've configured an External UI path. You'll be able to download the dashboard in the next step.
            </Alert>
          </div>
        </div>
      </Card>

      <!-- Cache File 配置 -->
      <Card>
        <div class="space-y-4">
          <div>
            <h3 class="text-lg font-semibold text-gray-900 mb-2">Cache File</h3>
            <p class="text-sm text-gray-600">
              Enable persistent cache to store routing decisions and fake-ip mappings.
            </p>
          </div>

          <!-- 启用 Cache File -->
          <div class="flex items-center space-x-3 p-4 bg-gray-50 rounded-lg">
            <input
              v-model="enableCacheFile"
              type="checkbox"
              id="enable-cache-file"
              class="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
            />
            <label for="enable-cache-file" class="flex-1 cursor-pointer">
              <span class="text-sm font-medium text-gray-900">Enable Cache File</span>
              <p class="text-xs text-gray-500 mt-0.5">
                Improves performance by caching routing results
              </p>
            </label>
          </div>

          <div v-if="enableCacheFile" class="space-y-4 pl-4 border-l-2 border-green-200">
            <Input
              v-model="cacheFileConfig.path"
              label="Cache File Path"
              placeholder="e.g., /var/lib/sing-box/cache.db (optional)"
            />

            <Input
              v-model="cacheFileConfig.cache_id"
              label="Cache ID"
              placeholder="Unique identifier for this cache (optional)"
            />

            <!-- Store FakeIP -->
            <div class="flex items-center space-x-3 p-3 bg-gray-50 rounded-lg">
              <input
                v-model="cacheFileConfig.store_fakeip"
                type="checkbox"
                id="store-fakeip"
                class="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
              />
              <label for="store-fakeip" class="flex-1 cursor-pointer">
                <span class="text-sm font-medium text-gray-900">Store Fake-IP mappings</span>
                <p class="text-xs text-gray-500 mt-0.5">
                  Persist fake-ip to real-ip mappings
                </p>
              </label>
            </div>
          </div>
        </div>
      </Card>

      <!-- 操作按钮 -->
      <div class="flex gap-3">
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

      <!-- 说明信息 -->
      <Card padding="sm" class="bg-blue-50 border-blue-200">
        <div class="flex items-start space-x-3">
          <InformationCircleIcon class="h-5 w-5 text-blue-600 flex-shrink-0 mt-0.5" />
          <div class="text-sm text-blue-900 space-y-2">
            <p class="font-medium">About Experimental Features:</p>
            <ul class="list-disc list-inside space-y-1 ml-2 text-blue-800">
              <li><strong>Clash API:</strong> Enables web-based dashboard and third-party app control</li>
              <li><strong>External Controller:</strong> The HTTP API endpoint (host:port)</li>
              <li><strong>External UI:</strong> Path to web dashboard files (e.g., yacd, metacubexd)</li>
              <li><strong>Cache File:</strong> Speeds up DNS resolution and routing decisions</li>
            </ul>
            <p class="mt-2 text-xs text-blue-700">
              💡 Tip: Enable Clash API if you want to use a web dashboard to manage sing-box.
            </p>
          </div>
        </div>
      </Card>
    </div>
  </div>
</template>
