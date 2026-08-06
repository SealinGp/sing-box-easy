<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ClashAPI, CacheFile } from '../../types/api'
import { Button, Input, Alert, Card, Loading } from '../../components'
import { Select } from '../../volt'
import { InformationCircleIcon } from '@heroicons/vue/24/outline'
import { experimentalService } from '../../services'
import {
  CUSTOM_DASHBOARD_ID,
  DASHBOARD_OPTIONS,
  dashboardIdForUrl,
} from '../../constants/dashboards'

const { t } = useI18n()

const emit = defineEmits<{
  next: []
  prev: []
}>()

const loading = ref(false)
const saving = ref(false)
const error = ref('')
const success = ref(false)
const loadFailed = ref(false)

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
const defaultModeOptions = computed(() => [
  { label: t('init.experimental.modeOptions.rule'), value: 'rule' },
  { label: t('init.experimental.modeOptions.global'), value: 'global' },
  { label: t('init.experimental.modeOptions.direct'), value: 'direct' },
])

// Dashboard 预设选项来自共享列表（src/constants/dashboards.ts），
// 与 Clash API 设置页的下拉选择器保持一致。
const dashboardOptions = DASHBOARD_OPTIONS

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
  if (selectedDashboard.value === CUSTOM_DASHBOARD_ID && newValue) {
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
  loadFailed.value = false

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
      const matchedId = dashboardIdForUrl(clashAPI.external_ui_download_url)
      if (matchedId) {
        selectedDashboard.value = matchedId
        if (matchedId === CUSTOM_DASHBOARD_ID) {
          customDownloadUrl.value = clashAPI.external_ui_download_url ?? ''
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
    // Surface the error so the user knows the loaded values may not reflect
    // the existing configuration.  Do NOT silently fall back to defaults —
    // that is precisely what causes the 0.0.0.0 / port mismatch bug where
    // the user only corrects the port but leaves the host at its default.
    loadFailed.value = true
    error.value = t('init.experimental.loadFailed')
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
    error.value = err.message || t('init.experimental.saveFailed')
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
      <Loading size="lg" :text="$t('init.experimental.loading')" />
    </div>

    <!-- 配置表单 -->
    <div v-else class="space-y-6">
      <!-- 成功提示 -->
      <Alert v-if="success" type="success" :title="$t('init.experimental.savedTitle')">
        {{ $t('init.experimental.savedDesc') }}
      </Alert>

      <!-- 配置加载失败提示 — 阻止保存以防止意外覆盖现有配置 -->
      <Alert v-if="loadFailed" type="error" :title="$t('init.experimental.loadFailedTitle')">
        {{ $t('init.experimental.loadFailedDesc') }}
        <button
          class="mt-2 text-sm underline hover:no-underline"
          @click="loadConfigs"
        >
          {{ $t('init.experimental.retryLoad') }}
        </button>
      </Alert>

      <!-- 其他错误提示 -->
      <Alert v-else-if="error" type="error" closable @close="error = ''">
        {{ error }}
      </Alert>

      <!-- Clash API 配置 -->
      <Card>
        <div class="space-y-4">
          <div>
            <h3 class="text-lg font-semibold text-gray-900 mb-2">{{ $t('init.experimental.clashHeading') }}</h3>
            <p class="text-sm text-gray-600">
              {{ $t('init.experimental.clashIntro') }}
            </p>
          </div>

          <!-- 启用 Clash API -->
          <div class="flex items-center space-x-3 p-4 bg-gray-50 rounded-surface">
            <input
              v-model="enableClashAPI"
              type="checkbox"
              id="enable-clash-api"
              class="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500"
            />
            <label for="enable-clash-api" class="flex-1 cursor-pointer">
              <span class="text-sm font-medium text-gray-900">{{ $t('init.experimental.enableClashLabel') }}</span>
              <p class="text-xs text-gray-500 mt-0.5">
                {{ $t('init.experimental.enableClashDesc') }}
              </p>
            </label>
          </div>

          <div v-if="enableClashAPI" class="space-y-4 pl-4 border-l-2 border-primary-200">
            <Input
              v-model="clashAPIConfig.external_controller"
              :label="$t('init.experimental.controllerLabel')"
              :placeholder="$t('init.experimental.controllerPlaceholder')"
              required
            />

            <Input
              v-model="clashAPIConfig.external_ui"
              :label="$t('init.experimental.externalUILabel')"
              :placeholder="$t('init.experimental.externalUIPlaceholder')"
            />

            <!-- Dashboard 选择器 -->
            <div v-if="clashAPIConfig.external_ui" class="space-y-3">
              <label class="block text-sm font-medium text-gray-700">
                {{ $t('init.experimental.downloadUrlLabel') }}
              </label>

              <!-- 当前选择的 URL 显示 -->
              <div v-if="clashAPIConfig.external_ui_download_url" class="p-3 bg-green-50 border border-green-200 rounded-control">
                <div class="flex items-start space-x-2">
                  <svg class="w-5 h-5 text-green-600 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
                    <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
                  </svg>
                  <div class="flex-1 min-w-0">
                    <p class="text-xs font-medium text-green-900">{{ $t('init.experimental.selectedUrlLabel') }}</p>
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
                    'relative border-2 rounded-control p-3 cursor-pointer transition-all',
                    selectedDashboard === dashboard.id
                      ? 'border-primary-500 bg-primary-50'
                      : 'border-gray-200 hover:border-primary-300 hover:bg-gray-50'
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
                        class="text-sm font-semibold text-primary-600 hover:text-primary-800 hover:underline"
                        @click.stop
                      >
                        {{ dashboard.name }}
                      </a>
                      <p class="text-xs text-gray-500 mt-0.5">{{ $t(dashboard.descKey) }}</p>
                    </div>
                    <!-- 选中标记 -->
                    <div
                      v-if="selectedDashboard === dashboard.id"
                      class="absolute top-2 right-2 w-5 h-5 bg-primary-500 rounded-pill flex items-center justify-center"
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
                @click="selectedDashboard = CUSTOM_DASHBOARD_ID"
                :class="[
                  'border-2 rounded-control p-4 cursor-pointer transition-all',
                  selectedDashboard === CUSTOM_DASHBOARD_ID
                    ? 'border-primary-500 bg-primary-50'
                    : 'border-gray-200 hover:border-primary-300 hover:bg-gray-50'
                ]"
              >
                <div class="flex items-center justify-between mb-3">
                  <span class="text-sm font-semibold text-gray-900">{{ $t('init.experimental.customLabel') }}</span>
                  <div
                    v-if="selectedDashboard === CUSTOM_DASHBOARD_ID"
                    class="w-5 h-5 bg-primary-500 rounded-pill flex items-center justify-center"
                  >
                    <svg class="w-3 h-3 text-white" fill="currentColor" viewBox="0 0 20 20">
                      <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
                    </svg>
                  </div>
                </div>
                <Input
                  v-model="customDownloadUrl"
                  :placeholder="$t('init.experimental.customPlaceholder')"
                  @click.stop
                  @focus="selectedDashboard = CUSTOM_DASHBOARD_ID"
                />
              </div>
            </div>

            <Input
              v-model="clashAPIConfig.secret"
              type="password"
              :label="$t('init.experimental.secretLabel')"
              :placeholder="$t('init.experimental.secretPlaceholder')"
            />

            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('init.experimental.modeLabel') }}</label>
            <Select
              class="w-full"
              v-model="clashAPIConfig.default_mode"
              :options="defaultModeOptions"
              optionLabel="label"
              optionValue="value"
            />

            <!-- Dashboard 下载提示 -->
            <Alert v-if="needsDashboard" type="info" :title="$t('init.experimental.dashboardRequiredTitle')">
              {{ $t('init.experimental.dashboardRequiredDesc') }}
            </Alert>
          </div>
        </div>
      </Card>

      <!-- Cache File 配置 -->
      <Card>
        <div class="space-y-4">
          <div>
            <h3 class="text-lg font-semibold text-gray-900 mb-2">{{ $t('init.experimental.cacheHeading') }}</h3>
            <p class="text-sm text-gray-600">
              {{ $t('init.experimental.cacheIntro') }}
            </p>
          </div>

          <!-- 启用 Cache File -->
          <div class="flex items-center space-x-3 p-4 bg-gray-50 rounded-surface">
            <input
              v-model="enableCacheFile"
              type="checkbox"
              id="enable-cache-file"
              class="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500"
            />
            <label for="enable-cache-file" class="flex-1 cursor-pointer">
              <span class="text-sm font-medium text-gray-900">{{ $t('init.experimental.enableCacheLabel') }}</span>
              <p class="text-xs text-gray-500 mt-0.5">
                {{ $t('init.experimental.enableCacheDesc') }}
              </p>
            </label>
          </div>

          <div v-if="enableCacheFile" class="space-y-4 pl-4 border-l-2 border-green-200">
            <Input
              v-model="cacheFileConfig.path"
              :label="$t('init.experimental.cachePathLabel')"
              :placeholder="$t('init.experimental.cachePathPlaceholder')"
            />

            <Input
              v-model="cacheFileConfig.cache_id"
              :label="$t('init.experimental.cacheIdLabel')"
              :placeholder="$t('init.experimental.cacheIdPlaceholder')"
            />

            <!-- Store FakeIP -->
            <div class="flex items-center space-x-3 p-3 bg-gray-50 rounded-control">
              <input
                v-model="cacheFileConfig.store_fakeip"
                type="checkbox"
                id="store-fakeip"
                class="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500"
              />
              <label for="store-fakeip" class="flex-1 cursor-pointer">
                <span class="text-sm font-medium text-gray-900">{{ $t('init.experimental.storeFakeipLabel') }}</span>
                <p class="text-xs text-gray-500 mt-0.5">
                  {{ $t('init.experimental.storeFakeipDesc') }}
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
          :disabled="saving || success || loadFailed"
          @click="handleNext"
        >
          {{ success ? $t('init.experimental.savedBtn') : saving ? $t('common.saving') : $t('init.experimental.saveContinueBtn') }}
        </Button>

        <Button
          variant="ghost"
          :disabled="saving || success"
          @click="handleSkip"
        >
          {{ $t('init.experimental.skipBtn') }}
        </Button>
      </div>

      <!-- 说明信息 -->
      <Card padding="sm" class="bg-primary-50 border-primary-200">
        <div class="flex items-start space-x-3">
          <InformationCircleIcon class="h-5 w-5 text-primary-600 flex-shrink-0 mt-0.5" />
          <div class="text-sm text-primary-900 space-y-2">
            <p class="font-medium">{{ $t('init.experimental.aboutTitle') }}</p>
            <ul class="list-disc list-inside space-y-1 ml-2 text-primary-800">
              <li><strong>Clash API:</strong> {{ $t('init.experimental.about.clash') }}</li>
              <li><strong>External Controller:</strong> {{ $t('init.experimental.about.controller') }}</li>
              <li><strong>External UI:</strong> {{ $t('init.experimental.about.externalUI') }}</li>
              <li><strong>Cache File:</strong> {{ $t('init.experimental.about.cache') }}</li>
            </ul>
            <p class="mt-2 text-xs text-primary-700">
              💡 {{ $t('init.experimental.aboutTip') }}
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
            :alt="$t('init.experimental.previewAlt')"
            class="max-w-full max-h-[90vh] object-contain rounded-surface shadow-float"
            @click.stop
          />
        </div>
      </div>
    </Teleport>
  </div>
</template>
