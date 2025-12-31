<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import type { CacheFile, ClashAPI, V2RayAPI } from '../../types/api'
import CacheFileSettings from '../../components/CacheFileSettings.vue'
import ClashAPISettings from '../../components/ClashAPISettings.vue'
import V2RayAPISettings from '../../components/V2RayAPISettings.vue'
import { experimentalService } from '../../services'
import { useToast } from 'primevue'

const toast = useToast()

// State
const loading = ref(false)
const activeTab = ref('cache-file')

// Data
const cacheFileConfig = ref<CacheFile | null>(null)
const clashAPIConfig = ref<ClashAPI | null>(null)
const v2rayAPIConfig = ref<V2RayAPI | null>(null)

// Tabs
const tabs = [
  { id: 'cache-file', label: 'Cache File' },
  { id: 'clash-api', label: 'Clash API' },
  { id: 'v2ray-api', label: 'V2Ray API' },
]

// Load data
const fetchCacheFile = async () => {
  loading.value = true
  try {
    const { data } = await experimentalService.getCacheFile()
    cacheFileConfig.value = data
  } catch (err: any) {
    console.error('Failed to fetch cache file config:', err)
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.response?.data?.error || 'Failed to fetch cache file configuration',
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

const fetchClashAPI = async () => {
  loading.value = true
  try {
    const { data } = await experimentalService.getClashAPI()
    clashAPIConfig.value = data
  } catch (err: any) {
    console.error('Failed to fetch Clash API config:', err)
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

const fetchV2RayAPI = async () => {
  loading.value = true
  try {
    const { data } = await experimentalService.getV2RayAPI()
    v2rayAPIConfig.value = data
  } catch (err: any) {
    console.error('Failed to fetch V2Ray API config:', err)
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.response?.data?.error || 'Failed to fetch V2Ray API configuration',
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

// Load data for the active tab
const loadTabData = async (tabId: string) => {
  switch (tabId) {
    case 'cache-file':
      await fetchCacheFile()
      break
    case 'clash-api':
      await fetchClashAPI()
      break
    case 'v2ray-api':
      await fetchV2RayAPI()
      break
  }
}

// Watch for tab changes - removed since we handle loading in handleTabClick

// Handle tab click - will load data even if clicking the same tab
const handleTabClick = async (tabId: string) => {
  if (activeTab.value === tabId) {
    // If clicking the same tab, force reload
    await loadTabData(tabId)
  } else {
    // If different tab, load data first, then change tab
    await loadTabData(tabId)
    activeTab.value = tabId
  }
}

// Update handlers
const handleUpdateCacheFile = async (config: CacheFile) => {
  loading.value = true
  try {
    await experimentalService.updateCacheFile(config)
    toast.add({
      severity: 'success',
      summary: 'Success',
      detail: 'Cache file configuration updated successfully',
      life: 3000
    })
    await fetchCacheFile()
  } catch (err: any) {
    console.error('Failed to update cache file config:', err)
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.response?.data?.error || 'Failed to update cache file configuration',
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

const handleUpdateClashAPI = async (config: ClashAPI) => {
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
    console.error('Failed to update Clash API config:', err)
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

const handleUpdateV2RayAPI = async (config: V2RayAPI) => {
  loading.value = true
  try {
    await experimentalService.updateV2RayAPI(config)
    toast.add({
      severity: 'success',
      summary: 'Success',
      detail: 'V2Ray API configuration updated successfully',
      life: 3000
    })
    await fetchV2RayAPI()
  } catch (err: any) {
    console.error('Failed to update V2Ray API config:', err)
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.response?.data?.error || 'Failed to update V2Ray API configuration',
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

// Load initial tab data on mount
onMounted(() => {
  loadTabData(activeTab.value)
})
</script>

<template>
  <div class="p-8">
    <h2 class="text-3xl font-bold text-gray-900 dark:text-gray-100 mb-6">Experimental Features</h2>

    <!-- Tabs -->
    <div class="mb-6 border-b border-gray-200 dark:border-gray-700">
      <nav class="-mb-px flex space-x-8">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          @click="handleTabClick(tab.id)"
          :class="[
            'py-2 px-1 border-b-2 font-medium text-sm transition-colors',
            activeTab === tab.id
              ? 'border-violet-500 text-violet-600 dark:text-violet-400'
              : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-300'
          ]"
        >
          {{ tab.label }}
        </button>
      </nav>
    </div>

    <!-- Cache File Tab -->
    <CacheFileSettings
      v-if="activeTab === 'cache-file'"
      :cache-file="cacheFileConfig"
      :loading="loading"
      @update="handleUpdateCacheFile"
    />

    <!-- Clash API Tab -->
    <ClashAPISettings
      v-if="activeTab === 'clash-api'"
      :clash-api="clashAPIConfig"
      :loading="loading"
      @update="handleUpdateClashAPI"
    />

    <!-- V2Ray API Tab -->
    <V2RayAPISettings
      v-if="activeTab === 'v2ray-api'"
      :v2ray-api="v2rayAPIConfig"
      :loading="loading"
      @update="handleUpdateV2RayAPI"
    />
  </div>
</template>