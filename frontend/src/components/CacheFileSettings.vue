<script setup lang="ts">
import FieldHelp from './FieldHelp.vue'
import Reveal from './Reveal.vue'
import SaveFeedback from './SaveFeedback.vue'
import { useInlineSave } from '../composables/useInlineSave'
import { watch, ref, onMounted } from 'vue'
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
const loadError = ref('')
const { saving, saveError, savedMessage, clearSaveFeedback, persist } = useInlineSave()
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
  loadError.value = ''
  try {
    const { data } = await experimentalService.getCacheFile()
    if (data) {
      settings.value = { ...data }
    }
  } catch (err: any) {
    loadError.value = err.message || t('experimental.cache.toast.fetchFailed')
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
  if (saving.value || loading.value || loadError.value) return
  await persist(() => experimentalService.updateCacheFile({ ...settings.value }),
    t('experimental.cache.toast.updatedOk'), t('experimental.cache.toast.updateFailed'))
}

watch([settings], clearSaveFeedback, { deep: true })

// Load data on mount
onMounted(() => {
  fetchCacheFile()
})
</script>

<template>
  <div>
    <Card>
      <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">{{ $t('experimental.cache.title') }}</h3>

      <div v-if="loading" class="flex items-center justify-center py-12">
        <div class="animate-spin rounded-pill h-8 w-8 border-b-2 border-primary-600"></div>
      </div>

      <div v-else-if="loadError" class="space-y-3">
        <p role="alert" class="text-sm text-danger">{{ loadError }}</p>
        <Button variant="secondary" @click="fetchCacheFile">{{ $t('common.refresh') }}</Button>
      </div>

      <fieldset v-else :disabled="saving" @input="clearSaveFeedback" @change="clearSaveFeedback" class="min-w-0 space-y-4">
        <!-- Enable Cache File -->
        <div class="flex items-start">
          <div class="flex items-center h-5">
            <input
              type="checkbox"
              id="cache_enabled"
              v-model="settings.enabled"
              class="w-4 h-4 text-primary-600 bg-gray-100 border-gray-300 rounded focus:ring-primary-500 dark:focus:ring-primary-600 dark:ring-offset-gray-800 focus:ring-2 dark:bg-gray-700 dark:border-gray-600"
            />
          </div>
          <div class="ml-3">
            <div class="flex items-center gap-1"><label for="cache_enabled" class="text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ $t('experimental.cache.enable') }}
            </label><FieldHelp :label="$t('experimental.cache.enable')" :text="$t('experimental.cache.enableHelp')" /></div>

          </div>
        </div>

        <Reveal :open="!!settings.enabled"><div class="space-y-4">
        <!-- Cache Path -->
        <div>
          <div class="flex items-center gap-1"><label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            {{ $t('experimental.cache.path') }}
          </label><FieldHelp :label="$t('experimental.cache.path')" :text="$t('experimental.cache.pathHelp')" /></div>
          <Input
            :aria-label="$t('experimental.cache.path')"
            v-model="settings.path"
            placeholder="/etc/sing-box/cache.db"
            :disabled="!settings.enabled"
          />

        </div>

        <!-- Cache ID -->
        <div>
          <div class="flex items-center gap-1"><label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            {{ $t('experimental.cache.cacheId') }}
          </label><FieldHelp :label="$t('experimental.cache.cacheId')" :text="$t('experimental.cache.cacheIdHelp')" /></div>
          <Input
            :aria-label="$t('experimental.cache.cacheId')"
            v-model="settings.cache_id"
            :placeholder="$t('experimental.cache.cacheIdPlaceholder')"
            :disabled="!settings.enabled"
          />

        </div>

        <!-- Advanced Settings -->
        <div class="border-t border-gray-200 dark:border-gray-700 pt-4">
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
                  class="w-4 h-4 text-primary-600 bg-gray-100 border-gray-300 rounded focus:ring-primary-500 dark:focus:ring-primary-600 dark:ring-offset-gray-800 focus:ring-2 dark:bg-gray-700 dark:border-gray-600"
                />
              </div>
              <div class="ml-3">
                <div class="flex items-center gap-1"><label for="store_fakeip" class="text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ $t('experimental.cache.advanced.storeFakeip') }}
                </label><FieldHelp :label="$t('experimental.cache.advanced.storeFakeip')" :text="$t('experimental.cache.advanced.storeFakeipHelp')" /></div>

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
                  class="w-4 h-4 text-primary-600 bg-gray-100 border-gray-300 rounded focus:ring-primary-500 dark:focus:ring-primary-600 dark:ring-offset-gray-800 focus:ring-2 dark:bg-gray-700 dark:border-gray-600"
                />
              </div>
              <div class="ml-3">
                <div class="flex items-center gap-1"><label for="store_rdrc" class="text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ $t('experimental.cache.advanced.storeRdrc') }}
                </label><FieldHelp :label="$t('experimental.cache.advanced.storeRdrc')" :text="$t('experimental.cache.advanced.storeRdrcHelp')" /></div>

              </div>
            </div>

            <!-- RDRC Timeout -->
            <Reveal :open="!!settings.store_rdrc">
              <div class="flex items-center gap-1"><label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                {{ $t('experimental.cache.advanced.rdrcTimeout') }}
              </label><FieldHelp :label="$t('experimental.cache.advanced.rdrcTimeout')" :text="$t('experimental.cache.advanced.rdrcTimeoutHelp')" /></div>
              <Input
            :aria-label="$t('experimental.cache.advanced.rdrcTimeout')"
                v-model="settings.rdrc_timeout"
                placeholder="7d"
                :disabled="!settings.enabled || !settings.store_rdrc"
              />

            </Reveal>
          </div>
        </div>

        </div></Reveal>

        <!-- Save Button -->
        <div class="flex items-center justify-end gap-3 pt-4 border-t border-gray-200 dark:border-gray-700">
          <SaveFeedback :error="saveError" :message="savedMessage" />
          <!-- `action` drops the drop shadow: the footer sits inside the panel,
               so a raised pill reads as floating above it. -->
          <Button @click="handleSave" variant="primary" action :loading="saving">
            {{ $t('experimental.cache.save') }}
          </Button>
        </div>
      </fieldset>
    </Card>
  </div>
</template>
