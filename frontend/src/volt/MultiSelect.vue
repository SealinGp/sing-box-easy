<template>
    <MultiSelect
        unstyled
        :pt="theme"
        :ptOptions="{
            mergeProps: ptViewMerge
        }"
    >
        <template v-for="(_, slotName) in $slots" #[slotName]="slotProps">
            <slot :name="slotName" v-bind="slotProps ?? {}" />
        </template>
    </MultiSelect>
</template>

<script setup lang="ts">
import MultiSelect, { type MultiSelectPassThroughOptions, type MultiSelectProps } from 'primevue/multiselect';
import { ptViewMerge } from './utils';

interface Props extends /* @vue-ignore */ MultiSelectProps {}
defineProps<Props>();

/*
 * Themed to match `volt/Select.vue`, which it deliberately mirrors — the two
 * appear side by side in the same forms, and until this rewrite they looked
 * like widgets from different applications.
 *
 * Three things were wrong and are worth not reintroducing:
 *
 *  1. DOUBLE-RINGED CHECKBOXES. Both `root` AND `box` carried `border-2` plus a
 *     background, so every option drew two concentric rings 2px apart. `root` is
 *     only a positioning wrapper for the visually-hidden input; the ring belongs
 *     to `box` alone.
 *  2. AN OPAQUE PANEL. `header`, `listContainer` and `emptyMessage` each set
 *     `bg-white dark:bg-gray-800`, painting over the frosted `.volt-multiselect-panel`
 *     surface that `primevue.css` provides. The panel rendered as a flat white
 *     box next to Select's glass one. Backgrounds are now left to that class.
 *  3. A SPLIT FIELD. `label` and `dropdown` each drew their own border and fill
 *     with `rounded-l-control` / `rounded-r-control`, leaving a visible seam down
 *     the middle. The field surface now comes from the shared `.volt-select` rule
 *     in `style/controls.css` — the same declaration that styles native inputs —
 *     and the two halves are transparent children of it.
 *
 * See src/volt/Select.vue for the same reasoning applied there first.
 */
