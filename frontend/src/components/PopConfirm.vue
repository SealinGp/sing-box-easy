<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Button from './Button.vue'

/**
 * Anchored confirmation popover. Wraps a trigger button and shows a small
 * confirm/cancel panel next to it — the standard pattern for inline
 * destructive actions, in place of a full modal.
 *
 * Why anchored rather than a centre-screen dialog: a modal detaches the
 * question from the row it applies to, so "Delete this rule?" gives the user
 * no way to check WHICH rule they clicked. The popover stays next to the
 * trigger and names the item via `target`, so the answer is on screen.
 *
 * Implemented without a headless UI library: the interaction surface here
 * (toggle, click-outside, Escape, focus return) is small enough that a second
 * component library was not worth the vocabulary split.
 *
 * Usage:
 *   <PopConfirm
 *     :message="t('route.ruleSets.confirm.delete')"
 *     :target="ruleSet.tag"
 *     tone="danger"
 *     @confirm="doDelete()"
 *   >Delete</PopConfirm>
 *
 * For a confirmation whose text depends on a server lookup, listen for `open`
 * and drive `details` + `loading` from the parent; the confirm button stays
 * disabled until the lookup settles.
 */
const props = withDefaults(
  defineProps<{
    /** Primary question, e.g. "Delete this rule set?" */
    message: string
    /**
     * The item being acted on, rendered as a prominent monospace chip.
     * This is what tells the user exactly what they are about to destroy.
     */
    target?: string
    /**
     * Optional secondary line — consequences, cascade summaries, warnings.
     * Safe to populate asynchronously after `open` fires.
     */
    details?: string
    /** Blocks confirm and shows a spinner while the parent resolves `details`. */
    loading?: boolean
    confirmLabel?: string
    cancelLabel?: string
    /** Text shown beside the spinner while `loading`. */
    loadingLabel?: string
    tone?: 'danger' | 'primary'
    /** Classes for the trigger button. Overrides the default ghost styling. */
    triggerClass?: string
    /** Which side of the trigger the panel opens toward. */
    align?: 'left' | 'right'
  }>(),
  { tone: 'danger', align: 'right', loading: false },
)

const emit = defineEmits<{
  (e: 'confirm'): void
  /** Fires each time the panel opens — the hook for lazy/async `details`. */
  (e: 'open'): void
}>()

const { t } = useI18n()

const open = ref(false)
const root = ref<HTMLElement | null>(null)
const triggerEl = ref<HTMLElement | null>(null)
const confirmEl = ref<InstanceType<typeof Button> | null>(null)

const close = () => {
  open.value = false
}

const onDocumentPointerDown = (event: PointerEvent) => {
  if (root.value && !root.value.contains(event.target as Node)) close()
}

const onDocumentKeydown = (event: KeyboardEvent) => {
  if (event.key !== 'Escape') return
  close()
  // Return focus to the trigger so keyboard users are not dropped at the top
  // of the document after dismissing.
  triggerEl.value?.focus()
}

// Listeners exist only while the panel is open, so a page full of PopConfirms
// does not attach a document listener per instance.
watch(open, (isOpen) => {
  if (isOpen) {
    document.addEventListener('pointerdown', onDocumentPointerDown)
    document.addEventListener('keydown', onDocumentKeydown)
    emit('open')
    // Move focus into the panel so Enter confirms and Tab stays local.
    nextTick(() => (confirmEl.value?.$el as HTMLElement | undefined)?.focus())
  } else {
    document.removeEventListener('pointerdown', onDocumentPointerDown)
    document.removeEventListener('keydown', onDocumentKeydown)
  }
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', onDocumentPointerDown)
  document.removeEventListener('keydown', onDocumentKeydown)
})

const confirm = () => {
  if (props.loading) return
  emit('confirm')
  close()
}
</script>

<template>
  <div ref="root" class="relative inline-block">
    <button
      ref="triggerEl"
      type="button"
      :class="
        triggerClass ??
        'inline-flex items-center rounded-control px-2 py-1 text-xs font-semibold text-danger transition-colors hover:bg-danger/10 focus:outline-none focus-visible:ring-2 focus-visible:ring-danger'
      "
      :aria-expanded="open"
      aria-haspopup="dialog"
      @click="open = !open"
    >
      <slot>{{ t('common.delete') }}</slot>
    </button>

    <transition
      enter-active-class="transition duration-100 ease-out"
      enter-from-class="opacity-0 translate-y-1"
      enter-to-class="opacity-100 translate-y-0"
      leave-active-class="transition duration-75 ease-in"
      leave-from-class="opacity-100 translate-y-0"
      leave-to-class="opacity-0 translate-y-1"
    >
      <div
        v-if="open"
        role="dialog"
        :aria-busy="loading"
        class="liquid-glass-float absolute z-50 mt-1 w-80 max-w-[calc(100vw-2rem)] rounded-surface p-3 text-left"
        :class="align === 'right' ? 'right-0' : 'left-0'"
      >
        <p class="whitespace-pre-line text-sm text-gray-700 dark:text-gray-300">{{ message }}</p>

        <!--
          The item under the knife.
          Wraps rather than truncates: this string is the entire reason the
          popover exists, so clipping it to one line would hide exactly the
          detail the user opened the panel to check. `break-words` keeps long
          unbroken tags inside the panel instead of overflowing it.
        -->
        <p
          v-if="target"
          class="mt-2 max-h-24 overflow-y-auto rounded-control bg-black/5 px-2 py-1 font-mono text-sm font-semibold break-words text-gray-900 dark:bg-white/10 dark:text-gray-100"
          :title="target"
        >
          {{ target }}
        </p>

        <div v-if="loading" class="mt-2 flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
          <span class="h-3 w-3 animate-spin rounded-pill border-b-2 border-current" aria-hidden="true"></span>
          {{ loadingLabel ?? t('common.loading') }}
        </div>
        <p
          v-else-if="details"
          class="mt-2 whitespace-pre-line text-xs text-gray-500 dark:text-gray-400"
        >
          {{ details }}
        </p>

        <div class="mt-3 flex justify-end gap-2">
          <Button variant="ghost" size="sm" action @click="close">
            {{ cancelLabel ?? t('common.cancel') }}
          </Button>
          <Button
            ref="confirmEl"
            :variant="tone === 'danger' ? 'danger' : 'primary'"
            size="sm"
            :disabled="loading"
            @click="confirm"
          >
            {{ confirmLabel ?? t('common.confirm') }}
          </Button>
        </div>
      </div>
    </transition>
  </div>
</template>
