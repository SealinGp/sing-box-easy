<script setup lang="ts">
import { computed } from 'vue'
import { XMarkIcon } from '@heroicons/vue/24/outline'
import Dialog from '../volt/Dialog.vue'

/**
 * The application modal.
 *
 * Backed by PrimeVue's unstyled Dialog (themed in `src/volt/Dialog.vue`). It
 * previously used HeadlessUI's Dialog, which meant the app shipped two
 * independent modal implementations — HeadlessUI here and PrimeVue in the
 * `volt` dialogs — with different focus-trap behaviour, different transitions,
 * and different panel styling. PrimeVue won because the volt widgets that also
 * render overlays (Select, MultiSelect, Toast) already depend on it.
 *
 * The prop API is unchanged (`modelValue`, `title`, `size`, `showClose`) so
 * existing call sites keep working.
 */
interface Props {
  modelValue: boolean
  title?: string
  size?: 'sm' | 'md' | 'lg' | 'wide' | 'xl'
  showClose?: boolean
}

const props = withDefaults(defineProps<Props>(), { size: 'md' })

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

// `wide` exists because the outbound add/edit form was authored at max-w-3xl,
// which sits between `lg` and `xl`. Without it that form would have to round to
// one neighbour and either lose ~6rem of width or gain ~8rem.
const SIZE_CLASSES: Record<NonNullable<Props['size']>, string> = {
  sm: 'max-w-md',
  md: 'max-w-lg',
  lg: 'max-w-2xl',
  wide: 'max-w-3xl',
  xl: 'max-w-4xl',
}

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const close = () => {
  visible.value = false
}
</script>

<template>
  <!--
    Size is passed as `class`, not `pt`. volt/Dialog.vue binds `:pt="theme"`
    internally, so an outer `pt` would race with it; `class` instead falls
    through and is reconciled by `ptViewMerge`'s tailwind-merge, letting
    `max-w-md` cleanly win over the theme's default `max-w-lg`.

    Padding comes from the theme's `content` slot — no wrapper needed here.
  -->
  <Dialog
    v-model:visible="visible"
    modal
    dismissable-mask
    :draggable="false"
    :show-header="false"
    :class="SIZE_CLASSES[size]"
    :aria-label="title"
  >
    <div v-if="title || showClose" class="mb-4 flex items-center justify-between">
      <h3 v-if="title" class="text-lg font-semibold text-gray-900 dark:text-gray-100">
        {{ title }}
      </h3>
      <button
        v-if="showClose"
        type="button"
        class="rounded-control p-1 text-gray-400 transition-colors hover:bg-black/5 hover:text-gray-600 dark:text-gray-500 dark:hover:bg-white/10 dark:hover:text-gray-300"
        :aria-label="$t('common.close')"
        @click="close"
      >
        <XMarkIcon class="h-6 w-6" />
      </button>
    </div>

    <div class="text-gray-700 dark:text-gray-300">
      <slot />
    </div>

    <div v-if="$slots.footer" class="mt-6 flex justify-end gap-3">
      <slot name="footer" />
    </div>
  </Dialog>
</template>
