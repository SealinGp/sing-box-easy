<script setup lang="ts">
/**
 * The one table.
 *
 * Nine tables in this app each hand-rolled the same three things: a scroll
 * wrapper, a `<thead>` of identically-classed `<th>`s, and the
 * loading / empty / data triad around it. That is how `<td class="px-6 py-4">`
 * ended up in six files with two other row heights nobody shared.
 *
 * The visual layer still lives in `style/density.css` (`.scroll-region`,
 * `.data-table`) — this component owns the STRUCTURE, so a call site writes
 * only what is actually specific to it: its columns and its rows.
 *
 * Body rows stay hand-authored via the default slot rather than being driven
 * off a `rows` prop. The nine tables render checkboxes, badges, index numbers,
 * copy-to-clipboard buttons and PopConfirms in their cells; a generic cell
 * renderer would need an escape hatch per column and end up longer than the
 * markup it replaced.
 *
 *     <Table :columns="cols" :loading="loading" :empty="!rows.length">
 *       <template #empty>Nothing here yet</template>
 *       <tr v-for="r in rows" :key="r.tag">
 *         <td>{{ r.tag }}</td>
 *         <td class="col-actions">…</td>
 *       </tr>
 *     </Table>
 *
 * ⚠️ `columns` describes the HEADER only. Body cells are positional, so a
 * column added here needs a matching `<td>` in the row markup — there is no
 * runtime check for that. Use the `#head` slot instead when a header needs
 * markup of its own (a select-all checkbox, say).
 */
import { computed, ref, watch, type VNode } from 'vue'
import { useFillHeight } from '../composables/useFillHeight'
import { useInterfaceMotion } from '../composables/useInterfaceMotion'
import Skeleton from './Skeleton.vue'
import TableRecords from './TableRecords'
import '../style/collections.css'
import { useI18n } from 'vue-i18n'

const interfaceMotion = useInterfaceMotion()
const { t } = useI18n()

export interface TableColumn {
  /** Stable identity for the `v-for` key. */
  key: string
  minWidth?: number
  /** Header text. Omit for columns that need no label (checkbox, index). */
  label?: string
  /** `right` adds `col-actions`, which right-aligns the header AND its cells. */
  align?: 'left' | 'right'
  /** Extra classes for the `<th>` — an explicit `w-8`, most often. */
  class?: string
}

interface Props {
  /** Table layout utilities; the scroll wrapper stays independent. */
  tableClass?: string
  presentation?: boolean
  selectable?: boolean
  resizable?: boolean
  columnStorageKey?: string
  /** Header definition. Omit and use the `#head` slot for bespoke headers. */
  columns?: TableColumn[]
  /** Shows the spinner instead of the table. */
  loading?: boolean
  /** Shows the `#empty` slot instead of the table. Ignored while `loading`. */
  empty?: boolean
  /**
   * Overrides `--scroll-max-h` for this table only — e.g. `'20rem'`. The token
   * default fills the window, which is right for a table that IS the page.
   */
  maxHeight?: string
  /**
   * Set `false` when an ancestor already scrolls. Config.vue's version list
   * sits in a modal body that carries its own `overflow-y-auto`, and two
   * nested scrollbars are worse than one in the wrong place.
   */
  scroll?: boolean
  /**
   * Animates rows sliding to new positions (Vue's FLIP `<TransitionGroup>`,
   * rendered as the `<tbody>`).
   *
   * Opt-in, and it only does anything if the caller keys rows by a STABLE
   * identity rather than by array index: with index keys Vue patches cell text
   * in place and no `<tr>` ever moves, so there is nothing to animate.
   */
  transition?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
  empty: false,
  scroll: true,
  transition: false,
  presentation: true,
  selectable: true,
})

defineSlots<{
  /** Body rows — `<tr>` elements. */
  default: () => VNode[]
  /** Replaces the generated `<th>`s. Provide `<th>` elements, not a `<tr>`. */
  head?: () => VNode[]
  cards?: (state: { selected: Set<string>; toggle: (key: string) => void }) => VNode[]
  /** Shown when `empty` and not `loading`. */
  empty?: () => VNode[]
  /** Replaces the default spinner. */
  loading?: () => VNode[]
}>()

