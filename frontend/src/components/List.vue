<script setup lang="ts">
/**
 * The one list — `<Table>`'s sibling for record-per-row panels.
 *
 * `RuleSets.vue` and `RoutingRuleItem.vue` rendered the same shape from two
 * separate copies of the same utilities, and neither had a height cap: a
 * config with 40 routing rules pushed the page furniture off-screen exactly
 * the way the tables used to.
 *
 * This owns the STRUCTURE — the capped scroll region, the row spacing, and the
 * loading / empty / data triad. Rows come from `<ListRow>`, fields from
 * `<ListField>`. The visual layer lives in `style/density.css`.
 *
 *     <List :loading="loading" :empty="!rows.length">
 *       <template #empty>Nothing here yet</template>
 *       <ListRow v-for="r in rows" :key="r.tag">
 *         <ListField :label="$t('…tag')" :value="r.tag" />
 *         <template #actions>…</template>
 *       </ListRow>
 *     </List>
 *
 * Props mirror `<Table>` deliberately: the two are the same component with a
 * different row primitive, and an operator switching between the Route and
 * Outbounds pages should not meet two different scroll behaviours.
 */
import { computed, ref, type VNode } from 'vue'
import { useFillHeight } from '../composables/useFillHeight'

interface Props {
  /** Shows the spinner instead of the rows. */
  loading?: boolean
  /** Shows the `#empty` slot instead of the rows. Ignored while `loading`. */
  empty?: boolean
  /**
   * Overrides `--scroll-max-h` for this list only — e.g. `'20rem'`. The token
   * default fills the window, which is right for a list that IS the page.
   */
  maxHeight?: string
  /**
   * Set `false` when an ancestor already scrolls, so the page does not end up
   * with two nested scrollbars.
   */
  scroll?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
  empty: false,
  scroll: true,
})

defineSlots<{
  /** The rows — `<ListRow>` elements. */
  default: () => VNode[]
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
  <div v-if="props.loading" class="flex items-center justify-center py-6">
    <slot name="loading">
      <div class="animate-spin rounded-pill h-6 w-6 border-b-2 border-primary-600"></div>
    </slot>
  </div>

  <div v-else-if="props.empty" class="py-6 text-center text-gray-500 dark:text-gray-400">
    <slot name="empty" />
  </div>

  <!--
    The scroll region and the row stack are separate elements on purpose. Row
    gap belongs to the inner stack; putting it on the scroller would let the
    last row's gap collapse against the scroll edge and make the list look like
    it ends one row early.
  -->
  <div
    v-else
    ref="region"
    :class="props.scroll ? 'scroll-region' : undefined"
    :style="props.maxHeight ? { '--scroll-max-h': props.maxHeight } : undefined"
  >
    <div class="list-rows">
      <slot />
    </div>
  </div>
</template>
