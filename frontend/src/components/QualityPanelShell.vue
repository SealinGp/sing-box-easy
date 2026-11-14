<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import ContextPanel from '../volt/ContextPanel.vue'
import { useOverlayAnchor } from '../composables/useOverlayAnchor'

const props = defineProps<{ visible: boolean; header: string }>()
const emit = defineEmits<{ 'update:visible': [boolean] }>()
const { t } = useI18n()
const findAnchor = useOverlayAnchor()
const media = window.matchMedia('(max-width: 767px)')
const compact = ref(media.matches)
const host = ref<HTMLElement | null>(null)
const content = ref<HTMLElement>()
let row: HTMLElement | null = null
let anchor: HTMLElement | null = null
function cleanup() { row?.remove(); row = null; host.value = null }
function close() { emit('update:visible', false); anchor?.focus({ preventScroll: true }) }
function resize() { compact.value = media.matches }
media.addEventListener('change', resize)
watch(() => [props.visible, compact.value], async () => {
  cleanup()
  if (!props.visible) return
  anchor = findAnchor() ?? anchor
  if (!compact.value) return
  const sourceRow = anchor?.closest('tr')
  if (sourceRow) {
    const detailRow = document.createElement('tr')
    const cell = document.createElement('td')
    cell.colSpan = [...sourceRow.cells].reduce((total, item) => total + item.colSpan, 0)
    detailRow.append(cell)
    sourceRow.after(detailRow)
    row = detailRow
    host.value = cell
  }
  await nextTick()
  content.value?.focus({ preventScroll: true })
}, { immediate: true, flush: 'post' })
onBeforeUnmount(() => { cleanup(); media.removeEventListener('change', resize) })
</script>

<template>
  <template v-if="compact">
    <Teleport v-if="visible" :to="host ?? 'body'" :disabled="!host">
      <section ref="content" tabindex="-1" class="my-3 w-[calc(100vw-3rem)] max-w-xl rounded-surface border border-border bg-bg-secondary p-4" :aria-label="header" @keydown.esc.stop="close">
        <div class="mb-4 flex items-center justify-between gap-3">
          <h3 class="text-sm font-semibold">{{ header }}</h3>
          <button type="button" class="icon-action" :aria-label="t('common.close')" @click="close">×</button>
        </div>
        <slot />
      </section>
    </Teleport>
  </template>
  <ContextPanel v-else :visible="visible" :header="header" class="max-w-xl" @update:visible="emit('update:visible', $event)">
    <slot />
  </ContextPanel>
</template>