/*
 * The cap is measured, not computed from a constant — `useFillHeight` explains
 * why at length (a wrapping top bar alone swings the available height by
 * 136px). An explicit `maxHeight` opts out, as does `scroll: false`.
 */
const region = ref<HTMLElement | null>(null)
const noFill = computed(() => !props.scroll || !!props.maxHeight)
useFillHeight(region, noFill)

const loadingHeight = ref(0)
const loadingWidths = ref<number[]>([])
const loadingRows = ref([36, 36, 36, 36, 36])
watch(() => props.loading, (loading) => {
  if (!interfaceMotion || !loading || !region.value) return
  loadingHeight.value = region.value.getBoundingClientRect().height
  loadingWidths.value = [...region.value.querySelectorAll('thead th')].map(cell => cell.getBoundingClientRect().width)
  const heights = [...region.value.querySelectorAll('tbody tr')].slice(0, 12).map(row => row.getBoundingClientRect().height)
  if (heights.length) loadingRows.value = heights
}, { flush: 'pre' })
// On an initial load there was no region to measure in the pre-update watcher.
// Read the rendered custom header as soon as its ref exists, before first paint,
// so a six-column table does not start with the four-column fallback skeleton.
watch(region, (element) => {
  if (!interfaceMotion || !element || !props.loading) return
  loadingWidths.value = [...element.querySelectorAll('thead th')].map(cell => cell.getBoundingClientRect().width)
}, { flush: 'post' })
const skeletonColumns = computed(() => loadingWidths.value.length || props.columns?.length || 4)
</script>

<template>
  <div v-if="props.loading && !interfaceMotion" class="flex items-center justify-center py-8">
    <slot name="loading">
      <div class="animate-spin rounded-pill h-6 w-6 border-b-2 border-primary-600"></div>
    </slot>
  </div>

  <div v-else-if="props.empty && !props.loading" class="py-8 text-center text-gray-500 dark:text-gray-400">
    <slot name="empty" />
  </div>

  <!--
    `--scroll-max-h` is set inline rather than through a class so a caller can
    pass any length without minting a utility for it. The wrapper is kept even
    when `scroll` is false so the DOM shape does not change between modes.
  -->
  <div
    v-else
    ref="region"
    :class="props.scroll ? 'scroll-region' : undefined"
    :aria-busy="props.loading || undefined"
    :aria-label="props.loading ? t('common.loading') : undefined"
    :style="{
      ...(props.maxHeight ? { '--scroll-max-h': props.maxHeight } : {}),
      ...(props.loading && interfaceMotion ? { minHeight: `${loadingHeight}px` } : {}),
    }"
  >
    <span v-if="props.loading && interfaceMotion" class="sr-only" role="status">{{ t('common.loading') }}</span>
    <TableRecords :columns="props.columns" :table-class="props.tableClass"
      :transition="props.transition" :presentation="props.presentation" :selectable="props.selectable"
      :storage-key="props.columnStorageKey" :loading="props.loading">
      <template v-if="$slots.head" #head><slot name="head" /></template>
      <template v-if="$slots.cards" #cards="state"><slot name="cards" v-bind="state" /></template>
      <template #default>
        <tr v-if="props.loading" v-for="(height, row) in loadingRows" :key="`skeleton-${row}`" :style="{ height: `${height}px` }">
          <td v-for="column in skeletonColumns" :key="column"><Skeleton class="h-3 w-3/4" /></td>
        </tr>
        <slot v-else />
      </template>
    </TableRecords>
  </div>
</template>

<style>
.table-column-resizer { position: absolute; inset: 0 -5px 0 auto; width: 10px; cursor: col-resize; touch-action: none; z-index: 2; }
.table-column-resizer::after { content: ''; position: absolute; top: 25%; bottom: 25%; left: 4px; width: 2px; background: var(--color-border); }
.table-column-resizer:hover::after, .table-column-resizer:focus-visible::after { background: var(--color-primary-500); }
.table-column-resizer:focus-visible { outline: 2px solid var(--color-primary-500); outline-offset: -2px; }
</style>
