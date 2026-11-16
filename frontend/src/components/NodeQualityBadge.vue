<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { HandThumbUpIcon, HandThumbDownIcon, MinusCircleIcon, QuestionMarkCircleIcon } from '@heroicons/vue/24/outline'
import TooltipTarget from './TooltipTarget.vue'
import { rateNode } from '../utils/nodeQuality'
import { useNodeQualityStore } from '../stores/nodeQuality'
import type { NodeDelaySample } from '../types/runtime'

const props = defineProps<{ history?: NodeDelaySample[] }>()
const { t } = useI18n()
const store = useNodeQualityStore()
const rating = computed(() => rateNode(store.ready ? props.history : [], store.rules))
const icons = { good: HandThumbUpIcon, fair: MinusCircleIcon, poor: HandThumbDownIcon, unknown: QuestionMarkCircleIcon }
const tones = { good: 'text-green-600 dark:text-green-400', fair: 'text-amber-600 dark:text-amber-400', poor: 'text-red-600 dark:text-red-400', unknown: 'text-text-secondary' }
const description = computed(() => {
  if (!store.ready) return t(store.failed ? 'proxies.quality.unavailable' : 'common.loading')
  if (!rating.value.count) return t('proxies.quality.unknown')
  return t('proxies.quality.summary', {
    level: t(`proxies.quality.${rating.value.level}`), passed: rating.value.passed,
    count: rating.value.count, percent: Math.round(rating.value.percent * 10) / 10,
    ms: store.rules.max_delay_ms,
  })
})
</script>

<template>
  <TooltipTarget as="span" role="img" tabindex="0" :aria-label="description" :text="description"
    class="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-control focus-visible:outline-2 focus-visible:outline-primary-500"
    :class="tones[rating.level]" @click.stop @keydown.enter.stop @keydown.space.stop>
    <component :is="icons[rating.level]" class="h-4 w-4" aria-hidden="true" />
  </TooltipTarget>
</template>
