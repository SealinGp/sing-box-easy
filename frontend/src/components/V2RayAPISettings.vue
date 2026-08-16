<script setup lang="ts">
import { ref, onMounted } from 'vue'
import type { V2RayAPI } from '../types/api'
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
const settings = ref<V2RayAPI>({
  listen: '',
  stats: {
    enabled: false,
    inbounds: [],
    outbounds: [],
    users: [],
  }
})

const inboundsText = ref('')
const outboundsText = ref('')
const usersText = ref('')

// Fetch V2Ray API config
const fetchV2RayAPI = async () => {
  loading.value = true
  try {
    const { data } = await experimentalService.getV2RayAPI()
    if (data) {
      settings.value = { ...data }

      // Initialize stats if not exists
      if (!settings.value.stats) {
        settings.value.stats = {
          enabled: false,
          inbounds: [],
          outbounds: [],
          users: [],
        }
      }

      // Convert arrays to comma-separated strings for display
      if (data.stats) {
        inboundsText.value = data.stats.inbounds?.join(', ') || ''
        outboundsText.value = data.stats.outbounds?.join(', ') || ''
        usersText.value = data.stats.users?.join(', ') || ''
      }
    }
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: err.message || t('experimental.v2ray.toast.fetchFailed'),
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

// Update V2Ray API config
const handleSave = async () => {
  const config: V2RayAPI = {
    listen: settings.value.listen,
    stats: {
      enabled: settings.value.stats?.enabled || false,
      inbounds: [],
      outbounds: [],
      users: [],
    }
  }

  // Convert text inputs to arrays
  if (config.stats) {
    if (inboundsText.value) {
      config.stats.inbounds = inboundsText.value
        .split(',')
        .map(s => s.trim())
        .filter(Boolean)
    }

    if (outboundsText.value) {
      config.stats.outbounds = outboundsText.value
        .split(',')
        .map(s => s.trim())
        .filter(Boolean)
    }

    if (usersText.value) {
      config.stats.users = usersText.value
        .split(',')
        .map(s => s.trim())
        .filter(Boolean)
    }
  }

  loading.value = true
  try {
    await experimentalService.updateV2RayAPI(config)
    toast.add({
      severity: 'success',
      summary: t('common.success'),
      detail: t('experimental.v2ray.toast.updatedOk'),
      life: 3000
    })
    await fetchV2RayAPI()
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: err.message || t('experimental.v2ray.toast.updateFailed'),
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

// Load data on mount
onMounted(() => {
  fetchV2RayAPI()
})
</script>

<template>
  <div>
    <Card>
      <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">{{ $t('experimental.v2ray.title') }}</h3>

      <div v-if="loading" class="flex items-center justify-center py-12">
        <div class="animate-spin rounded-pill h-8 w-8 border-b-2 border-primary-600"></div>
      </div>

      <div v-else class="space-y-4">
        <!-- API Listen Address -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            {{ $t('experimental.v2ray.listen') }}
          </label>
          <Input
            v-model="settings.listen"
            placeholder="127.0.0.1:8080"
          />
          <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
            {{ $t('experimental.v2ray.listenHelp') }}
          </p>
        </div>

        <!-- Stats Service -->
        <div class="border-t border-gray-200 dark:border-gray-700 pt-4">
          <h4 class="text-sm font-medium text-gray-900 dark:text-gray-100 mb-4">{{ $t('experimental.v2ray.stats.title') }}</h4>

          <div class="space-y-4">
            <!-- Enable Stats -->
            <div class="flex items-start">
              <div class="flex items-center h-5">
                <input
                  type="checkbox"
                  id="stats_enabled"
                  v-model="settings.stats!.enabled"
                  class="w-4 h-4 text-primary-600 bg-gray-100 border-gray-300 rounded focus:ring-primary-500 dark:focus:ring-primary-600 dark:ring-offset-gray-800 focus:ring-2 dark:bg-gray-700 dark:border-gray-600"
                />
              </div>
              <div class="ml-3">
                <label for="stats_enabled" class="text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ $t('experimental.v2ray.stats.enable') }}
                </label>
                <p class="text-sm text-gray-500 dark:text-gray-400">
                  {{ $t('experimental.v2ray.stats.enableHelp') }}
                </p>
              </div>
            </div>

            <!-- Monitored Inbounds -->
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                {{ $t('experimental.v2ray.stats.inbounds') }}
              </label>
              <Input
                v-model="inboundsText"
                :placeholder="$t('experimental.v2ray.stats.inboundsPlaceholder')"
                :disabled="!settings.stats?.enabled"
              />
              <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
                {{ $t('experimental.v2ray.stats.inboundsHelp') }}
              </p>
            </div>

            <!-- Monitored Outbounds -->
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                {{ $t('experimental.v2ray.stats.outbounds') }}
              </label>
              <Input
                v-model="outboundsText"
                :placeholder="$t('experimental.v2ray.stats.outboundsPlaceholder')"
                :disabled="!settings.stats?.enabled"
              />
              <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
                {{ $t('experimental.v2ray.stats.outboundsHelp') }}
              </p>
            </div>

            <!-- Monitored Users -->
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                {{ $t('experimental.v2ray.stats.users') }}
              </label>
              <Input
                v-model="usersText"
                :placeholder="$t('experimental.v2ray.stats.usersPlaceholder')"
                :disabled="!settings.stats?.enabled"
              />
              <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
                {{ $t('experimental.v2ray.stats.usersHelp') }}
              </p>
            </div>
          </div>
        </div>

        <!-- Info Box -->
        <div class="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-surface p-4">
          <p class="text-sm text-blue-700 dark:text-blue-300">
            <strong>{{ $t('experimental.v2ray.note') }}</strong> {{ $t('experimental.v2ray.noteText') }}
          </p>
        </div>

        <!-- Save Button -->
        <div class="flex justify-end pt-4 border-t border-gray-200 dark:border-gray-700">
          <!-- `action` drops the drop shadow: the footer sits inside the panel,
               so a raised pill reads as floating above it. -->
          <Button @click="handleSave" variant="primary" action :disabled="loading">
            {{ $t('experimental.v2ray.save') }}
          </Button>
        </div>
      </div>
    </Card>
  </div>
</template>