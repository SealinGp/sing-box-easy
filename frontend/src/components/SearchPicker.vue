<script setup lang="ts">
/**
 * A search box that drops a filtered list under itself and emits one pick.
 *
 * Extracted from NodeRules.vue, which had four copies of it — include-node,
 * exclude-node, country-code and group-tag — each with its own `query` ref,
 * `open` ref, `filtered` computed and ~25 lines of markup. The four differed
 * only in their list, their placeholder and one accent colour, so a fix to one
 * (the `mousedown.prevent` that makes a click beat the input's blur, say) had
 * to be made four times or not at all.
 *
 * WHY NOT volt/Select
 * ───────────────────
 * PrimeVue's Select with `filter` is a *value picker*: it holds a selection and
 * shows it in the field. This is an *action* — pick a node and it becomes a
 * matcher, pick another and that becomes a second matcher — so there is no
 * selected value to display, and the box must clear itself after every pick to
 * be ready for the next one. Bending Select into that shape means fighting its
 * model; 60 lines of input + popover does not.
 *
 * The behaviour that is easy to get wrong and is therefore fixed here:
 *   - options close on blur, so they are chosen with `mousedown.prevent` —
 *     a plain `click` fires after blur has already unmounted the list;
 *   - the query resets on every pick, so the next one starts from the full
 *     list rather than the leftovers of the last search;
 *   - the list is capped (`limit`), because a large subscription has hundreds
 *     of nodes and rendering all of them on focus is felt.
 */
import { computed, ref } from 'vue'

export interface PickerOption {
  /** What `select` emits, and the default label/tooltip. */
  value: string
  label?: string
  /** `title` attribute — the country picker uses it to list synonyms. */
  title?: string
  /** Extra text the query matches against (a country's code and synonyms). */
  terms?: string[]
  /** Right-aligned pill, e.g. "DIRECT" for an opt-in outbound. */
  badge?: string
}

const props = withDefaults(
  defineProps<{
    options: PickerOption[]
    placeholder?: string
    emptyMessage?: string
    /** `danger` tints focus and hover red — used by the exclude (deny-list) picker. */
    tone?: 'primary' | 'danger'
    /** Max options rendered. 0 disables the cap. */
    limit?: number
    /** Popover width; it is positioned against the input and may be wider. */
    panelClass?: string
    inputClass?: string
  }>(),
  { tone: 'primary', limit: 50, panelClass: 'w-full', inputClass: 'w-full' },
)

const emit = defineEmits<{ (e: 'select', value: string): void }>()

const query = ref('')
const open = ref(false)

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  const matches = q
    ? props.options.filter((option) =>
        [option.value, option.label, ...(option.terms ?? [])]
          .filter(Boolean)
          .some((text) => String(text).toLowerCase().includes(q)),
      )
    : props.options
  return props.limit > 0 ? matches.slice(0, props.limit) : matches
})

function pick(option: PickerOption) {
  emit('select', option.value)
  // Reset so the list is whole again, ready for the next pick.
  query.value = ''
}
</script>

<template>
  <div class="relative">
    <input
      v-model="query"
      :class="[
        'bg-gray-50 dark:bg-slate-800 border border-gray-200 dark:border-gray-700 rounded-control px-3 py-1.5 text-xs text-gray-900 dark:text-white focus:outline-none',
        tone === 'danger' ? 'focus:border-red-500' : 'focus:border-primary-500',
        inputClass,
      ]"
      :placeholder="placeholder"
      @focus="open = true"
      @blur="open = false"
    />

    <div
      v-if="open"
      :class="[
        'search-picker-popover absolute z-20 mt-1 max-h-48 overflow-y-auto rounded-surface border border-gray-200 dark:border-gray-800 bg-white dark:bg-slate-900 text-xs',
        panelClass,
      ]"
    >
      <!--
        `mousedown.prevent`, not `click`: the input's blur closes this list, and
        blur wins the race against click. Preventing default on mousedown also
        keeps focus in the input, so the box stays open for a second pick.
      -->
      <button
        v-for="option in filtered"
        :key="option.value"
        type="button"
        :class="[
          'flex w-full items-center gap-1.5 cursor-pointer px-3 py-2 text-left text-gray-700 dark:text-gray-300',
          tone === 'danger'
            ? 'hover:bg-red-50 dark:hover:bg-red-950/30 hover:text-red-600 dark:hover:text-red-400'
            : 'hover:bg-primary-50 dark:hover:bg-primary-950/30 hover:text-primary-600 dark:hover:text-primary-400',
        ]"
        :title="option.title ?? option.label ?? option.value"
        @mousedown.prevent="pick(option)"
      >
        <span class="truncate">{{ option.label ?? option.value }}</span>
        <span
          v-if="option.badge"
          class="ml-auto shrink-0 px-1.5 py-0.5 rounded-pill text-[9px] font-bold uppercase bg-emerald-100 dark:bg-emerald-950/40 text-emerald-700 dark:text-emerald-400"
        >
          {{ option.badge }}
        </span>
      </button>

      <div v-if="!filtered.length" class="px-3 py-2 text-gray-400 text-center">
        {{ emptyMessage }}
      </div>
    </div>
  </div>
</template>

<style scoped>
/* Moved with the markup from NodeRules.vue's `.node-rule-popover`. */
.search-picker-popover {
  box-shadow:
    0 14px 32px rgba(15, 23, 42, 0.12),
    inset 0 1px 0 rgba(255, 255, 255, 0.45);
}
</style>
