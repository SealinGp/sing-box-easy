<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import type { DNS, DNSServer } from '../types/api'
import Button from './Button.vue'
import Card from './Card.vue'
import Select from './Select.vue'

const props = defineProps<{
  dnsConfig: DNS | null
  servers: DNSServer[]
  loading: boolean
}>()

const emit = defineEmits<{
  update: [dns: DNS]
}>()

const strategyOptions = [
  { value: 'prefer_ipv4', label: 'Prefer IPv4' },
  { value: 'prefer_ipv6', label: 'Prefer IPv6' },
  { value: 'ipv4_only', label: 'IPv4 Only' },
  { value: 'ipv6_only', label: 'IPv6 Only' },
]

const serverOptions = computed(() => {
  const options = [
    { value: '', label: 'Select default server' }
  ]
  if (props.servers) {
    props.servers.forEach(server => {
      options.push({ value: server.tag, label: server.tag })
    })
  }
  return options
})

const settings = ref({
  strategy: 'prefer_ipv4',
  disable_cache: false,
  disable_expire: false,
  final: '',
})

// Watch for props changes
watch(() => props.dnsConfig, (newConfig) => {
  if (newConfig) {
    settings.value = {
      strategy: newConfig.strategy || 'prefer_ipv4',
      disable_cache: newConfig.disable_cache || false,
      disable_expire: newConfig.disable_expire || false,
      final: newConfig.final || '',
    }
  }
}, { immediate: true })

const handleSave = () => {
  const updatedDNS = {
    ...props.dnsConfig,
    ...settings.value,
  } as DNS
  emit('update', updatedDNS)
}
</script>

<template>
  <div>
    <Card>
      <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-6">Global DNS Settings</h3>

      <div v-if="loading" class="flex items-center justify-center py-12">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-violet-600"></div>
      </div>

      <div v-else class="space-y-6">
        <!-- Domain Strategy -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            Domain Strategy
          </label>
          <Select v-model="settings.strategy" :options="strategyOptions" />
          <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
            IP version preference for DNS queries
          </p>
        </div>

        <!-- Final DNS Server -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            Final DNS Server
          </label>
          <Select v-model="settings.final" :options="serverOptions" />
          <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
            Fallback server when no rules match
          </p>
        </div>

        <!-- Cache Settings -->
        <div class="border-t border-gray-200 dark:border-gray-700 pt-6">
          <h4 class="text-sm font-medium text-gray-900 dark:text-gray-100 mb-4">Cache Settings</h4>

          <div class="space-y-4">
            <div class="flex items-start">
              <div class="flex items-center h-5">
                <input
                  type="checkbox"
                  id="disable_cache"
                  v-model="settings.disable_cache"
                  class="w-4 h-4 text-violet-600 bg-gray-100 border-gray-300 rounded focus:ring-violet-500 dark:focus:ring-violet-600 dark:ring-offset-gray-800 focus:ring-2 dark:bg-gray-700 dark:border-gray-600"
                />
              </div>
              <div class="ml-3">
                <label for="disable_cache" class="text-sm font-medium text-gray-700 dark:text-gray-300">
                  Disable DNS Cache
                </label>
                <p class="text-sm text-gray-500 dark:text-gray-400">
                  Turn off caching of DNS responses
                </p>
              </div>
            </div>

            <div class="flex items-start">
              <div class="flex items-center h-5">
                <input
                  type="checkbox"
                  id="disable_expire"
                  v-model="settings.disable_expire"
                  class="w-4 h-4 text-violet-600 bg-gray-100 border-gray-300 rounded focus:ring-violet-500 dark:focus:ring-violet-600 dark:ring-offset-gray-800 focus:ring-2 dark:bg-gray-700 dark:border-gray-600"
                />
              </div>
              <div class="ml-3">
                <label for="disable_expire" class="text-sm font-medium text-gray-700 dark:text-gray-300">
                  Disable Cache Expiration
                </label>
                <p class="text-sm text-gray-500 dark:text-gray-400">
                  Keep cached DNS records indefinitely
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
