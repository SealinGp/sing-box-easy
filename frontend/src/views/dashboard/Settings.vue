<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { settingsService } from '../../services'
import { useNotify } from '../../composables/useNotify'
import AboutCard from '../../components/AboutCard.vue'
import GitHubAuthCard from '../../components/GitHubAuthCard.vue'

const notify = useNotify()
const { t } = useI18n()

const loading = ref(false)
const saving = ref(false)
const keep = ref(10)
const limits = ref({ min: 1, max: 100 })

onMounted(async () => {
  await loadSettings()
})

const loadSettings = async () => {
  loading.value = true
  try {
    const { data } = await settingsService.getSettings()
    keep.value = data.config_versions_keep
  } catch (err) {
    notify.apiError(err, t('settings.toast.loadFailed'))
  } finally {
    loading.value = false
  }
}

const saveSettings = async () => {
  saving.value = true
  try {
    const resp = await settingsService.updateSettings({ config_versions_keep: keep.value })
    keep.value = resp.data.config_versions_keep
    limits.value = resp.data.limits
    notify.success(t('settings.toast.savedDetail'), t('settings.toast.savedTitle'))
  } catch (err) {
    notify.apiError(err, t('settings.toast.saveFailed'))
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="p-4">
    <h2 class="text-3xl font-bold text-gray-900 dark:text-gray-100 mb-6">{{ $t('settings.title') }}</h2>

    <!--
      Cards flow into a responsive grid rather than a single narrow column:
      one per row when cramped, two from a 768px-wide container, three from
      1152px — which keeps every card at least ~370px, the width the About
      card's longest row (a fully-qualified hostname) needs without truncating.

      Container queries, not viewport ones: this grid's width depends on the
      navigation shell around it. The same 1280px screen leaves ~1030px here
      with the sidebar but ~1250px with the OpenWrt top bar, so keying off the
      viewport would either crush the sidebar layout or waste the top-bar one.

      `items-start` keeps each card at its natural height — without it the grid
      stretches the short retention card to match the tall About card, leaving
      a large empty box.
    -->
    <div class="@container">
      <div class="grid grid-cols-1 @3xl:grid-cols-2 @6xl:grid-cols-3 gap-6 items-start">
        <!--
          About: what is running and on what, plus the language picker and the
          self-update controls. Deliberately one card — these are the things an
          operator checks together when reporting or diagnosing a problem.
        -->
        <AboutCard />

        <!-- GitHub sign-in (lifts the 60 req/h anonymous API rate limit) -->
        <GitHubAuthCard />

        <!-- Config version retention -->
        <div class="bg-white dark:bg-gray-800 rounded-surface shadow p-5">
          <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-1">{{ $t('settings.versionHistory.title') }}</h3>
          <p class="text-sm text-gray-500 dark:text-gray-400 mb-4">
            {{ $t('settings.versionHistory.desc', { min: limits.min, max: limits.max }) }}
          </p>

          <div v-if="loading" class="h-10 flex items-center">
            <div class="animate-spin rounded-pill h-5 w-5 border-b-2 border-primary-600"></div>
          </div>
          <div v-else class="flex items-end gap-3">
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('settings.versionHistory.versionsToKeep') }}</label>
              <input
                v-model.number="keep"
                type="number"
                :min="limits.min"
                :max="limits.max"
                class="w-32 rounded-control border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-900 px-3 py-2 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
              />
            </div>
            <button
              @click="saveSettings"
              :disabled="saving"
              class="px-4 py-2 text-sm font-medium text-white bg-primary-600 rounded-control hover:bg-primary-700 transition-colors disabled:opacity-50"
            >
              <span v-if="saving">{{ $t('common.saving') }}</span>
              <span v-else>{{ $t('common.save') }}</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
