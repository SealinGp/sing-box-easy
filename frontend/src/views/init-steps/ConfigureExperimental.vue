<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import type { ClashAPI, CacheFile } from '../../types/api'
import { Button, Input, Select, Alert, Card, Loading } from '../../components'
import { InformationCircleIcon } from '@heroicons/vue/24/outline'
import zashboardIcon from '../../assets/zashboard.svg'
import yacdIcon from '../../assets/yacd.ico'
import { experimentalService } from '../../services'

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
  external_controller: '0.0.0.0:9090',
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

// Dashboard 预设选项
const dashboardOptions = [
  {
    id: 'zashboard',
    name: 'Zashboard',
    url: 'https://github.com/Zephyruso/zashboard/archive/gh-pages.zip',
    link: 'https://github.com/Zephyruso/zashboard',
    icon: zashboardIcon,
    preview: 'https://raw.githubusercontent.com/Zephyruso/zashboard/refs/heads/main/readme/pc.png',
    description: 'Modern dashboard with clean UI',
  },
  {
    id: 'yacd',
    name: 'Yacd',
    url: 'https://github.com/MetaCubeX/Yacd-meta/archive/gh-pages.zip',
    link: 'https://github.com/haishanh/yacd',
    icon: yacdIcon,
    preview: 'https://user-images.githubusercontent.com/1166872/47954055-97e6cb80-dfc0-11e8-991f-230fd40481e5.png',
    description: 'Yet another Clash dashboard',
  },
]

// 选中的 dashboard
const selectedDashboard = ref('')
const customDownloadUrl = ref('')
const showPreview = ref(false)
const previewImage = ref('')

// 是否需要下载 Dashboard
const needsDashboard = computed(() => {
  return enableClashAPI.value && clashAPIConfig.value.external_ui
})

// 打开预览图
const openPreview = (imageUrl: string) => {
  previewImage.value = imageUrl
  showPreview.value = true
}

// 关闭预览图
const closePreview = () => {
  showPreview.value = false
  previewImage.value = ''
}

// 选择 dashboard
const selectDashboard = (dashboardId: string) => {
  selectedDashboard.value = dashboardId
  const dashboard = dashboardOptions.find(d => d.id === dashboardId)
  if (dashboard) {
    clashAPIConfig.value.external_ui_download_url = dashboard.url
    customDownloadUrl.value = ''
  }
}

// 使用自定义 URL - removed as not currently used
// const useCustomUrl = () => {
//   selectedDashboard.value = 'custom'
//   clashAPIConfig.value.external_ui_download_url = customDownloadUrl.value
// }

// 监听自定义 URL 变化
watch(customDownloadUrl, (newValue) => {
  if (selectedDashboard.value === 'custom' && newValue) {
    clashAPIConfig.value.external_ui_download_url = newValue
  }
})

// 监听 External UI Path 变化，清空时重置 dashboard 选择
watch(() => clashAPIConfig.value.external_ui, (newValue) => {
  if (!newValue) {
    selectedDashboard.value = ''
    customDownloadUrl.value = ''
    clashAPIConfig.value.external_ui_download_url = undefined
  }
})

onMounted(async () => {
  await loadConfigs()
})

