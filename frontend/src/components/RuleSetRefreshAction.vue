<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowPathIcon } from '@heroicons/vue/24/outline'
import TooltipTarget from './TooltipTarget.vue'
import Button from './Button.vue'
import Popover from '../volt/Popover.vue'
import { routeService } from '../services'
import type { RuleSetRefreshResult } from '../services/route'
import { apiErrorMessage } from '../utils/apiErrorMessage'

const props = defineProps<{ tag?: string; tags?: string[]; disabled?: boolean }>()
const emit = defineEmits<{ busy: [busy: boolean] }>()
const { t } = useI18n()
const panel = ref<InstanceType<typeof Popover>>()
const open = ref(false)
const busy = ref(false)
const results = ref<RuleSetRefreshResult[]>([])
const error = ref('')
const targets = computed(() => props.tags ?? (props.tag ? [props.tag] : []))
const label = computed(() => t(props.tags ? 'route.ruleSets.refresh.batchLabel' : 'route.ruleSets.refresh.label'))
watch(() => targets.value.join('\u0000'), () => { results.value = []; error.value = '' })

async function refresh() {
  if (busy.value || props.disabled || !targets.value.length) return
  const snapshot = [...targets.value]
  busy.value = true
  emit('busy', true)
  error.value = ''
  results.value = []
  try {
    results.value = props.tags ? await routeService.refreshRuleSets(snapshot) : [await routeService.refreshRuleSet(snapshot[0]!)]
  } catch (err) {
    error.value = apiErrorMessage(err, t('route.ruleSets.refresh.failed'))
  } finally {
    busy.value = false
    emit('busy', false)
  }
}
</script>

<template>
  <span class="inline-flex">
    <TooltipTarget as="button" type="button" class="icon-action"
      :aria-label="props.tags ? label : `${label} ${tag}`" :disabled="disabled"
      :text="label" :aria-expanded="open" aria-haspopup="dialog"
      :class="{ 'selection-batch-action': props.tags }"
      @click="panel?.toggle($event)">
      <ArrowPathIcon class="h-4 w-4" :class="{ 'motion-safe:animate-spin': busy }" aria-hidden="true" />
      <span v-if="props.tags">{{ label }}</span>
    </TooltipTarget>
    <Popover ref="panel" :aria-label="t('route.ruleSets.refresh.label')" :dismissable="!busy"
      @show="open = true" @hide="open = false">
      <div class="space-y-3 text-left" :aria-busy="busy">
        <ul class="max-h-32 overflow-y-auto break-words font-mono text-sm font-medium text-text-primary"><li v-for="target in targets" :key="target">{{ target }}</li></ul>
        <p class="text-xs text-text-secondary">{{ t('route.ruleSets.refresh.notice') }}</p>
        <p v-if="props.tags" class="text-xs text-text-secondary">{{ t('route.ruleSets.refresh.batchNotice') }}</p>
        <p v-if="busy" role="status" class="text-sm text-text-primary">{{ t('route.ruleSets.refresh.running') }}</p>
        <p v-if="error" role="alert" class="break-words text-sm text-red-600 dark:text-red-400">{{ error }}</p>
        <div v-if="results.length" role="status" class="max-h-52 space-y-2 overflow-y-auto text-sm text-text-primary">
          <div v-for="result in results" :key="result.tag">
            <p v-if="props.tags" class="break-words font-medium">{{ result.tag }}</p>
            <p>{{ t(result.changed ? 'route.ruleSets.refresh.updated' : 'route.ruleSets.refresh.unchanged') }}</p>
            <p class="mt-1 text-xs text-text-secondary">{{ new Date(result.updated_at_unix * 1000).toLocaleString() }}</p>
          </div>
        </div>
        <div class="flex justify-end">
          <Button size="sm" :loading="busy" :disabled="disabled || !targets.length" @click="refresh">{{ label }}</Button>
        </div>
      </div>
    </Popover>
  </span>
</template>

<style scoped>
.selection-batch-action { width: auto; gap: 8px; padding: 6px 12px; font-size: 14px; font-weight: 600; }
</style>
