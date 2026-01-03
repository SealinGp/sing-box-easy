<script setup lang="ts">
import { ref, onMounted } from 'vue'
import Card from './Card.vue'
import { Select } from '../volt'
import { routeService } from '../services'
import { useToast } from 'primevue'

const toast = useToast()

// Local state
const loading = ref(false)
const finalRoute = ref<string>('proxy')

const finalOptions = [
  { label: 'Proxy', value: 'proxy' },
  { label: 'Direct', value: 'direct' },
  { label: 'Block', value: 'block' },
  { label: 'DNS', value: 'dns' },
]

// Fetch final route
const fetchFinalRoute = async () => {
  loading.value = true
  try {
    const { data } = await routeService.getRouteFinal()
    finalRoute.value = data.final || 'proxy'
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.response?.data?.error || 'Failed to fetch final route',
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

// Update final route
const handleUpdate = async () => {
  loading.value = true
  try {
    await routeService.updateRouteFinal(finalRoute.value)
    toast.add({
      severity: 'success',
      summary: 'Success',
      detail: 'Final route policy updated successfully',
      life: 3000
    })
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.response?.data?.error || 'Failed to update final route',
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

// Load data on mount
onMounted(() => {
  fetchFinalRoute()
})
</script>

<template>
  <div class="space-y-6">
    <Card>
      <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">
        Final Route Policy
      </h3>
      <p class="text-gray-600 dark:text-gray-400 mb-4">
        Configure the default outbound for traffic that doesn't match any specific routing rules.
      </p>

      <div v-if="loading" class="text-center py-4">
        <div class="inline-block animate-spin rounded-full h-6 w-6 border-b-2 border-violet-600"></div>
      </div>

      <div v-else class="flex items-center space-x-4">
        <label class="text-sm font-medium text-gray-700 dark:text-gray-300">Final Policy:</label>
        <Select
          v-model="finalRoute"
          :options="finalOptions"
          optionLabel="label"
          optionValue="value"
          placeholder="Select final policy"
          @change="handleUpdate"
          class="w-64"
          :disabled="loading"
        />
      </div>
    </Card>
  </div>
</template>