const theme: MultiSelectPassThroughOptions = {
    root: ({ props }) => ({
        class: `volt-select relative inline-flex items-stretch cursor-pointer select-none text-sm
            transition-[border-color,box-shadow] duration-200
            ${props.disabled ? 'opacity-50 cursor-not-allowed' : ''}
            ${props.invalid ? 'border-red-400 dark:border-red-500' : ''}`
    }),
    labelContainer: `flex flex-auto overflow-hidden min-w-0`,
    /*
     * In `display="chip"` mode the chips live inside `label`. A single
     * nowrap/ellipsis line silently hides every selection past the first few —
     * for an outbound group with dozens of members that means the user can
     * neither see nor remove them. Chip mode therefore wraps, and caps the
     * height so a large selection scrolls instead of growing the dialog.
     *
     * Plain (comma-joined) mode keeps the single-line ellipsis, which is what
     * that display is for.
     */
    label: ({ props }) => ({
        class: `flex-auto cursor-pointer px-3 py-2 bg-transparent border-0
            text-gray-900 dark:text-gray-100
            ${
                props.display === 'chip'
                    ? 'flex flex-wrap items-center gap-1 max-h-32 overflow-y-auto'
                    : 'block overflow-hidden text-ellipsis whitespace-nowrap'
            }
            ${!props.modelValue || (Array.isArray(props.modelValue) && props.modelValue.length === 0) ? 'text-gray-400 dark:text-gray-500' : ''}`
    }),
    // Compact pills, matching volt/Chips.vue: a rule can hold a dozen rule sets,
    // and full-size chips turn the field into a wall of boxes.
    chipItem: `inline-flex items-center`,
    pcChip: {
        root: `inline-flex items-center gap-1 px-2 py-0.5 rounded-pill
            bg-primary-100 dark:bg-primary-900/40
            text-primary-700 dark:text-primary-200
            text-xs border-0`,
        label: `text-xs font-medium`,
        removeIcon: `w-3.5 h-3.5 shrink-0 cursor-pointer opacity-60 hover:opacity-100 hover:text-red-600 dark:hover:text-red-400 transition`
    },
    clearIcon: `absolute top-1/2 right-9 -translate-y-1/2 w-4 h-4 text-gray-400 hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-300 cursor-pointer transition-colors duration-200`,
    dropdown: `flex items-center justify-center shrink-0 pl-1 pr-3 bg-transparent border-0 text-gray-500 dark:text-gray-400`,
    dropdownIcon: `w-4 h-4`,
    loadingIcon: `w-4 h-4 animate-spin text-gray-500 dark:text-gray-400`,
    overlay: `volt-multiselect-panel absolute rounded-surface mt-1.5 max-h-80 overflow-hidden z-50`,
    /*
     * One row: the "select all" checkbox sits inline with the filter box. It
     * used to stack above it — an unlabelled circle floating in the corner of
     * the panel, which read as a rendering glitch rather than a control.
     */
    header: `flex items-center gap-2 p-2 border-b border-black/5 dark:border-white/10`,
    pcFilterContainer: {
        root: `relative flex-1 min-w-0`
    },
    pcFilter: {
        root: `w-full px-3 py-1.5 pr-9 text-sm
            bg-black/[0.03] dark:bg-white/5
            border border-transparent rounded-control
            text-gray-900 dark:text-gray-100
            placeholder:text-gray-400 dark:placeholder:text-gray-500
            focus:outline-none focus:border-[var(--color-primary)]
            transition-colors duration-200`
    },
    pcFilterIconContainer: {
        root: `absolute top-1/2 right-3 -translate-y-1/2`
    },
    filterIcon: `w-4 h-4 text-gray-400 dark:text-gray-500`,
    listContainer: `overflow-auto`,
    list: `p-1.5 list-none m-0`,
    optionGroup: `px-2 pt-2 pb-1 text-xs font-semibold uppercase tracking-wide text-gray-400 dark:text-gray-500`,
    option: ({ context }) => ({
        class: `relative flex items-center gap-2 px-2.5 py-1.5 rounded-control cursor-pointer text-sm
            text-gray-700 dark:text-gray-200
            transition-colors duration-150
            ${context.focused ? 'volt-select-option-focus' : ''}
            ${context.selected ? 'volt-select-option-selected font-semibold' : ''}
            ${context.disabled ? 'opacity-50 cursor-not-allowed' : ''}`
    }),
    optionLabel: `flex-auto truncate`,
    /*
     * Checkbox geometry lives in ONE place — `box`. `root` positions the
     * visually-hidden input and nothing else; giving it a border too is what
     * produced the double ring. `rounded-pill` on a 1rem box is a true circle.
     */
    pcHeaderCheckbox: {
        root: `relative inline-flex items-center justify-center w-4 h-4 shrink-0`,
        box: `flex items-center justify-center w-4 h-4 rounded-pill
            border border-gray-300 dark:border-white/25
            bg-white/70 dark:bg-white/10
            transition-colors duration-150
            peer-hover:border-primary-400
            peer-checked:bg-primary peer-checked:border-primary
            peer-focus-visible:ring-2 peer-focus-visible:ring-primary-500/40`,
        input: `absolute inset-0 m-0 appearance-none w-4 h-4 peer cursor-pointer opacity-0`,
        icon: `w-2.5 h-2.5 text-white`
    },
    pcOptionCheckbox: {
        root: `relative inline-flex items-center justify-center w-4 h-4 shrink-0`,
        box: `flex items-center justify-center w-4 h-4 rounded-pill
            border border-gray-300 dark:border-white/25
            bg-white/70 dark:bg-white/10
            transition-colors duration-150
            peer-checked:bg-primary peer-checked:border-primary`,
        input: `absolute inset-0 m-0 appearance-none w-4 h-4 peer cursor-pointer opacity-0`,
        icon: `w-2.5 h-2.5 text-white`
    },
    emptyMessage: `px-3 py-6 text-center text-sm text-gray-500 dark:text-gray-400`,
    /*
     * No open/close animation.
     *
     * The panel used to scale from 95% over 100ms in and 75ms out. Below roughly
     * 150ms motion stops reading as motion and just registers as a flicker, and
     * a scale transform on a list of text makes the labels visibly reflow. The
     * panel now simply appears. (volt/Select.vue keeps a transition because its
     * 220ms travel is long enough to read as a deliberate drop-down; if this one
     * ever gets motion back, it needs the same duration, not the old 100ms.)
     */
    transition: {
        enterFromClass: '',
        enterActiveClass: '',
        enterToClass: '',
        leaveFromClass: '',
        leaveActiveClass: '',
        leaveToClass: ''
    }
};
</script>
