<script setup lang="ts">
import { ref, onMounted } from 'vue'
import type { CacheFile } from '../types/api'
import Button from './Button.vue'
import Card from './Card.vue'
import Input from './Input.vue'
import { experimentalService } from '../services'
import { useToast } from 'primevue'
import { useI18n } from 'vue-i18n'

const toast = useToast()
const { t } = useI18n()

// Local state
const loading = ref(false)
const settings = ref<CacheFile>({
  enabled: false,
  path: '',
  cache_id: '',
  store_fakeip: false,
  store_rdrc: false,
  rdrc_timeout: '7d',
})

// Fetch cache file config
const fetchCacheFile = async () => {
  loading.value = true
  try {
    const { data } = await experimentalService.getCacheFile()
    if (data) {
      settings.value = { ...data }
    }
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: err.message || t('experimental.cache.toast.fetchFailed'),
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

// Update cache file config
const handleSave = async () => {
  loading.value = true
  try {
    await experimentalService.updateCacheFile(settings.value)
    toast.add({
      severity: 'success',
      summary: t('common.success'),
      detail: t('experimental.cache.toast.updatedOk'),
      life: 3000
    })
    await fetchCacheFile()
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: err.message || t('experimental.cache.toast.updateFailed'),
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

// Load data on mount
onMounted(() => {
  fetchCacheFile()
})
</script>

<template>
  <div>
    <Card>
      <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-6">{{ $t('experimental.cache.title') }}</h3>

      <div v-if="loading" class="flex items-center justify-center py-12">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-violet-600"></div>
      </div>

      <div v-else class="space-y-6">
        <!-- Enable Cache File -->
        <div class="flex items-start">
          <div class="flex items-center h-5">
            <input
              type="checkbox"
              id="cache_enabled"
              v-model="settings.enabled"
              class="w-4 h-4 text-violet-600 bg-gray-100 border-gray-300 rounded focus:ring-violet-500 dark:focus:ring-violet-600 dark:ring-offset-gray-800 focus:ring-2 dark:bg-gray-700 dark:border-gray-600"
            />
          </div>
          <div class="ml-3">
            <label for="cache_enabled" class="text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ $t('experimental.cache.enable') }}
            </label>
            <p class="text-sm text-gray-500 dark:text-gray-400">
              {{ $t('experimental.cache.enableHelp') }}
            </p>
          </div>
        </div>

        <!-- Cache Path -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            {{ $t('experimental.cache.path') }}
          </label>
          <Input
            v-model="settings.path"
            placeholder="/etc/sing-box/cache.db"
            :disabled="!settings.enabled"
          />
          <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
            {{ $t('experimental.cache.pathHelp') }}
          </p>
        </div>

        <!-- Cache ID -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            {{ $t('experimental.cache.cacheId') }}
          </label>
          <Input
            v-model="settings.cache_id"
            :placeholder="$t('experimental.cache.cacheIdPlaceholder')"
            :disabled="!settings.enabled"
          />
          <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
            {{ $t('experimental.cache.cacheIdHelp') }}
          </p>
        </div>

        <!-- Advanced Settings -->
        <div class="border-t border-gray-200 dark:border-gray-700 pt-6">
          <h4 class="text-sm font-medium text-gray-900 dark:text-gray-100 mb-4">{{ $t('experimental.cache.advanced.title') }}</h4>

          <div class="space-y-4">
            <!-- Store FakeIP -->
            <div class="flex items-start">
              <div class="flex items-center h-5">
                <input
                  type="checkbox"
                  id="store_fakeip"
                  v-model="settings.store_fakeip"
                  :disabled="!settings.enabled"
                  class="w-4 h-4 text-violet-600 bg-gray-100 border-gray-300 rounded focus:ring-violet-500 dark:focus:ring-violet-600 dark:ring-offset-gray-800 focus:ring-2 dark:bg-gray-700 dark:border-gray-600"
                />
              </div>
              <div class="ml-3">
                <label for="store_fakeip" class="text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ $t('experimental.cache.advanced.storeFakeip') }}
                </label>
                <p class="text-sm text-gray-500 dark:text-gray-400">
                  {{ $t('experimental.cache.advanced.storeFakeipHelp') }}
                </p>
              </div>
            </div>

            <!-- Store RDRC -->
            <div class="flex items-start">
              <div class="flex items-center h-5">
                <input
                  type="checkbox"
                  id="store_rdrc"
                  v-model="settings.store_rdrc"
                  :disabled="!settings.enabled"
                  class="w-4 h-4 text-violet-600 bg-gray-100 border-gray-300 rounded focus:ring-violet-500 dark:focus:ring-violet-600 dark:ring-offset-gray-800 focus:ring-2 dark:bg-gray-700 dark:border-gray-600"
                />
              </div>
              <div class="ml-3">
                <label for="store_rdrc" class="text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ $t('experimental.cache.advanced.storeRdrc') }}
                </label>
                <p class="text-sm text-gray-500 dark:text-gray-400">
                  {{ $t('experimental.cache.advanced.storeRdrcHelp') }}
                </p>
              </div>
            </div>

            <!-- RDRC Timeout -->
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                {{ $t('experimental.cache.advanced.rdrcTimeout') }}
              </label>
              <Input
                v-model="settings.rdrc_timeout"
                placeholder="7d"
                :disabled="!settings.enabled || !settings.store_rdrc"
              />
              <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
                {{ $t('experimental.cache.advanced.rdrcTimeoutHelp') }}
              </p>
            </div>
          </div>
        </div>

        <!-- Save Button -->
        <div class="flex justify-end pt-4 border-t border-gray-200 dark:border-gray-700">
          <Button @click="handleSave" variant="primary" :disabled="loading">
            {{ $t('experimental.cache.save') }}
          </Button>
        </div>
      </div>
    </Card>
  </div>
</template>