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
import ActionIcon from './ActionIcon.vue'
import { useNotify } from '../composables/useNotify'
import { writeTextToClipboard } from '../utils/clipboard'

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

const iconStyle = computed(() => ({
  width: props.size === 'xs' ? '16px' : '20px',
  height: props.size === 'xs' ? '16px' : '20px',
}))

const title = computed(() => {
  if (state.value === 'copied') return t('common.copied')
  if (state.value === 'failed') return t('common.copyFailed')
  return props.label || t('common.copy')
})

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
    await writeTextToClipboard(props.text)
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
    class="icon-action copy-action"
    :class="{ 'copy-action-small': size === 'xs' }"
    :data-state="state"
    :title="title"
    :aria-label="title"
    @click.stop.prevent="copy"
  >
    <ActionIcon :action="state === 'copied' ? 'check' : state === 'failed' ? 'close' : 'copy'" :style="iconStyle" />
    <!-- Announced to screen readers, which cannot see the icon swap. -->
    <span class="sr-only" aria-live="polite">
      {{ state === 'idle' ? '' : title }}
    </span>
  </button>
</template>

<style scoped>
.copy-action { width: 26px; height: 26px; }
.copy-action-small { width: 24px; height: 24px; }
.copy-action[data-state="copied"] { color: var(--color-success); }
.copy-action[data-state="failed"] { color: var(--color-danger); }
</style>