const loadConfigs = async () => {
  loading.value = true
  error.value = ''

  try {
    // 加载 Clash API 配置    
    const resp = await experimentalService.getClashAPI()
    const clashAPI = resp.data
    if (clashAPI && clashAPI.external_controller) {
      enableClashAPI.value = true
      clashAPIConfig.value = {
        external_controller: clashAPI.external_controller || '127.0.0.1:9090',
        external_ui: clashAPI.external_ui || '/etc/sing-box/ui',
        external_ui_download_url: clashAPI.external_ui_download_url,
        external_ui_download_detour: clashAPI.external_ui_download_detour,
        secret: clashAPI.secret || '',
        default_mode: clashAPI.default_mode || 'rule',
      }

      // 检测是否匹配预设的 dashboard
      if (clashAPI.external_ui_download_url) {
        const matchedDashboard = dashboardOptions.find(d => d.url === clashAPI.external_ui_download_url)
        if (matchedDashboard) {
          selectedDashboard.value = matchedDashboard.id
        } else {
          selectedDashboard.value = 'custom'
          customDownloadUrl.value = clashAPI.external_ui_download_url
        }
      }
    }

    // 加载 Cache File 配置
    const respCache = await experimentalService.getCacheFile()
    const cacheFile = respCache.data
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
      await experimentalService.updateClashAPI(clashAPIConfig.value)
    } else {
      // 如果禁用，发送空配置
      await experimentalService.updateClashAPI({})
    }

    // 保存 Cache File 配置
    if (enableCacheFile.value) {
      await experimentalService.updateCacheFile({
        ...cacheFileConfig.value,
        enabled: true,
      })
    } else {
      // 如果禁用，发送禁用配置
      await experimentalService.updateCacheFile({ enabled: false })
    }

    success.value = true

    // 2秒后自动进入下一步
    setTimeout(() => {
      emit('next')
    }, 2000)
  } catch (err: any) {
    error.value = err.message || 'Failed to save experimental configuration'
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
              class="w-4 h-4 text-violet-600 border-gray-300 rounded focus:ring-violet-500"
            />
            <label for="enable-clash-api" class="flex-1 cursor-pointer">
              <span class="text-sm font-medium text-gray-900">Enable Clash API</span>
              <p class="text-xs text-gray-500 mt-0.5">
                Allows external control via HTTP API
              </p>
            </label>
          </div>

          <div v-if="enableClashAPI" class="space-y-4 pl-4 border-l-2 border-violet-200">
            <Input
              v-model="clashAPIConfig.external_controller"
              label="External Controller"
              placeholder="e.g., 127.0.0.1:9090"
              required
            />

            <Input
              v-model="clashAPIConfig.external_ui"
              label="External UI Path"
              placeholder="e.g., /etc/sing-box/ui(default) or ./ui (optional)"
            />

            <!-- Dashboard 选择器 -->
            <div v-if="clashAPIConfig.external_ui" class="space-y-3">
              <label class="block text-sm font-medium text-gray-700">
                Dashboard Download URL (Optional)
              </label>

              <!-- 当前选择的 URL 显示 -->
              <div v-if="clashAPIConfig.external_ui_download_url" class="p-3 bg-green-50 border border-green-200 rounded-lg">
                <div class="flex items-start space-x-2">
                  <svg class="w-5 h-5 text-green-600 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
                    <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
                  </svg>
                  <div class="flex-1 min-w-0">
                    <p class="text-xs font-medium text-green-900">Selected Download URL:</p>
                    <p class="text-xs text-green-700 break-all mt-1 font-mono">{{ clashAPIConfig.external_ui_download_url }}</p>
                  </div>
                </div>
              </div>

              <!-- 预设 Dashboard 选项 -->
              <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
                <div
                  v-for="dashboard in dashboardOptions"
                  :key="dashboard.id"
                  @click="selectDashboard(dashboard.id)"
                  :class="[
                    'relative border-2 rounded-lg p-3 cursor-pointer transition-all',
                    selectedDashboard === dashboard.id
                      ? 'border-violet-500 bg-violet-50'
                      : 'border-gray-200 hover:border-violet-300 hover:bg-gray-50'
                  ]"
                >
                  <div class="flex items-start space-x-3">
                    <!-- 图标 -->
                    <img
                      v-if="dashboard.icon"
                      :src="dashboard.icon"
                      :alt="dashboard.name"
                      class="w-10 h-10 rounded flex-shrink-0"
                      @error="(e) => (e.target as HTMLImageElement).style.display = 'none'"
                    />
                    <div class="flex-1 min-w-0">
                      <!-- 名称（带链接） -->
                      <a
                        :href="dashboard.link"
                        target="_blank"
                        rel="noopener noreferrer"
                        class="text-sm font-semibold text-violet-600 hover:text-violet-800 hover:underline"
                        @click.stop
                      >
                        {{ dashboard.name }}
                      </a>
                      <p class="text-xs text-gray-500 mt-0.5">{{ dashboard.description }}</p>
                    </div>
                    <!-- 选中标记 -->
                    <div
                      v-if="selectedDashboard === dashboard.id"
                      class="absolute top-2 right-2 w-5 h-5 bg-violet-500 rounded-full flex items-center justify-center"
                    >
                      <svg class="w-3 h-3 text-white" fill="currentColor" viewBox="0 0 20 20">
                        <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
                      </svg>
                    </div>
                  </div>
                  <!-- 预览图 -->
                  <div v-if="dashboard.preview" class="mt-3">
                    <img
                      :src="dashboard.preview"
                      :alt="`${dashboard.name} preview`"
                      class="w-full h-32 object-cover rounded cursor-pointer hover:opacity-75 transition-opacity"
                      @click.stop="openPreview(dashboard.preview)"
                      @error="(e) => (e.target as HTMLImageElement).style.display = 'none'"
                    />
                  </div>
                </div>
              </div>

              <!-- 自定义 URL 选项 -->
              <div
                @click="selectedDashboard = 'custom'"
                :class="[
                  'border-2 rounded-lg p-4 cursor-pointer transition-all',
                  selectedDashboard === 'custom'
                    ? 'border-violet-500 bg-violet-50'
                    : 'border-gray-200 hover:border-violet-300 hover:bg-gray-50'
                ]"
              >
                <div class="flex items-center justify-between mb-3">
                  <span class="text-sm font-semibold text-gray-900">Custom Download URL</span>
                  <div
                    v-if="selectedDashboard === 'custom'"
                    class="w-5 h-5 bg-violet-500 rounded-full flex items-center justify-center"
                  >
                    <svg class="w-3 h-3 text-white" fill="currentColor" viewBox="0 0 20 20">
                      <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
                    </svg>
                  </div>
                </div>
                <Input
                  v-model="customDownloadUrl"
                  placeholder="https://github.com/user/repo/archive/gh-pages.zip"
                  @click.stop
                  @focus="selectedDashboard = 'custom'"
                />
              </div>
            </div>

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
              class="w-4 h-4 text-violet-600 border-gray-300 rounded focus:ring-violet-500"
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
                class="w-4 h-4 text-violet-600 border-gray-300 rounded focus:ring-violet-500"
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
      <Card padding="sm" class="bg-violet-50 border-violet-200">
        <div class="flex items-start space-x-3">
          <InformationCircleIcon class="h-5 w-5 text-violet-600 flex-shrink-0 mt-0.5" />
          <div class="text-sm text-violet-900 space-y-2">
            <p class="font-medium">About Experimental Features:</p>
            <ul class="list-disc list-inside space-y-1 ml-2 text-violet-800">
              <li><strong>Clash API:</strong> Enables web-based dashboard and third-party app control</li>
              <li><strong>External Controller:</strong> The HTTP API endpoint (host:port)</li>
              <li><strong>External UI:</strong> Path to web dashboard files (e.g., yacd, metacubexd)</li>
              <li><strong>Cache File:</strong> Speeds up DNS resolution and routing decisions</li>
            </ul>
            <p class="mt-2 text-xs text-violet-700">
              💡 Tip: Enable Clash API if you want to use a web dashboard to manage sing-box.
            </p>
          </div>
        </div>
      </Card>
    </div>

    <!-- 图片预览 Modal -->
    <Teleport to="body">
      <div
        v-if="showPreview"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black p-4"
        @click="closePreview"
      >
        <div class="relative max-w-6xl max-h-full">
          <!-- 关闭按钮 -->
          <button
            @click.stop="closePreview"
            class="absolute -top-10 right-0 text-white hover:text-gray-300 transition-colors"
          >
            <svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
          <!-- 预览图片 -->
          <img
            :src="previewImage"
            alt="Dashboard preview"
            class="max-w-full max-h-[90vh] object-contain rounded-lg shadow-2xl"
            @click.stop
          />
        </div>
      </div>
    </Teleport>
  </div>
</template>
