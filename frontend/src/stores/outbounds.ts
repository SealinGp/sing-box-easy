import { ref } from 'vue'
import { defineStore } from 'pinia'
import type { Outbound } from '../types/api'
import { outboundService } from '../services'

export const useOutboundsStore = defineStore('outbounds', () => {
  const outbounds = ref<Outbound[]>([])
  const loading = ref(false)

  // Fetch all outbounds
  const fetchOutbounds = async () => {
    loading.value = true
    try {
      const { data } = await outboundService.getOutbounds()
      outbounds.value = data.outbounds
      return data.outbounds
    } catch (err: any) {
      console.error('Failed to fetch outbounds:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  // Add a new outbound
  const addOutbound = async (outbound: Outbound) => {
    loading.value = true
    try {
      await outboundService.addOutbound(outbound)
      await fetchOutbounds()
    } catch (err: any) {
      console.error('Failed to add outbound:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  // Update an existing outbound
  const updateOutbound = async (tag: string, outbound: Outbound) => {
    loading.value = true
    try {
      await outboundService.updateOutbound(tag, outbound)
      await fetchOutbounds()
    } catch (err: any) {
      console.error('Failed to update outbound:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  // Delete an outbound
  const deleteOutbound = async (tagOrIndex: string | number) => {
    loading.value = true
    try {
      await outboundService.deleteOutbound(String(tagOrIndex))
      await fetchOutbounds()
    } catch (err: any) {
      console.error('Failed to delete outbound:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  // Add outbounds in batch
  const addOutboundsBatch = async (outboundsToAdd: Outbound[]) => {
    loading.value = true
    try {
      const result = await outboundService.addOutboundsBatch(outboundsToAdd)
      await fetchOutbounds()
      return result
    } catch (err: any) {
      console.error('Failed to add outbounds batch:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  // Delete outbounds in batch
  const deleteOutboundsBatch = async (tags: string[]) => {
    loading.value = true
    try {
      const result = await outboundService.deleteOutboundsBatch(tags)
      await fetchOutbounds()
      return result
    } catch (err: any) {
      console.error('Failed to delete outbounds batch:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  return {
    outbounds,
    loading,
    fetchOutbounds,
    addOutbound,
    updateOutbound,
    deleteOutbound,
    deleteOutboundsBatch,
    addOutboundsBatch,
  }
})