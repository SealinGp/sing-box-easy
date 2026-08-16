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
import { computed, ref, type VNode } from 'vue'
import { useFillHeight } from '../composables/useFillHeight'

export interface TableColumn {
  /** Stable identity for the `v-for` key. */
  key: string
  /** Header text. Omit for columns that need no label (checkbox, index). */
  label?: string
  /** `right` adds `col-actions`, which right-aligns the header AND its cells. */
  align?: 'left' | 'right'
  /** Extra classes for the `<th>` — an explicit `w-8`, most often. */
  class?: string
}

interface Props {
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
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
  empty: false,
  scroll: true,
})

defineSlots<{
  /** Body rows — `<tr>` elements. */
  default: () => VNode[]
  /** Replaces the generated `<th>`s. Provide `<th>` elements, not a `<tr>`. */
  head?: () => VNode[]
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
</script>

<template>
  <div v-if="props.loading" class="flex items-center justify-center py-8">
    <slot name="loading">
      <div class="animate-spin rounded-pill h-6 w-6 border-b-2 border-primary-600"></div>
    </slot>
  </div>

  <div v-else-if="props.empty" class="py-8 text-center text-gray-500 dark:text-gray-400">
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
    :style="props.maxHeight ? { '--scroll-max-h': props.maxHeight } : undefined"
  >
    <table class="data-table">
      <thead v-if="$slots.head || props.columns?.length">
        <tr>
          <slot name="head">
            <th
              v-for="col in props.columns"
              :key="col.key"
              :class="[col.align === 'right' ? 'col-actions' : undefined, col.class]"
            >
              {{ col.label }}
            </th>
          </slot>
        </tr>
      </thead>
      <tbody>
        <slot />
      </tbody>
    </table>
  </div>
</template>
