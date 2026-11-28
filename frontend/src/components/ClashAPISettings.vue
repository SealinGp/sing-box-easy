<script setup lang="ts">
import { configSnapshot } from '../utils/configSnapshot'
import FieldHelp from './FieldHelp.vue'
import Reveal from './Reveal.vue'
import SaveFeedback from './SaveFeedback.vue'
import { useInlineSave } from '../composables/useInlineSave'
import { watch, ref, computed, onMounted } from 'vue'
import type { ClashAPI } from '../types/api'
import Button from './Button.vue'
import Card from './Card.vue'
import Input from './Input.vue'
import { Select } from '../volt'
import DashboardUrlSelect from './DashboardUrlSelect.vue'
import DashboardDownloader from './DashboardDownloader.vue'
import { experimentalService } from '../services'
import { useToast } from 'primevue'
import { useI18n } from 'vue-i18n'
import { LinkIcon } from '@heroicons/vue/24/outline'
import {
  clashDashboardHref,
  hasDashboard as clashHasDashboard,
  isLinkBlockedBySecret,
} from '../utils/clashDashboardUrl'

const toast = useToast()
const { t } = useI18n()
const snapshot = configSnapshot(() => t('nav.stalePage'))

// Local state
const loading = ref(false)
const loadError = ref('')
const { saving, saveError, savedMessage, clearSaveFeedback, persist } = useInlineSave()
const enabled = ref(false)
const settings = ref<ClashAPI>({
  external_controller: '',
  external_ui: '',
  external_ui_download_url: '',
  external_ui_download_detour: '',
  secret: '',
  default_mode: 'rule',
  access_control_allow_origin: [],
  access_control_allow_private_network: false,
})

const allowOriginText = ref('')

/*
 * External controller link.
 *
 * The rules (why the bare API root 401s under a secret, why /ui/ does not, and
 * how each dashboard wants the backend details) live in utils/clashDashboardUrl
 * — shared with the Overview card, which offers the same link.
 */
const pageHost = () => window.location.hostname

const externalControllerHref = computed(() => clashDashboardHref(settings.value, pageHost()))

const hasDashboard = computed(() => clashHasDashboard(settings.value))

/**
 * True when the controller is configured and secured, but no dashboard exists
 * to open — the one case where we deliberately render no link, because any
 * link we could produce would 401.
 */
const linkBlockedBySecret = computed(() => isLinkBlockedBySecret(settings.value, pageHost()))

// Mode options
const modeOptions = computed(() => [
  { value: 'rule', label: t('experimental.clash.mode.rule') },
  { value: 'global', label: t('experimental.clash.mode.global') },
  { value: 'direct', label: t('experimental.clash.mode.direct') },
])

