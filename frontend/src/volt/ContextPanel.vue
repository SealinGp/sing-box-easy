<script setup lang="ts">
import { computed, nextTick, ref, useAttrs, useId, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { twMerge } from 'tailwind-merge'
import type { DialogProps } from 'primevue/dialog'
import Dialog from './Dialog.vue'
import Popover from './Popover.vue'
import { useOverlayAnchor } from '../composables/useOverlayAnchor'

// Contextual forms keep the header/footer API of Dialog, but open at their
// initiating control. An invocation without a valid anchor has a dialog fallback.
interface Props extends /* @vue-ignore */ DialogProps {
  visible?: boolean
  header?: string
  closable?: boolean
  dismissable?: boolean
}
defineOptions({ inheritAttrs: false })
const props = withDefaults(defineProps<Props>(), { visible: false, closable: true, dismissable: true })
const emit = defineEmits<{ 'update:visible': [boolean]; hide: []; show: [] }>()
const attrs = useAttrs()
const titleId = useId()
const { t } = useI18n()
const findAnchor = useOverlayAnchor()
const anchored = ref(false)
const popover = ref<InstanceType<typeof Popover> | null>(null)
const panelClass = computed(() => twMerge(
  'flex w-[calc(100vw-2rem)] max-w-lg max-h-[min(80dvh,48rem)] flex-col overflow-hidden',
  attrs.class as string | undefined, 'w-[calc(100vw-2rem)]',
))
const close = () => emit('update:visible', false)
function onPopoverHide() {
  if (props.visible) close()
  emit('hide')
}
let opening = 0
watch(() => props.visible, async (visible) => {
  const turn = ++opening
  if (!visible) { popover.value?.hide(); return }
  const anchor = findAnchor()
  anchored.value = !!anchor
  if (!anchor) return
  await nextTick()
  if (turn === opening && props.visible) popover.value?.showAt(anchor)
}, { immediate: true })
</script>

<template>
  <Popover v-if="anchored" ref="popover" :dismissable="dismissable" :panel-class="panelClass" content-class="flex min-h-0 flex-col p-0"
    :aria-labelledby="header || $slots.header ? titleId : undefined" @hide="onPopoverHide" @show="emit('show')">
    <div v-if="header || $slots.header" class="flex shrink-0 items-center justify-between gap-3 border-b border-border px-4 py-3">
      <div :id="titleId" class="min-w-0 text-base font-semibold text-text-primary"><slot name="header">{{ header }}</slot></div>
      <button v-if="closable" type="button" class="icon-action" :aria-label="t('common.close')" @click="close">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" class="h-4 w-4" aria-hidden="true"><path d="m6 6 12 12M6 18 18 6" /></svg>
      </button>
    </div>
    <div class="panel-form-body min-h-0 flex-1 overflow-y-auto px-4 py-3 text-text-primary"><slot /></div>
    <div v-if="$slots.footer" class="flex shrink-0 flex-wrap items-center justify-end gap-2 border-t border-border px-4 py-3"><slot name="footer" /></div>
  </Popover>
  <Dialog v-else v-bind="$attrs" :visible="visible" :header="header" :closable="closable"
    @update:visible="emit('update:visible', $event)" @hide="emit('hide')" @show="emit('show')">
    <template v-for="(_, slotName) in $slots" #[slotName]="slotProps">
      <div v-if="slotName === 'default'" class="panel-form-body"><slot v-bind="slotProps ?? {}" /></div>
      <slot v-else :name="slotName" v-bind="slotProps ?? {}" />
    </template>
  </Dialog>
</template>
