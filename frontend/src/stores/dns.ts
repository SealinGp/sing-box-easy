import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { DNSServer } from '../types/api'
import { dnsService } from '../services'

export const useDNSStore = defineStore('dns', () => {
  // Shared state - DNS Servers are needed by multiple components
  const dnsServers = ref<DNSServer[]>([])
  const loading = ref(false)

  // Actions for DNS Servers (shared across components)
  const fetchDNSServers = async () => {
    loading.value = true
    try {
      const { data } = await dnsService.getDNSServers()
      dnsServers.value = data.servers || []
      return data.servers
    } catch (err: any) {
      console.error('Failed to fetch DNS servers:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  const addDNSServer = async (server: DNSServer) => {
    loading.value = true
    try {
      await dnsService.addDNSServer(server)
      await fetchDNSServers()
    } catch (err: any) {
      console.error('Failed to add DNS server:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  const updateDNSServer = async (tag: string, server: DNSServer) => {
    loading.value = true
    try {
      await dnsService.updateDNSServer(tag, server)
      await fetchDNSServers()
    } catch (err: any) {
      console.error('Failed to update DNS server:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  const deleteDNSServer = async (tag: string) => {
    loading.value = true
    try {
      await dnsService.deleteDNSServer(tag)
      await fetchDNSServers()
    } catch (err: any) {
      console.error('Failed to delete DNS server:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  return {
    // State
    dnsServers,
    loading,

    // Server actions
    fetchDNSServers,
    addDNSServer,
    updateDNSServer,
    deleteDNSServer
  }
})