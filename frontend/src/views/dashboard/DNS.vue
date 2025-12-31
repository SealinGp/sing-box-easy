<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import type { DNSServer, DNS, DNSRule } from '../../types/api'
import DNSServers from '../../components/DNSServers.vue'
import DNSRules from '../../components/DNSRules.vue'
import DNSSettings from '../../components/DNSSettings.vue'
import { dnsService } from '../../services'
import { useToast } from 'primevue'

const toast = useToast()

// State
const loading = ref(false)
const activeTab = ref('servers')

// Data
const dnsServers = ref<DNSServer[]>([])
const dnsRules = ref<DNSRule[]>([])
const dnsConfig = ref<DNS | null>(null)

// Tabs
const tabs = [
  { id: 'servers', label: 'DNS Servers' },
  { id: 'rules', label: 'DNS Rules' },
  { id: 'settings', label: 'Settings' },
]

// Load data
const fetchDNSServers = async () => {
  loading.value = true
  try {
    const { data } = await dnsService.getDNSServers()
    dnsServers.value = data.servers
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.response?.data?.error || 'Failed to fetch DNS servers',
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

const fetchDNSRules = async () => {
  try {
    const { data } = await dnsService.getDNSRules()
    dnsRules.value = data.rules || []
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.response?.data?.error || 'Failed to fetch DNS rules',
      life: 3000
    })
  }
}


const fetchDNSConfig = async () => {
  try {
    const { data } = await dnsService.getDNS()
    dnsConfig.value = data
  } catch (err: any) {
    console.error('Failed to fetch DNS config:', err)
  }
}

// Load data for the active tab
const loadTabData = async (tabId: string) => {
  switch (tabId) {
    case 'servers':
      await fetchDNSServers()
      break
    case 'rules':
      await fetchDNSRules()
      break
    case 'settings':
      await fetchDNSConfig()
      break
  }
}

// Watch for tab changes and load data
watch(activeTab, async (newTab) => {
  // Load data when tab changes
  await loadTabData(newTab)
})

// Handle tab click - will load data even if clicking the same tab
const handleTabClick = async (tabId: string) => {
  if (activeTab.value === tabId) {
    // If clicking the same tab, force reload
    await loadTabData(tabId)
  } else {
    // If different tab, just change activeTab and let the watch handle loading
    activeTab.value = tabId
  }
}

