<script setup lang="ts">
import Card from './Card.vue'
import { Select } from '../volt'

interface Props {
  finalRoute: string
}

interface Emits {
  (e: 'update', value: string): void
}

defineProps<Props>()
const emit = defineEmits<Emits>()

const finalRoute = defineModel<string>('finalRoute')

const finalOptions = [
  { label: 'Proxy', value: 'proxy' },
  { label: 'Direct', value: 'direct' },
  { label: 'Block', value: 'block' },
  { label: 'DNS', value: 'dns' },
]

function handleUpdate() {
  if (finalRoute.value) {
    emit('update', finalRoute.value)
  }
}
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
      <div class="flex items-center space-x-4">
        <label class="text-sm font-medium text-gray-700 dark:text-gray-300">Final Policy:</label>
        <Select
          v-model="finalRoute"
          :options="finalOptions"
          optionLabel="label"
          optionValue="value"
          placeholder="Select final policy"
          @change="handleUpdate"
          class="w-64"
        />
      </div>
    </Card>
  </div>
</template>
