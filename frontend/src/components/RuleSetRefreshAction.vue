<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowPathIcon } from '@heroicons/vue/24/outline'
import TooltipTarget from './TooltipTarget.vue'
import Button from './Button.vue'
import Popover from '../volt/Popover.vue'
import { routeService } from '../services'
import type { RuleSetRefreshResult } from '../services/route'
import { apiErrorMessage } from '../utils/apiErrorMessage'

defineProps<{ tag: string }>()
const { t } = useI18n()
const panel = ref<InstanceType<typeof Popover>>()
const open = ref(false)
const busy = ref(false)
const result = ref<RuleSetRefreshResult>()
const error = ref('')

async function refresh(tag: string) {
  if (busy.value) return
  busy.value = true
  error.value = ''
  result.value = undefined
  try {
    result.value = await routeService.refreshRuleSet(tag)
  } catch (err) {
    error.value = apiErrorMessage(err, t('route.ruleSets.refresh.failed'))
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <span class="inline-flex">
    <TooltipTarget as="button" type="button" class="icon-action"
      :aria-label="`${t('route.ruleSets.refresh.label')} ${tag}`"
      :text="t('route.ruleSets.refresh.label')" :aria-expanded="open" aria-haspopup="dialog"
      @click="panel?.toggle($event)">
      <ArrowPathIcon class="h-4 w-4" :class="{ 'motion-safe:animate-spin': busy }" aria-hidden="true" />
    </TooltipTarget>
    <Popover ref="panel" :aria-label="t('route.ruleSets.refresh.label')" :dismissable="!busy"
      @show="open = true" @hide="open = false">
      <div class="space-y-3 text-left" :aria-busy="busy">
        <p class="break-words font-mono text-sm font-medium text-text-primary">{{ tag }}</p>
        <p class="text-xs text-text-secondary">{{ t('route.ruleSets.refresh.notice') }}</p>
        <p v-if="busy" role="status" class="text-sm text-text-primary">{{ t('route.ruleSets.refresh.running') }}</p>
        <p v-if="error" role="alert" class="break-words text-sm text-red-600 dark:text-red-400">{{ error }}</p>
        <div v-if="result" role="status" class="text-sm text-text-primary">
          <p>{{ t(result.changed ? 'route.ruleSets.refresh.updated' : 'route.ruleSets.refresh.unchanged') }}</p>
          <p class="mt-1 text-xs text-text-secondary">{{ new Date(result.updated_at_unix * 1000).toLocaleString() }}</p>
        </div>
        <div class="flex justify-end">
          <Button size="sm" :loading="busy" @click="refresh(tag)">{{ t('route.ruleSets.refresh.label') }}</Button>
        </div>
      </div>
    </Popover>
  </span>
</template>
