<script setup lang="ts">
import { ref, watch, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import type { DNS } from '../types/api'
import Button from './Button.vue'
import Card from './Card.vue'
import { Select } from '../volt'
import { dnsService } from '../services'
import { useToast } from 'primevue'
import { useDNSStore } from '../stores/dns'
import { storeToRefs } from 'pinia'
import { dnsServerOptionLabel } from '../utils/dnsServerLabel'

const toast = useToast()
const { t } = useI18n()
const dnsStore = useDNSStore()
const { dnsServers } = storeToRefs(dnsStore)

// Local state
const loading = ref(false)
const dnsConfig = ref<DNS | null>(null)

const strategyOptions = computed(() => [
  { value: 'prefer_ipv4', label: t('dns.settings.strategyOptions.preferIpv4') },
  { value: 'prefer_ipv6', label: t('dns.settings.strategyOptions.preferIpv6') },
  { value: 'ipv4_only', label: t('dns.settings.strategyOptions.ipv4Only') },
  { value: 'ipv6_only', label: t('dns.settings.strategyOptions.ipv6Only') },
])

const serverOptions = computed(() => {
  const options = [
    { value: '', label: t('dns.settings.finalServerSelect') }
  ]
  if (dnsServers.value) {
    dnsServers.value.forEach(server => {
      options.push({ value: server.tag, label: dnsServerOptionLabel(server) })
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

// Fetch DNS config
const fetchDNSConfig = async () => {
  loading.value = true
  try {
    const { data } = await dnsService.getDNS()
    dnsConfig.value = data
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: err.message || t('dns.settings.toast.fetchFailed'),
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

// Watch for config changes. `immediate: true` is unnecessary because the
// initial value is always null (fetched after mount), so the guarded body
// would not run.
watch(() => dnsConfig.value, (newConfig) => {
  if (newConfig) {
    settings.value = {
      strategy: newConfig.strategy || 'prefer_ipv4',
      disable_cache: newConfig.disable_cache || false,
      disable_expire: newConfig.disable_expire || false,
      final: newConfig.final || '',
    }
  }
})

const handleSave = async () => {
  const updatedDNS = {
    ...dnsConfig.value,
    ...settings.value,
  } as DNS

  loading.value = true
  try {
    await dnsService.updateDNS(updatedDNS)
    toast.add({
      severity: 'success',
      summary: t('common.success'),
      detail: t('dns.settings.toast.updatedOk'),
      life: 3000
    })
    await fetchDNSConfig()
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: err.message || t('dns.settings.toast.updateFailed'),
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

// Load data on mount
onMounted(() => {
  fetchDNSConfig()
  dnsStore.fetchDNSServers() // Fetch shared DNS servers
})
</script>

<template>
  <div>
    <Card>
      <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-6">{{ $t('dns.settings.heading') }}</h3>

      <div v-if="loading" class="flex items-center justify-center py-12">
        <div class="animate-spin rounded-pill h-8 w-8 border-b-2 border-primary-600"></div>
      </div>

      <div v-else class="space-y-6">
        <!-- Domain Strategy -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            {{ $t('dns.settings.strategy') }}
          </label>
          <Select class="w-full" optionLabel="label" optionValue="value" v-model="settings.strategy" :options="strategyOptions" />
          <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
            {{ $t('dns.settings.strategyHelp') }}
          </p>
        </div>

        <!-- Final DNS Server -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            {{ $t('dns.settings.finalServer') }}
          </label>
          <Select class="w-full" optionLabel="label" optionValue="value" v-model="settings.final" :options="serverOptions" />
          <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
            {{ $t('dns.settings.finalServerHelp') }}
          </p>
        </div>

        <!-- Cache Settings -->
        <div class="border-t border-gray-200 dark:border-gray-700 pt-6">
          <h4 class="text-sm font-medium text-gray-900 dark:text-gray-100 mb-4">{{ $t('dns.settings.cacheHeading') }}</h4>

          <div class="space-y-4">
            <div class="flex items-start">
              <div class="flex items-center h-5">
                <input
                  type="checkbox"
                  id="disable_cache"
                  v-model="settings.disable_cache"
                  class="w-4 h-4 text-primary-600 bg-gray-100 border-gray-300 rounded focus:ring-primary-500 dark:focus:ring-primary-600 dark:ring-offset-gray-800 focus:ring-2 dark:bg-gray-700 dark:border-gray-600"
                />
              </div>
              <div class="ml-3">
                <label for="disable_cache" class="text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ $t('dns.settings.disableCache') }}
                </label>
                <p class="text-sm text-gray-500 dark:text-gray-400">
                  {{ $t('dns.settings.disableCacheHelp') }}
                </p>
              </div>
            </div>

            <div class="flex items-start">
              <div class="flex items-center h-5">
                <input
                  type="checkbox"
                  id="disable_expire"
                  v-model="settings.disable_expire"
                  class="w-4 h-4 text-primary-600 bg-gray-100 border-gray-300 rounded focus:ring-primary-500 dark:focus:ring-primary-600 dark:ring-offset-gray-800 focus:ring-2 dark:bg-gray-700 dark:border-gray-600"
                />
              </div>
              <div class="ml-3">
                <label for="disable_expire" class="text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ $t('dns.settings.disableExpire') }}
                </label>
                <p class="text-sm text-gray-500 dark:text-gray-400">
                  {{ $t('dns.settings.disableExpireHelp') }}
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
            {{ $t('dns.settings.save') }}
          </Button>
        </div>
      </div>
    </Card>
  </div>
</template>
