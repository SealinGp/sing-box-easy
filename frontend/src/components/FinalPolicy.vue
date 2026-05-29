<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import Card from './Card.vue'
import { Select } from '../volt'
import { routeService } from '../services'
import { useToast } from 'primevue'

const toast = useToast()
const { t } = useI18n()

// Local state
const loading = ref(false)
const finalRoute = ref<string>('proxy')

const finalOptions = computed(() => [
  { label: t('route.finalPolicy.options.proxy'), value: 'proxy' },
  { label: t('route.finalPolicy.options.direct'), value: 'direct' },
  { label: t('route.finalPolicy.options.block'), value: 'block' },
  { label: t('route.finalPolicy.options.dns'), value: 'dns' },
])

// Fetch final route
const fetchFinalRoute = async () => {
  loading.value = true
  try {
    const { data } = await routeService.getRouteFinal()
    finalRoute.value = data.final || 'proxy'
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: err.message || t('route.finalPolicy.toast.fetchFailed'),
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
      summary: t('common.success'),
      detail: t('route.finalPolicy.toast.updated'),
      life: 3000
    })
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: err.message || t('route.finalPolicy.toast.updateFailed'),
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
        {{ $t('route.finalPolicy.title') }}
      </h3>
      <p class="text-gray-600 dark:text-gray-400 mb-4">
        {{ $t('route.finalPolicy.description') }}
      </p>

      <div v-if="loading" class="text-center py-4">
        <div class="inline-block animate-spin rounded-full h-6 w-6 border-b-2 border-violet-600"></div>
      </div>

      <div v-else class="flex items-center space-x-4">
        <label class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ $t('route.finalPolicy.label') }}</label>
        <Select
          v-model="finalRoute"
          :options="finalOptions"
          optionLabel="label"
          optionValue="value"
          :placeholder="$t('route.finalPolicy.placeholder')"
          @change="handleUpdate"
          class="w-64"
          :disabled="loading"
        />
      </div>
    </Card>
  </div>
</template>