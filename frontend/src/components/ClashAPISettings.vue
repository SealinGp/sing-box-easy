<script setup lang="ts">
import { ref, onMounted } from 'vue'
import type { ClashAPI } from '../types/api'
import Button from './Button.vue'
import Card from './Card.vue'
import Input from './Input.vue'
import Select from './Select.vue'
import { experimentalService } from '../services'
import { useToast } from 'primevue'
import { LinkIcon } from '@heroicons/vue/24/outline'

const toast = useToast()

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

// Mode options
const modeOptions = [
  { value: 'rule', label: 'Rule Mode' },
  { value: 'global', label: 'Global Mode' },
  { value: 'direct', label: 'Direct Mode' },
]

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
      summary: 'Error',
      detail: err.response?.data?.error || 'Failed to fetch Clash API configuration',
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
      summary: 'Success',
      detail: 'Clash API configuration updated successfully',
      life: 3000
    })
    await fetchClashAPI()
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.response?.data?.error || 'Failed to update Clash API configuration',
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
      <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-6">Clash API Settings</h3>

      <div v-if="loading" class="flex items-center justify-center py-12">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-violet-600"></div>
      </div>

      <div v-else class="space-y-6">
        <!-- External Controller -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2 flex">
            External Controller
            <a class="cursor-pointer hover:opacity-50" :href="`http://${settings.external_controller}`" target="_blank">
              <LinkIcon class="h-5 w-5 mr-2" />
            </a>
          </label>
          <Input
            v-model="settings.external_controller"
            placeholder="127.0.0.1:9090"
          />
          <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
            RESTful API listening address
          </p>
        </div>

        <!-- External UI -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            External UI
          </label>
          <Input
            v-model="settings.external_ui"
            placeholder="/usr/share/sing-box/ui"
          />
          <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
            Path to external UI directory
          </p>
        </div>

        <!-- External UI Download URL -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            External UI Download URL
          </label>
          <Input
            v-model="settings.external_ui_download_url"
            placeholder="https://github.com/MetaCubeX/Yacd-meta/archive/gh-pages.zip"
          />
          <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
            URL to download UI if not exists
          </p>
        </div>

        <!-- External UI Download Detour -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            External UI Download Detour
          </label>
          <Input
            v-model="settings.external_ui_download_detour"
            placeholder="proxy"
          />
          <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
            Outbound tag for downloading UI
          </p>
        </div>

        <!-- Secret -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            Secret
          </label>
          <Input
            v-model="settings.secret"
            type="password"
            placeholder="Enter secret key"
          />
          <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
            Authentication secret for API access
          </p>
        </div>

        <!-- Default Mode -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            Default Mode
          </label>
          <Select v-model="settings.default_mode" :options="modeOptions" />
          <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
            Default proxy mode for Clash API
          </p>
        </div>

        <!-- CORS Settings -->
        <div class="border-t border-gray-200 dark:border-gray-700 pt-6">
          <h4 class="text-sm font-medium text-gray-900 dark:text-gray-100 mb-4">CORS Settings</h4>

          <div class="space-y-4">
            <!-- Access Control Allow Origin -->
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Access Control Allow Origin
              </label>
              <Input
                v-model="allowOriginText"
                placeholder="http://localhost:3000, https://example.com (comma separated)"
              />
              <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
                Allowed origins for CORS requests
              </p>
            </div>

            <!-- Access Control Allow Private Network -->
            <div class="flex items-start">
              <div class="flex items-center h-5">
                <input
                  type="checkbox"
                  id="allow_private_network"
                  v-model="settings.access_control_allow_private_network"
                  class="w-4 h-4 text-violet-600 bg-gray-100 border-gray-300 rounded focus:ring-violet-500 dark:focus:ring-violet-600 dark:ring-offset-gray-800 focus:ring-2 dark:bg-gray-700 dark:border-gray-600"
                />
              </div>
              <div class="ml-3">
                <label for="allow_private_network" class="text-sm font-medium text-gray-700 dark:text-gray-300">
                  Allow Private Network Access
                </label>
                <p class="text-sm text-gray-500 dark:text-gray-400">
                  Allow requests from private network addresses
                </p>
              </div>
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