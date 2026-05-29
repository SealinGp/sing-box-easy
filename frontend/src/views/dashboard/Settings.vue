<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { settingsService } from '../../services'
import { Code } from '../../types/api'
import { useToast } from 'primevue'
import LanguageSwitcher from '../../components/LanguageSwitcher.vue'

const toast = useToast()
const { t } = useI18n()

const loading = ref(false)
const saving = ref(false)
const keep = ref(10)
const limits = ref({ min: 1, max: 100 })
const version = __APP_VERSION__

onMounted(async () => {
  await loadSettings()
})

const loadSettings = async () => {
  loading.value = true
  try {
    const { data } = await settingsService.getSettings()
    keep.value = data.config_versions_keep
  } catch (err: any) {
    toast.add({ severity: 'error', summary: t('common.error'), detail: err.message || t('settings.toast.loadFailed'), life: 3000 })
  } finally {
    loading.value = false
  }
}

const saveSettings = async () => {
  saving.value = true
  try {
    const resp = await settingsService.updateSettings({ config_versions_keep: keep.value })
    if (resp.code !== Code.Success) {
      toast.add({ severity: 'error', summary: t('settings.toast.invalidTitle'), detail: resp.msg, life: 4000 })
      return
    }
    keep.value = resp.data.config_versions_keep
    limits.value = resp.data.limits
    toast.add({ severity: 'success', summary: t('settings.toast.savedTitle'), detail: t('settings.toast.savedDetail'), life: 3000 })
  } catch (err: any) {
    toast.add({ severity: 'error', summary: t('common.error'), detail: err.message || t('settings.toast.saveFailed'), life: 3000 })
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="p-4">
    <h2 class="text-3xl font-bold text-gray-900 dark:text-gray-100 mb-6">{{ $t('settings.title') }}</h2>

    <div class="max-w-xl space-y-6">
      <!-- Language -->
      <div class="bg-white dark:bg-gray-800 rounded-lg shadow p-5">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-1">{{ $t('settings.language.title') }}</h3>
        <p class="text-sm text-gray-500 dark:text-gray-400 mb-4">{{ $t('settings.language.desc') }}</p>
        <LanguageSwitcher variant="full" />
      </div>

      <!-- Config version retention -->
      <div class="bg-white dark:bg-gray-800 rounded-lg shadow p-5">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-1">{{ $t('settings.versionHistory.title') }}</h3>
        <p class="text-sm text-gray-500 dark:text-gray-400 mb-4">
          {{ $t('settings.versionHistory.desc', { min: limits.min, max: limits.max }) }}
        </p>

        <div v-if="loading" class="h-10 flex items-center">
          <div class="animate-spin rounded-full h-5 w-5 border-b-2 border-violet-600"></div>
        </div>
        <div v-else class="flex items-end gap-3">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('settings.versionHistory.versionsToKeep') }}</label>
            <input
              v-model.number="keep"
              type="number"
              :min="limits.min"
              :max="limits.max"
              class="w-32 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-900 px-3 py-2 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"
            />
          </div>
          <button
            @click="saveSettings"
            :disabled="saving"
            class="px-4 py-2 text-sm font-medium text-white bg-violet-600 rounded-lg hover:bg-violet-700 transition-colors disabled:opacity-50"
          >
            <span v-if="saving">{{ $t('common.saving') }}</span>
            <span v-else>{{ $t('common.save') }}</span>
          </button>
        </div>
      </div>

      <!-- About -->
      <div class="bg-white dark:bg-gray-800 rounded-lg shadow p-5">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-1">{{ $t('settings.about.title') }}</h3>
        <p class="text-sm text-gray-500 dark:text-gray-400">
          sing-box-easy <span class="font-mono text-gray-700 dark:text-gray-300">{{ version }}</span>
        </p>
      </div>
    </div>
  </div>
</template>
