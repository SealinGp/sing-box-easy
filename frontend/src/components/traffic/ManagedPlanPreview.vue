<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Button from '../Button.vue'
import CopyIcon from '../CopyIcon.vue'
import type { ManagedPlan } from '../../services/trafficPolicy'

defineProps<{ plan: ManagedPlan; busy: boolean }>()
defineEmits<{ apply: []; back: [] }>()
const { t } = useI18n()
const serialized = (value: unknown) => JSON.stringify(value, null, 2)
</script>

<template>
  <section class="min-w-0 space-y-3 rounded-control border border-border bg-bg-secondary p-4" :aria-label="t('managed.proposed')">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <h2 class="font-semibold">{{ t('managed.proposed') }}</h2>
      <div class="flex gap-2">
        <Button variant="ghost" :disabled="busy" @click="$emit('back')">{{ t('managed.back') }}</Button>
        <Button :loading="busy" @click="$emit('apply')">{{ t('managed.apply') }}</Button>
      </div>
    </div>
    <p class="text-sm">{{ t('managed.affected') }}: {{ plan.affected.join(', ') || t('managed.metadataOnly') }}</p>
    <p class="text-xs text-text-secondary">{{ t('managed.validation') }}</p>
    <details open>
      <summary class="cursor-pointer text-sm font-medium">{{ t('managed.after') }}</summary>
      <div class="mt-2 flex justify-end"><CopyIcon :text="serialized(plan.after)" /></div>
      <pre class="max-h-80 overflow-auto text-xs">{{ serialized(plan.after) }}</pre>
    </details>
    <details>
      <summary class="cursor-pointer text-sm font-medium">{{ t('managed.before') }}</summary>
      <pre class="mt-2 max-h-80 overflow-auto text-xs">{{ serialized(plan.before) }}</pre>
    </details>
  </section>
</template>
