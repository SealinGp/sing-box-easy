<script setup lang="ts">
/**
 * One `label: value` pair inside a `<ListRow>`.
 *
 * This markup was repeated 18 times across `RuleSets.vue` and
 * `RoutingRuleItem.vue`, each copy spelling its own
 * `font-medium text-gray-700 dark:text-gray-300` / `ml-2 text-gray-900
 * dark:text-gray-100` — which is 36 raw `text-gray-*` usages depending on the
 * `legacy.css` dark-mode shim. The colours now come from tokens.
 *
 * **A field with no value renders nothing.** sing-box rules are sparse — a
 * routing rule populates two or three of thirteen possible matchers — so every
 * one of those call sites carried its own `v-if`. Folding the guard in here
 * removes them, and "hide a field that has nothing in it" is the only sensible
 * behaviour anyway.
 *
 * Arrays are joined, because sing-box accepts scalar-or-array on the wire for
 * every list-like matcher and callers should not each re-implement that.
 */
import { computed } from 'vue'

interface Props {
  label: string
  /** Scalar or array. Nullish, `''` and `[]` all render nothing. */
  value?: string | number | readonly (string | number)[] | null
}

const props = defineProps<Props>()

const text = computed(() => {
  const v = props.value
  if (v === null || v === undefined) return ''
  if (Array.isArray(v)) return v.join(', ')
  return String(v)
})

// A caller supplying slot content owns the decision to render, not this guard.
const slots = defineSlots<{ default?: () => unknown }>()
const shown = computed(() => text.value !== '' || !!slots.default)
</script>

<template>
  <div v-if="shown" class="list-field">
    <span class="list-field-label">{{ label }}</span>
    <span class="list-field-value"><slot>{{ text }}</slot></span>
  </div>
</template>