// Fetch Clash API config
const fetchClashAPI = async () => {
  loading.value = true
  loadError.value = ''
  try {
    const { data } = await experimentalService.getClashAPI()
    snapshot.remember(data)
    enabled.value = Boolean(data?.external_controller)
    if (data) {
      settings.value = { ...data }
      // Convert array to comma-separated string for display
      if (data.access_control_allow_origin && Array.isArray(data.access_control_allow_origin)) {
        allowOriginText.value = data.access_control_allow_origin.join(', ')
      } else {
        allowOriginText.value = ''
      }
    }
  } catch (err: any) {
    loadError.value = err.message || t('experimental.clash.toast.fetchFailed')
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: err.message || t('experimental.clash.toast.fetchFailed'),
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

// Update Clash API config
const handleSave = async () => {
  if (saving.value || loading.value || loadError.value) return
  const config = { ...settings.value }

  // Convert origins text to array
  if (allowOriginText.value) {
    config.access_control_allow_origin = allowOriginText.value
      .split(',')
      .map(s => s.trim())
      .filter(Boolean)
  } else {
    config.access_control_allow_origin = []
  }

  await persist(async () => {
    snapshot.assertCurrent((await experimentalService.getClashAPI()).data)
    await experimentalService.updateClashAPI(enabled.value ? config : {})
    snapshot.remember((await experimentalService.getClashAPI()).data)
  },
    t('experimental.clash.toast.updatedOk'), t('experimental.clash.toast.updateFailed'))
}

watch([settings, enabled, allowOriginText], clearSaveFeedback, { deep: true })

// Load data on mount
onMounted(() => {
  fetchClashAPI()
})
</script>

<template>
  <div>
    <Card>
      <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">{{ $t('experimental.clash.title') }}</h3>

      <div v-if="loading" class="flex items-center justify-center py-12">
        <div class="animate-spin rounded-pill h-8 w-8 border-b-2 border-primary-600"></div>
      </div>

      <div v-else-if="loadError" class="space-y-3">
        <p role="alert" class="text-sm text-danger">{{ loadError }}</p>
        <Button variant="secondary" @click="fetchClashAPI">{{ $t('common.refresh') }}</Button>
      </div>

      <fieldset v-else :disabled="saving" @input="clearSaveFeedback" @change="clearSaveFeedback" class="min-w-0 space-y-4">
        <label class="flex items-center gap-2 text-sm font-medium text-text-primary"><input v-model="enabled" type="checkbox" class="h-4 w-4 accent-primary" />{{ $t('workspace.clashEnabled') }}</label>
        <Reveal :open="enabled"><div class="space-y-4">
        <!-- External Controller -->
        <div>
          <div class="flex items-center gap-1"><label class="mb-2 flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ $t('experimental.clash.externalController') }}
            <a
              v-if="externalControllerHref"
              class="cursor-pointer hover:opacity-50"
              :href="externalControllerHref"
              target="_blank"
              rel="noopener noreferrer"
              :title="
                hasDashboard
                  ? $t('experimental.clash.openDashboard')
                  : $t('experimental.clash.openController')
              "
            >
              <LinkIcon class="h-5 w-5" />
            </a>
          </label><FieldHelp :label="$t('experimental.clash.externalController')" :text="$t('experimental.clash.externalControllerHelp')" /></div>
          <Input
            :aria-label="$t('experimental.clash.externalController')"
            v-model="settings.external_controller"
            placeholder="127.0.0.1:9090"
          />
          <!--
            The link is deliberately absent here: a secret is set but no
            dashboard is installed, so every URL we could offer would return
            401 Unauthorized. Explain that rather than showing a broken link.
          -->
          <p
            v-if="linkBlockedBySecret"
            class="mt-2 text-sm text-amber-600 dark:text-amber-400"
          >
            {{ $t('experimental.clash.secretBlocksDirectLink') }}
          </p>

        </div>

        <!-- External UI -->
        <div>
          <div class="flex items-center gap-1"><label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            {{ $t('experimental.clash.externalUi') }}
          </label><FieldHelp :label="$t('experimental.clash.externalUi')" :text="$t('experimental.clash.externalUiHelp')" /></div>
          <Input
            :aria-label="$t('experimental.clash.externalUi')"
            v-model="settings.external_ui"
            placeholder="/usr/share/sing-box/ui"
          />

        </div>

        <!-- External UI Download URL -->
        <div>
          <div class="flex items-center gap-1"><label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            {{ $t('experimental.clash.externalUiDownloadUrl') }}
          </label><FieldHelp :label="$t('experimental.clash.externalUiDownloadUrl')" :text="$t('experimental.clash.externalUiDownloadUrlHelp')" /></div>
          <DashboardUrlSelect :disabled="saving" v-model="settings.external_ui_download_url" />
          <!-- Fetch the chosen dashboard into `external_ui` without leaving the page. -->
          <!-- The download writes `external_ui` server-side; mirror that back
               into the field rather than refetching, which would discard any
               other edit the operator has not saved yet. -->
          <DashboardDownloader
            class="mt-3"
            :target-dir="settings.external_ui"
            :download-url="settings.external_ui_download_url"
            @installed="settings.external_ui = $event"
          />

        </div>

        <!-- External UI Download Detour -->
        <div>
          <div class="flex items-center gap-1"><label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            {{ $t('experimental.clash.externalUiDownloadDetour') }}
          </label><FieldHelp :label="$t('experimental.clash.externalUiDownloadDetour')" :text="$t('experimental.clash.externalUiDownloadDetourHelp')" /></div>
          <Input
            :aria-label="$t('experimental.clash.externalUiDownloadDetour')"
            v-model="settings.external_ui_download_detour"
            placeholder="proxy"
          />

        </div>

        <!-- Secret -->
        <div>
          <div class="flex items-center gap-1"><label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            {{ $t('experimental.clash.secret') }}
          </label><FieldHelp :label="$t('experimental.clash.secret')" :text="$t('experimental.clash.secretHelp')" /></div>
          <Input
            :aria-label="$t('experimental.clash.secret')"
            v-model="settings.secret"
            type="password"
            :placeholder="$t('experimental.clash.secretPlaceholder')"
          />

        </div>

        <!-- Default Mode -->
        <div>
          <div class="flex items-center gap-1"><label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            {{ $t('experimental.clash.defaultMode') }}
          </label><FieldHelp :label="$t('experimental.clash.defaultMode')" :text="$t('experimental.clash.defaultModeHelp')" /></div>
          <Select :disabled="saving" class="w-full" optionLabel="label" optionValue="value" v-model="settings.default_mode" :options="modeOptions" />

        </div>

        <!-- CORS Settings -->
        <div class="border-t border-gray-200 dark:border-gray-700 pt-4">
          <h4 class="text-sm font-medium text-gray-900 dark:text-gray-100 mb-4">{{ $t('experimental.clash.cors.title') }}</h4>

          <div class="space-y-4">
            <!-- Access Control Allow Origin -->
            <div>
              <div class="flex items-center gap-1"><label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                {{ $t('experimental.clash.cors.allowOrigin') }}
              </label><FieldHelp :label="$t('experimental.clash.cors.allowOrigin')" :text="$t('experimental.clash.cors.allowOriginHelp')" /></div>
              <Input
            :aria-label="$t('experimental.clash.cors.allowOrigin')"
                v-model="allowOriginText"
                :placeholder="$t('experimental.clash.cors.allowOriginPlaceholder')"
              />

            </div>

            <!-- Access Control Allow Private Network -->
            <div class="flex items-start">
              <div class="flex items-center h-5">
                <input
                  type="checkbox"
                  id="allow_private_network"
                  v-model="settings.access_control_allow_private_network"
                  class="w-4 h-4 text-primary-600 bg-gray-100 border-gray-300 rounded focus:ring-primary-500 dark:focus:ring-primary-600 dark:ring-offset-gray-800 focus:ring-2 dark:bg-gray-700 dark:border-gray-600"
                />
              </div>
              <div class="ml-3">
                <div class="flex items-center gap-1"><label for="allow_private_network" class="text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ $t('experimental.clash.cors.allowPrivateNetwork') }}
                </label><FieldHelp :label="$t('experimental.clash.cors.allowPrivateNetwork')" :text="$t('experimental.clash.cors.allowPrivateNetworkHelp')" /></div>

              </div>
            </div>
          </div>
        </div>

        </div></Reveal>

        <!-- Save Button -->
        <div class="flex items-center justify-end gap-3 pt-4 border-t border-gray-200 dark:border-gray-700">
          <SaveFeedback :error="saveError" :message="savedMessage" />
          <!-- `action` drops the drop shadow: the footer sits inside the panel,
               so a raised pill reads as floating above it. -->
          <Button @click="handleSave" variant="primary" action :loading="saving">
            {{ $t('experimental.clash.save') }}
          </Button>
        </div>
      </fieldset>
    </Card>
  </div>
</template>
