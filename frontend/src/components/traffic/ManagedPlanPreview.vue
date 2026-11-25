<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Button from '../Button.vue'
import Dialog from '../../volt/Dialog.vue'
import MonacoDiffEditor from '../MonacoDiffEditor.vue'
import type { ManagedPlan } from '../../services/trafficPolicy'

defineProps<{ plan: ManagedPlan; busy: boolean; error?: string }>()
const emit = defineEmits<{ apply: []; back: [] }>()
const { t } = useI18n()
const serialized = (value: unknown) => JSON.stringify(value, null, 2)
</script>

<template>
  <Dialog :visible="true" modal class="w-full max-w-6xl" :header="t('managed.diffTitle')"
    :closable="!busy" :close-on-escape="!busy" @update:visible="(value: boolean) => { if (!value && !busy) emit('back') }">
    <p class="mb-3 text-sm">{{ t('managed.affected') }} · {{ plan.affected.join(', ') || t('managed.metadataOnly') }}</p>
    <div class="flex justify-between gap-4 text-xs text-text-secondary">
      <span>{{ t('managed.before') }}</span><span>{{ t('managed.after') }}</span>
    </div>
    <div class="mt-2 h-[60vh] min-h-[300px]">
      <MonacoDiffEditor :original="serialized(plan.before)" :modified="serialized(plan.after)" />
    </div>
    <p class="mt-3 text-xs text-text-secondary">{{ t('managed.validation') }}</p>
    <p v-if="error" role="alert" class="mt-2 text-sm text-danger">{{ error }}</p>
    <template #footer>
      <Button variant="ghost" :disabled="busy" @click="emit('back')">{{ t('managed.back') }}</Button>
      <Button :loading="busy" @click="emit('apply')">{{ t('managed.apply') }}</Button>
    </template>
  </Dialog>
</template>
