<template>
    <Select
        unstyled
        :pt="theme"
        :ptOptions="{
            mergeProps: ptViewMerge
        }"
    >
        <template v-for="(_, slotName) in $slots" #[slotName]="slotProps">
            <slot :name="slotName" v-bind="slotProps ?? {}" />
        </template>
    </Select>
</template>

<script setup lang="ts">
import Select, { type SelectPassThroughOptions, type SelectProps } from 'primevue/select';
import { ptViewMerge } from './utils';

interface Props extends /* @vue-ignore */ SelectProps {}
defineProps<Props>();

const theme: SelectPassThroughOptions = {
    /*
     * The field is one surface on the root; `label` and `dropdown` are
     * transparent children.
     *
     * Previously each carried its own border and background, which produced a
     * visible seam AND a real bug: `dropdown` is a <div> with `bg-white`, so it
     * matched the global `.liquid-app div.bg-white` card rule in style.css and
     * inherited a full panel shadow (0 10px 24px) on a 32px icon box — it read
     * as a button floating outside the field. `label` is a <span>, so it never
     * matched, which is why only one half looked detached.
     *
     * So no bare `bg-white` token appears below. The field's surface now comes
     * from the shared `.volt-select` rule in style.css, which sits in the same
     * declaration as the native `.select` so both stay pixel-identical.
     */
    root: ({ props }) => ({
        class: `volt-select relative inline-flex items-stretch cursor-pointer select-none
            text-sm
            transition-[border-color,box-shadow] duration-200
            ${props.disabled ? 'opacity-50 cursor-not-allowed' : ''}
            ${props.invalid ? 'border-red-400 dark:border-red-500' : ''}`
    }),
    label: ({ props }) => ({
        class: `flex-auto block overflow-hidden text-ellipsis whitespace-nowrap cursor-pointer
            px-3 py-2 bg-transparent border-0
            text-gray-900 dark:text-gray-100
            ${!props.modelValue || (Array.isArray(props.modelValue) && props.modelValue.length === 0) ? 'text-gray-400 dark:text-gray-500' : ''}`
    }),
    dropdown: () => ({
        class: `flex items-center justify-center shrink-0
            pl-1 pr-3 bg-transparent border-0
            text-gray-500 dark:text-gray-400`
    }),
    dropdownIcon: `w-4 h-4`,
    overlay: `volt-select-panel absolute rounded-surface mt-1.5 max-h-80 overflow-hidden z-50`,
    header: `p-2 border-b border-black/5 dark:border-white/10`,
    pcFilterContainer: {
        root: `relative`
    },
    pcFilter: {
        root: `w-full px-3 py-1.5 pr-9 text-sm
            bg-black/[0.03] dark:bg-white/5
            border border-transparent
            rounded-control
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
    optionGroup: `px-2 pt-2 pb-1`,
    optionGroupLabel: `text-xs font-semibold uppercase tracking-wide text-gray-400 dark:text-gray-500`,
    option: ({ context }) => ({
        class: `relative flex items-center gap-2 px-3 py-2 rounded-control cursor-pointer text-sm
            text-gray-700 dark:text-gray-200
            transition-colors duration-150
            ${context.focused ? 'volt-select-option-focus' : ''}
            ${context.selected ? 'volt-select-option-selected font-semibold' : ''}
            ${context.disabled ? 'opacity-50 cursor-not-allowed' : ''}`
    }),
    optionLabel: `flex-auto truncate`,
    optionCheckIcon: `w-4 h-4`,
    optionBlankIcon: `w-4 h-4`,
    emptyMessage: `px-3 py-8 text-center text-sm text-gray-500 dark:text-gray-400`,
    clearIcon: `absolute top-1/2 right-12 -translate-y-1/2 w-4 h-4 text-gray-400 hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-300 cursor-pointer transition-colors duration-200`,
    loadingIcon: `w-4 h-4 animate-spin text-gray-500 dark:text-gray-400`,
    /*
     * Overlay open/close.
     *
     * Was 100ms in / 75ms out, which is below the ~150ms threshold where motion
     * stops reading as motion and just looks like a flicker. Now 220ms in on the
     * same easing the sidebar uses, with a short travel so the panel appears to
     * drop out of the field rather than pop into existence. `origin-top` anchors
     * the scale to the field instead of the panel's centre.
     *
     * Only opacity and transform are transitioned (both compositor-driven);
     * `transition-all` would also animate the panel's size and colours.
     */
    transition: {
        enterFromClass: 'opacity-0 scale-[0.97] -translate-y-1',
        enterActiveClass:
            'origin-top transition-[opacity,transform] duration-220 ease-[cubic-bezier(0.32,0.72,0,1)]',
        enterToClass: 'opacity-100 scale-100 translate-y-0',
        leaveFromClass: 'opacity-100 scale-100 translate-y-0',
        leaveActiveClass: 'origin-top transition-[opacity,transform] duration-150 ease-in',
        leaveToClass: 'opacity-0 scale-[0.97] -translate-y-1'
    }
};
</script>
