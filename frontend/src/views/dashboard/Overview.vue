<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { apiService } from '../../services/api'
import type { ServiceStatus } from '../../types/api'

const status = ref<ServiceStatus | null>(null)
const loading = ref(false)

onMounted(async () => {
  loading.value = true
  try {
    status.value = await apiService.getServiceStatus()
  } catch (error) {
    console.error('Failed to get service status:', error)
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="p-8">
    <h2 class="text-3xl font-bold text-gray-900 mb-6">Dashboard Overview</h2>

    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
      <div class="bg-white p-6 rounded-lg shadow">
        <h3 class="text-lg font-semibold text-gray-700 mb-2">Service Status</h3>
        <p v-if="loading" class="text-gray-500">Loading...</p>
        <p v-else-if="status" :class="status.status === 'running' ? 'text-green-600' : 'text-red-600'" class="text-2xl font-bold">
          {{ status.status }}
        </p>
      </div>
    </div>
  </div>
</template>