// DNS Server handlers
const handleAddServer = async (server: DNSServer) => {
  loading.value = true
  try {
    await dnsService.addDNSServer(server)
    toast.add({
      severity: 'success',
      summary: 'Success',
      detail: 'DNS server added successfully',
      life: 3000
    })
    await fetchDNSServers()
  } catch (err: any) {
    console.error('Failed to save DNS server:', err)
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.response?.data?.error || 'Failed to save DNS server',
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

const handleUpdateServer = async (tag: string, server: DNSServer) => {
  loading.value = true
  try {
    await dnsService.updateDNSServer(tag, server)
    toast.add({
      severity: 'success',
      summary: 'Success',
      detail: 'DNS server updated successfully',
      life: 3000
    })
    await fetchDNSServers()
  } catch (err: any) {
    console.error('Failed to save DNS server:', err)
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.response?.data?.error || 'Failed to save DNS server',
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

const handleDeleteServer = async (tag: string) => {
  loading.value = true
  try {
    await dnsService.deleteDNSServer(tag)
    toast.add({
      severity: 'success',
      summary: 'Success',
      detail: 'DNS server deleted successfully',
      life: 3000
    })
    await fetchDNSServers()
  } catch (err: any) {
    console.error('Failed to delete DNS server:', err)
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.response?.data?.error || 'Failed to delete DNS server',
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

// DNS Rule handlers
const handleAddRule = async (rule: DNSRule) => {
  loading.value = true
  try {
    // Convert comma-separated strings to arrays
    if (typeof (rule as any).domain === 'string') {
      (rule as any).domain = (rule as any).domain.split(',').map((s: string) => s.trim()).filter(Boolean)
    }
    if (typeof (rule as any).domain_suffix === 'string') {
      (rule as any).domain_suffix = (rule as any).domain_suffix.split(',').map((s: string) => s.trim()).filter(Boolean)
    }
    if (typeof (rule as any).domain_keyword === 'string') {
      (rule as any).domain_keyword = (rule as any).domain_keyword.split(',').map((s: string) => s.trim()).filter(Boolean)
    }
    if (typeof (rule as any).geosite === 'string') {
      (rule as any).geosite = (rule as any).geosite.split(',').map((s: string) => s.trim()).filter(Boolean)
    }

    await dnsService.addDNSRule(rule)
    toast.add({
      severity: 'success',
      summary: 'Success',
      detail: 'DNS rule added successfully',
      life: 3000
    })
    await fetchDNSRules()
  } catch (err: any) {
    console.error('Failed to add DNS rule:', err)
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.response?.data?.error || 'Failed to add DNS rule',
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

const handleUpdateRule = async (index: number, rule: DNSRule) => {
  loading.value = true
  try {
    // Convert comma-separated strings to arrays
    if (typeof (rule as any).domain === 'string') {
      (rule as any).domain = (rule as any).domain.split(',').map((s: string) => s.trim()).filter(Boolean)
    }
    if (typeof (rule as any).domain_suffix === 'string') {
      (rule as any).domain_suffix = (rule as any).domain_suffix.split(',').map((s: string) => s.trim()).filter(Boolean)
    }
    if (typeof (rule as any).domain_keyword === 'string') {
      (rule as any).domain_keyword = (rule as any).domain_keyword.split(',').map((s: string) => s.trim()).filter(Boolean)
    }
    if (typeof (rule as any).geosite === 'string') {
      (rule as any).geosite = (rule as any).geosite.split(',').map((s: string) => s.trim()).filter(Boolean)
    }

    await dnsService.updateDNSRule(index, rule)
    toast.add({
      severity: 'success',
      summary: 'Success',
      detail: 'DNS rule updated successfully',
      life: 3000
    })
    await fetchDNSRules()
  } catch (err: any) {
    console.error('Failed to update DNS rule:', err)
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.response?.data?.error || 'Failed to update DNS rule',
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

const handleDeleteRule = async (index: number) => {
  loading.value = true
  try {
    await dnsService.deleteDNSRule(index)
    toast.add({
      severity: 'success',
      summary: 'Success',
      detail: 'DNS rule deleted successfully',
      life: 3000
    })
    await fetchDNSRules()
  } catch (err: any) {
    console.error('Failed to delete DNS rule:', err)
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.response?.data?.error || 'Failed to delete DNS rule',
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

// DNS Settings handler
const handleUpdateSettings = async (dns: DNS) => {
  loading.value = true
  try {
    await dnsService.updateDNS(dns)
    toast.add({
      severity: 'success',
      summary: 'Success',
      detail: 'DNS settings updated successfully',
      life: 3000
    })
    await fetchDNSConfig()
  } catch (err: any) {
    console.error('Failed to update DNS settings:', err)
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.response?.data?.error || 'Failed to update DNS settings',
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
    <h2 class="text-3xl font-bold text-gray-900 dark:text-gray-100 mb-6">DNS Configuration</h2>

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

    <!-- DNS Servers Tab -->
    <DNSServers
      v-if="activeTab === 'servers'"
      :servers="dnsServers"
      :loading="loading"
      @add-server="handleAddServer"
      @update-server="handleUpdateServer"
      @delete-server="handleDeleteServer"
    />

    <!-- DNS Rules Tab -->
    <DNSRules
      v-if="activeTab === 'rules'"
      :rules="dnsRules"
      :servers="dnsServers"
      :loading="loading"
      @add-rule="handleAddRule"
      @update-rule="handleUpdateRule"
      @delete-rule="handleDeleteRule"
    />

    <!-- DNS Settings Tab -->
    <DNSSettings
      v-if="activeTab === 'settings'"
      :dns-config="dnsConfig"
      :servers="dnsServers"
      :loading="loading"
      @update="handleUpdateSettings"
    />
  </div>
</template>
