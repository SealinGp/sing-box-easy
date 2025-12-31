<script setup lang="ts">
import { ref, watch } from 'vue'
import type { CacheFile } from '../types/api'
import Button from './Button.vue'
import Card from './Card.vue'
import Input from './Input.vue'

const props = defineProps<{
  cacheFile: CacheFile | null
  loading: boolean
}>()

const emit = defineEmits<{
  update: [cacheFile: CacheFile]
}>()

const settings = ref<CacheFile>({
  enabled: false,
  path: '',
  cache_id: '',
  store_fakeip: false,
  store_rdrc: false,
  rdrc_timeout: '7d',
})

// Watch for props changes
watch(() => props.cacheFile, (newConfig) => {
  if (newConfig) {
    settings.value = { ...newConfig }
  } else {
    // Reset to defaults when null
    settings.value = {
      enabled: false,
      path: '',
      cache_id: '',
      store_fakeip: false,
      store_rdrc: false,
      rdrc_timeout: '7d',
    }
  }
}, { immediate: true, deep: true })

const handleSave = () => {
  emit('update', settings.value)
}
</script>

<template>
  <div>
    <Card>
      <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-6">Cache File Settings</h3>

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
              Enable Cache File
            </label>
            <p class="text-sm text-gray-500 dark:text-gray-400">
              Store cache data to improve performance
            </p>
          </div>
        </div>

        <!-- Cache Path -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            Cache Path
          </label>
          <Input
            v-model="settings.path"
            placeholder="/etc/sing-box/cache.db"
            :disabled="!settings.enabled"
          />
          <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
            Path to the cache file
          </p>
        </div>

        <!-- Cache ID -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            Cache ID
          </label>
          <Input
            v-model="settings.cache_id"
            placeholder="unique-cache-id"
            :disabled="!settings.enabled"
          />
          <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
            Unique identifier for cache entries
          </p>
        </div>

        <!-- Advanced Settings -->
        <div class="border-t border-gray-200 dark:border-gray-700 pt-6">
          <h4 class="text-sm font-medium text-gray-900 dark:text-gray-100 mb-4">Advanced Settings</h4>

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
                  Store FakeIP
                </label>
                <p class="text-sm text-gray-500 dark:text-gray-400">
                  Cache FakeIP mappings
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
                  Store RDRC
                </label>
                <p class="text-sm text-gray-500 dark:text-gray-400">
                  Cache rule-based DNS routing
                </p>
              </div>
            </div>

            <!-- RDRC Timeout -->
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                RDRC Timeout
              </label>
              <Input
                v-model="settings.rdrc_timeout"
                placeholder="7d"
                :disabled="!settings.enabled || !settings.store_rdrc"
              />
              <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
                How long to cache RDRC entries (e.g., 7d, 24h, 30m)
              </p>
            </div>
          </div>
        </div>

        <!-- Save Button -->
        <div class="flex justify-end pt-4 border-t border-gray-200 dark:border-gray-700">
          <Button @click="handleSave" variant="primary" :disabled="loading">
            Save Settings
          </Button>
        </div>
      </div>
    </Card>
  </div>
</template>