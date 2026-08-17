<script setup lang="ts">
/**
 * One row inside a `<List>`: a two-column grid of `<ListField>`s, with an
 * optional action cluster pinned to the right.
 *
 * `min-w-0` on the field column is load-bearing — a grid/flex child defaults to
 * `min-width: auto`, so one long rule-set URL would otherwise push the actions
 * off the edge instead of truncating.
 */
import type { VNode } from 'vue'

defineSlots<{
  /** The `<ListField>`s. */
  default: () => VNode[]
  /** Edit / delete affordances. Use `class="list-action-btn"` on each. */
  actions?: () => VNode[]
  /**
   * Optional gutter pinned to the row's left edge — a drag handle, a checkbox.
   * Sits OUTSIDE the `min-w-0` field column so it never shrinks when a long
   * value pushes on it.
   */
  leading?: () => VNode[]
}>()
</script>

<template>
  <div class="list-row">
    <div class="flex items-start justify-between">
      <div v-if="$slots.leading" class="list-row-leading">
        <slot name="leading" />
      </div>
      <div class="min-w-0 flex-1">
        <div class="list-row-grid">
          <slot />
        </div>
      </div>
      <div v-if="$slots.actions" class="list-row-actions">
        <slot name="actions" />
      </div>
    </div>
  </div>
</template>
