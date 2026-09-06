<script setup lang="ts">
/**
 * A progress bar split into discrete blocks — Ant Design's `<Progress steps>`.
 *
 * Worth having as its own primitive because some quantities ARE discrete. A
 * continuous bar for "36 of 37 nodes answered" draws a quantity that was never
 * measured continuously; blocks say "some of a countable set", which is what
 * the number means.
 *
 * TWO WAYS TO COLOUR IT, both from Ant's API
 * ──────────────────────────────────────────
 *   strokeColor="bg-green-500"                  every filled block, one colour
 *   strokeColor="['bg-green-500','bg-red-500']" block 0 green, block 1 red
 *
 * The array form is what makes this useful for COMPOSITION rather than
 * progress: pass `percent="100"` so every block renders filled, and let the
 * colours carry the breakdown. A shorter array than `steps` reuses its last
 * entry, as Ant does.
 *
 * Colours are Tailwind classes by default so they follow the dark-mode tokens
 * the rest of the app uses. A raw CSS colour (`#0a0`, `rgb(...)`, `var(--x)`)
 * is also accepted and applied inline, for callers that have a computed value
 * rather than a class.
 */
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    /** 0-100. Drives how many blocks are filled. */
    percent?: number
    /** How many blocks to draw. */
    steps?: number
    /** One colour for every filled block, or one per block. */
    strokeColor?: string | string[]
    /** Colour of the unfilled remainder. */
    trailColor?: string
    size?: 'xs' | 'sm' | 'md'
    /** Renders the percentage after the blocks. Off by default: most callers
     *  already show a richer figure of their own ("97% (36/37)"). */
    showInfo?: boolean
    /** Replaces the percentage text when showInfo is on. */
    label?: string
    /** Screen-reader description. The blocks are decorative without one. */
    ariaLabel?: string
  }>(),
  {
    percent: 100,
    steps: 5,
    strokeColor: 'bg-primary-600',
    trailColor: 'bg-gray-200 dark:bg-gray-700',
    size: 'sm',
    showInfo: false,
    label: undefined,
    ariaLabel: undefined,
  },
)

const clampedPercent = computed(() => {
  if (!Number.isFinite(props.percent)) return 0
  return Math.min(100, Math.max(0, props.percent))
})

/**
 * How many blocks are filled.
 *
 * Rounded, matching Ant. Callers for whom rounding up to "complete" would be a
 * lie (a health gauge where 97% must not read as flawless) should pass
 * percent=100 and encode the shortfall in the colour array instead — see the
 * composition note above.
 */
const filled = computed(() => Math.round(props.steps * (clampedPercent.value / 100)))

const SIZE_CLASS = {
  xs: 'h-1.5 w-2.5',
  sm: 'h-2 w-3.5',
  md: 'h-2.5 w-5',
} as const

/** A raw CSS colour goes in `style`; anything else is a class list. */
function isCssColor(value: string): boolean {
  return /^(#|rgb|hsl|var\(|transparent$|currentColor$)/i.test(value.trim())
}

interface Block {
  filled: boolean
  class: string
  style: Record<string, string>
}

const blocks = computed<Block[]>(() => {
  const palette = Array.isArray(props.strokeColor) ? props.strokeColor : null

  return Array.from({ length: Math.max(0, props.steps) }, (_, index) => {
    const isFilled = index < filled.value
    // A short array reuses its last entry, so ['green'] fills everything green.
    // An EMPTY array falls through to the trail rather than to no colour at
    // all: callers use it to mean "nothing to show" (qualityStepColors returns
    // it when no node was measured), and blocks with no background would still
    // occupy their space as invisible gaps.
    const raw = isFilled
      ? palette
        ? (palette[index] ?? palette[palette.length - 1] ?? props.trailColor)
        : (props.strokeColor as string)
      : props.trailColor

    const value = raw ?? ''
    const inline = isCssColor(value)
    const style: Record<string, string> = inline ? { backgroundColor: value } : {}
    return {
      filled: isFilled,
      class: inline ? '' : value,
      style,
    }
  })
})

const infoText = computed(() => props.label ?? `${Math.round(clampedPercent.value)}%`)
</script>

<template>
  <span
    class="inline-flex items-center gap-1"
    role="progressbar"
    :aria-valuenow="Math.round(clampedPercent)"
    aria-valuemin="0"
    aria-valuemax="100"
    :aria-label="ariaLabel"
  >
    <span class="inline-flex items-center gap-0.5">
      <!--
        aria-hidden: the blocks are a picture of the value the wrapper already
        announces. Without this a screen reader reads five anonymous elements.
      -->
      <span
        v-for="(block, index) in blocks"
        :key="index"
        aria-hidden="true"
        class="inline-block rounded-[1px] transition-colors duration-200"
        :class="[SIZE_CLASS[size], block.class]"
        :style="block.style"
      ></span>
    </span>
    <span v-if="showInfo" class="text-xs tabular-nums text-gray-500 dark:text-gray-400">
      {{ infoText }}
    </span>
  </span>
</template>
