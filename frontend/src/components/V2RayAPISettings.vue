<script setup lang="ts">
import { ref, watch } from 'vue'
import type { V2RayAPI } from '../types/api'
import Button from './Button.vue'
import Card from './Card.vue'
import Input from './Input.vue'

const props = defineProps<{
  v2rayApi: V2RayAPI | null
  loading: boolean
}>()

const emit = defineEmits<{
  update: [v2rayAPI: V2RayAPI]
}>()

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

// Watch for props changes
watch(() => props.v2rayApi, (newConfig) => {
  if (newConfig) {
    settings.value = { ...newConfig }

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
    if (newConfig.stats) {
      inboundsText.value = newConfig.stats.inbounds?.join(', ') || ''
      outboundsText.value = newConfig.stats.outbounds?.join(', ') || ''
      usersText.value = newConfig.stats.users?.join(', ') || ''
    } else {
      inboundsText.value = ''
      outboundsText.value = ''
      usersText.value = ''
    }
  } else {
    // Reset to defaults when null
    settings.value = {
      listen: '',
      stats: {
        enabled: false,
        inbounds: [],
        outbounds: [],
        users: [],
      }
    }
    inboundsText.value = ''
    outboundsText.value = ''
    usersText.value = ''
  }
}, { immediate: true, deep: true })

// Convert comma-separated strings to arrays before saving
const handleSave = () => {
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

  emit('update', config)
}
</script>

<template>
  <div>
    <Card>
      <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-6">V2Ray API Settings</h3>

      <div v-if="loading" class="flex items-center justify-center py-12">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-violet-600"></div>
      </div>

      <div v-else class="space-y-6">
        <!-- API Listen Address -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            Listen Address
          </label>
          <Input
            v-model="settings.listen"
            placeholder="127.0.0.1:8080"
          />
          <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
            V2Ray API gRPC server listening address
          </p>
        </div>

        <!-- Stats Service -->
        <div class="border-t border-gray-200 dark:border-gray-700 pt-6">
          <h4 class="text-sm font-medium text-gray-900 dark:text-gray-100 mb-4">Stats Service</h4>

          <div class="space-y-4">
            <!-- Enable Stats -->
            <div class="flex items-start">
              <div class="flex items-center h-5">
                <input
                  type="checkbox"
                  id="stats_enabled"
                  v-model="settings.stats!.enabled"
                  class="w-4 h-4 text-violet-600 bg-gray-100 border-gray-300 rounded focus:ring-violet-500 dark:focus:ring-violet-600 dark:ring-offset-gray-800 focus:ring-2 dark:bg-gray-700 dark:border-gray-600"
                />
              </div>
              <div class="ml-3">
                <label for="stats_enabled" class="text-sm font-medium text-gray-700 dark:text-gray-300">
                  Enable Stats Service
                </label>
                <p class="text-sm text-gray-500 dark:text-gray-400">
                  Collect and provide traffic statistics
                </p>
              </div>
            </div>

            <!-- Monitored Inbounds -->
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Monitored Inbounds
              </label>
              <Input
                v-model="inboundsText"
                placeholder="http-in, socks-in (comma separated tags)"
                :disabled="!settings.stats?.enabled"
              />
              <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
                Inbound tags to collect statistics for
              </p>
            </div>

            <!-- Monitored Outbounds -->
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Monitored Outbounds
              </label>
              <Input
                v-model="outboundsText"
                placeholder="proxy-out, direct (comma separated tags)"
                :disabled="!settings.stats?.enabled"
              />
              <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
                Outbound tags to collect statistics for
              </p>
            </div>

            <!-- Monitored Users -->
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Monitored Users
              </label>
              <Input
                v-model="usersText"
                placeholder="user1, user2 (comma separated usernames)"
                :disabled="!settings.stats?.enabled"
              />
              <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
                User names to collect statistics for
              </p>
            </div>
          </div>
        </div>

        <!-- Info Box -->
        <div class="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4">
          <p class="text-sm text-blue-700 dark:text-blue-300">
            <strong>Note:</strong> V2Ray API provides a gRPC interface for programmatic access to sing-box statistics and control.
            This is useful for developing custom monitoring tools or integrations.
          </p>
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