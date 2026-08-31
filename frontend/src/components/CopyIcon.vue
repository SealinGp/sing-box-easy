<script setup lang="ts">
/**
 * A copy button that confirms in place.
 *
 * The confirmation is the whole point. Copying used to raise a global toast in
 * the corner, which puts the feedback somewhere the eye is not: the pointer is
 * on the icon, so that is where the answer has to appear. The icon swaps to a
 * check for `resetMs`, then swaps back — no layout shift, nothing to dismiss,
 * and repeated copies stay legible because each click restarts the timer.
 *
 * Failure is the exception: a copy that did not happen cannot be reported by
 * an icon alone, since the user has no way to tell "copied" from "looked
 * copied". It shows a red cross AND raises the notification, because the
 * recovery ("select the text and press Ctrl+C") is more than an icon can say.
 */
import { computed, onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { CheckIcon, ClipboardDocumentIcon, XMarkIcon } from '@heroicons/vue/24/outline'
import { useNotify } from '../composables/useNotify'

const props = withDefaults(defineProps<Props>(), {
  size: 'sm',
  label: '',
  resetMs: 3000,
})

export interface Props {
  /** The text placed on the clipboard. */
  text: string
  /** Icon size. `xs` suits a dense secondary line, `sm` a table cell. */
  size?: 'xs' | 'sm' | 'md'
  /** Accessible name and tooltip. Falls back to a generic "Copy". */
  label?: string
  /** How long the check stays up, in ms. */
  resetMs?: number
}

const emit = defineEmits<{ (e: 'copied', text: string): void }>()

const { t } = useI18n()
const notify = useNotify()

type State = 'idle' | 'copied' | 'failed'
const state = ref<State>('idle')
let timer: ReturnType<typeof setTimeout> | undefined

// A component can be unmounted while the check is still showing (the row is
// deleted, the modal closes). Without this the timer fires against a dead
// instance, and in tests it keeps the process alive.
onBeforeUnmount(() => clearTimeout(timer))

const iconClass = computed(() => {
  switch (props.size) {
    case 'xs':
      return 'h-3 w-3'
    case 'md':
      return 'h-4.5 w-4.5'
    default:
      return 'h-3.5 w-3.5'
  }
})

const title = computed(() => {
  if (state.value === 'copied') return t('common.copied')
  if (state.value === 'failed') return t('common.copyFailed')
  return props.label || t('common.copy')
})

/**
 * `navigator.clipboard` is undefined outside a secure context, and this panel
 * is routinely served over plain HTTP on a LAN address — so the deprecated
 * `execCommand` path is the one that actually runs for most users here, not a
 * legacy fallback.
 */
async function writeToClipboard(text: string) {
  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(text)
    return
  }

  const textarea = document.createElement('textarea')
  textarea.value = text
  textarea.style.cssText = 'position:fixed;top:0;left:0;opacity:0;pointer-events:none'
  document.body.appendChild(textarea)
  try {
    textarea.focus()
    textarea.select()
    if (!document.execCommand('copy')) {
      throw new Error('execCommand copy failed')
    }
  } finally {
    document.body.removeChild(textarea)
  }
}

function flash(next: State) {
  state.value = next
  clearTimeout(timer)
  timer = setTimeout(() => {
    state.value = 'idle'
  }, props.resetMs)
}

async function copy() {
  if (!props.text) return
  try {
    await writeToClipboard(props.text)
    flash('copied')
    emit('copied', props.text)
  } catch (error) {
    flash('failed')
    notify.apiError(error, t('common.copyFailed'))
  }
}
</script>

<template>
  <button
    type="button"
    class="shrink-0 p-0.5 rounded transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500"
    :class="
      state === 'copied'
        ? 'text-green-600 dark:text-green-400'
        : state === 'failed'
          ? 'text-red-600 dark:text-red-400'
          : 'text-gray-400 hover:text-primary-600 dark:hover:text-primary-400'
    "
    :title="title"
    :aria-label="title"
    @click.stop.prevent="copy"
  >
    <!--
      One <span> per state rather than a :is= swap, so the check inherits the
      button's own box and the row cannot reflow by a pixel as it changes.
    -->
    <CheckIcon v-if="state === 'copied'" :class="iconClass" />
    <XMarkIcon v-else-if="state === 'failed'" :class="iconClass" />
    <ClipboardDocumentIcon v-else :class="iconClass" />
    <!-- Announced to screen readers, which cannot see the icon swap. -->
    <span class="sr-only" aria-live="polite">
      {{ state === 'idle' ? '' : title }}
    </span>
  </button>
</template>
