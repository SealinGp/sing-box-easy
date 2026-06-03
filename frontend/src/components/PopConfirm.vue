<script setup lang="ts">
import { Popover, PopoverButton, PopoverPanel } from '@headlessui/vue'
import { useI18n } from 'vue-i18n'

/**
 * Anchored confirmation popover (HeadlessUI Popover). Wraps a trigger button and
 * shows a small confirm/cancel panel next to it — a lightweight alternative to
 * the global modal ConfirmDialog for inline, low-friction destructive actions.
 *
 * Usage:
 *   <PopConfirm :message="..." :confirm-label="..." tone="danger" @confirm="doDelete()">
 *     Delete
 *   </PopConfirm>
 */
withDefaults(
  defineProps<{
    message: string
    confirmLabel?: string
    cancelLabel?: string
    tone?: 'danger' | 'primary'
    /** Classes for the trigger button (defaults to a ghost danger link button). */
    triggerClass?: string
    /** Which side of the trigger the panel opens toward. */
    align?: 'left' | 'right'
  }>(),
  { tone: 'danger', align: 'right' },
)

const emit = defineEmits<{ (e: 'confirm'): void }>()
const { t } = useI18n()
</script>

<template>
  <Popover class="relative inline-block">
    <PopoverButton :class="triggerClass ?? 'btn btn-xs btn-ghost text-error'">
      <slot>{{ t('common.delete') }}</slot>
    </PopoverButton>

    <transition
      enter-active-class="transition duration-100 ease-out"
      enter-from-class="opacity-0 translate-y-1"
      enter-to-class="opacity-100 translate-y-0"
      leave-active-class="transition duration-75 ease-in"
      leave-from-class="opacity-100 translate-y-0"
      leave-to-class="opacity-0 translate-y-1"
    >
      <PopoverPanel
        v-slot="{ close }"
        class="absolute z-50 mt-1 w-64 rounded-lg border border-base-300 bg-base-100 p-3 shadow-lg"
        :class="align === 'right' ? 'right-0' : 'left-0'"
      >
        <p class="text-sm text-gray-700 dark:text-gray-300 whitespace-pre-line">{{ message }}</p>
        <div class="flex justify-end gap-2 mt-3">
          <button class="btn btn-xs btn-ghost" @click="close()">
            {{ cancelLabel ?? t('common.cancel') }}
          </button>
          <button
            class="btn btn-xs"
            :class="tone === 'danger' ? 'btn-error' : 'btn-primary'"
            @click="emit('confirm'); close()"
          >
            {{ confirmLabel ?? t('common.confirm') }}
          </button>
        </div>
      </PopoverPanel>
    </transition>
  </Popover>
</template>
