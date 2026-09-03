<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
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

// Local state
const loading = ref(false)
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
  try {
    const { data } = await experimentalService.getClashAPI()
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

  loading.value = true
  try {
    await experimentalService.updateClashAPI(config)
    toast.add({
      severity: 'success',
      summary: t('common.success'),
      detail: t('experimental.clash.toast.updatedOk'),
      life: 3000
    })
    await fetchClashAPI()
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: err.message || t('experimental.clash.toast.updateFailed'),
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

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

      <div v-else class="space-y-4">
        <!-- External Controller -->
        <div>
          <label class="mb-2 flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-300">
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
          </label>
          <Input
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
          <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
            {{ $t('experimental.clash.externalControllerHelp') }}
          </p>
        </div>

        <!-- External UI -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            {{ $t('experimental.clash.externalUi') }}
          </label>
          <Input
            v-model="settings.external_ui"
            placeholder="/usr/share/sing-box/ui"
          />
          <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
            {{ $t('experimental.clash.externalUiHelp') }}
          </p>
        </div>

        <!-- External UI Download URL -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            {{ $t('experimental.clash.externalUiDownloadUrl') }}
          </label>
          <DashboardUrlSelect v-model="settings.external_ui_download_url" />
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
          <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
            {{ $t('experimental.clash.externalUiDownloadUrlHelp') }}
          </p>
        </div>

        <!-- External UI Download Detour -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            {{ $t('experimental.clash.externalUiDownloadDetour') }}
          </label>
          <Input
            v-model="settings.external_ui_download_detour"
            placeholder="proxy"
          />
          <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
            {{ $t('experimental.clash.externalUiDownloadDetourHelp') }}
          </p>
        </div>

        <!-- Secret -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            {{ $t('experimental.clash.secret') }}
          </label>
          <Input
            v-model="settings.secret"
            type="password"
            :placeholder="$t('experimental.clash.secretPlaceholder')"
          />
          <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
            {{ $t('experimental.clash.secretHelp') }}
          </p>
        </div>

        <!-- Default Mode -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            {{ $t('experimental.clash.defaultMode') }}
          </label>
          <Select class="w-full" optionLabel="label" optionValue="value" v-model="settings.default_mode" :options="modeOptions" />
          <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
            {{ $t('experimental.clash.defaultModeHelp') }}
          </p>
        </div>

        <!-- CORS Settings -->
        <div class="border-t border-gray-200 dark:border-gray-700 pt-4">
          <h4 class="text-sm font-medium text-gray-900 dark:text-gray-100 mb-4">{{ $t('experimental.clash.cors.title') }}</h4>

          <div class="space-y-4">
            <!-- Access Control Allow Origin -->
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                {{ $t('experimental.clash.cors.allowOrigin') }}
              </label>
              <Input
                v-model="allowOriginText"
                :placeholder="$t('experimental.clash.cors.allowOriginPlaceholder')"
              />
              <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
                {{ $t('experimental.clash.cors.allowOriginHelp') }}
              </p>
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
                <label for="allow_private_network" class="text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ $t('experimental.clash.cors.allowPrivateNetwork') }}
                </label>
                <p class="text-sm text-gray-500 dark:text-gray-400">
                  {{ $t('experimental.clash.cors.allowPrivateNetworkHelp') }}
                </p>
              </div>
            </div>
          </div>
        </div>

        <!-- Save Button -->
        <div class="flex justify-end pt-4 border-t border-gray-200 dark:border-gray-700">
          <!-- `action` drops the drop shadow: the footer sits inside the panel,
               so a raised pill reads as floating above it. -->
          <Button @click="handleSave" variant="primary" action :disabled="loading">
            {{ $t('experimental.clash.save') }}
          </Button>
        </div>
      </div>
    </Card>
  </div>
</template>