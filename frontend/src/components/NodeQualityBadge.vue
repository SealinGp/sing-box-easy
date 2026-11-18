<script setup lang="ts">
import { computed } from 'vue'
import { HandThumbUpIcon, HandThumbDownIcon, MinusCircleIcon, QuestionMarkCircleIcon } from '@heroicons/vue/24/outline'
import { useNodeQuality } from '../composables/useNodeQuality'
import type { NodeDelaySample } from '../types/runtime'

const props = defineProps<{ history?: NodeDelaySample[]; actionLabel?: string; expanded?: boolean }>()
const { rating, description } = useNodeQuality(() => props.history)
const icons = { good: HandThumbUpIcon, fair: MinusCircleIcon, poor: HandThumbDownIcon, unknown: QuestionMarkCircleIcon }
const tones = { good: 'text-green-600 dark:text-green-400', fair: 'text-amber-600 dark:text-amber-400', poor: 'text-red-600 dark:text-red-400', unknown: 'text-text-secondary' }
const label = computed(() => props.actionLabel ? `${description.value} · ${props.actionLabel}` : description.value)
</script>

<template>
  <span :role="actionLabel ? 'button' : 'img'" tabindex="0" :aria-label="label"
    :aria-expanded="actionLabel ? expanded : undefined" :aria-haspopup="actionLabel ? 'dialog' : undefined"
    class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-control focus-visible:outline-2 focus-visible:outline-primary-500"
    :class="[tones[rating.level], actionLabel ? 'cursor-pointer hover:bg-bg-tertiary' : '']"
    @click.stop @keydown.enter.stop @keydown.space.stop>
    <component :is="icons[rating.level]" class="h-4 w-4" aria-hidden="true" />
  </span>
</template>
