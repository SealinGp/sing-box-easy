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

const theme: MultiSelectPassThroughOptions = {
    root: ({ props }) => ({
        class: `relative inline-flex cursor-pointer select-none
            ${props.disabled ? 'opacity-50 cursor-not-allowed' : ''}
            ${props.invalid ? 'border-red-500' : ''}`
    }),
    labelContainer: `flex flex-auto overflow-hidden`,
    label: ({ props, state }) => ({
        class: `flex-auto block overflow-hidden text-ellipsis whitespace-nowrap cursor-pointer
            px-3 py-2 rounded-l-control
            bg-white dark:bg-gray-800
            border border-r-0 border-gray-300 dark:border-gray-600
            text-gray-900 dark:text-gray-100
            transition-all duration-200
            ${state.focused ? 'border-primary-500 dark:border-primary-400 ring-1 ring-primary-500 dark:ring-primary-400' : ''}
            ${!props.modelValue || (Array.isArray(props.modelValue) && props.modelValue.length === 0) ? 'text-gray-400 dark:text-gray-500' : ''}`
    }),
    chipItem: `inline-flex items-center py-0.5 px-2 mr-1 mb-0.5 rounded-control
        bg-primary-100 dark:bg-primary-900/30
        text-primary-700 dark:text-primary-300
        text-sm`,
    pcChip: {
        root: `inline-flex items-center bg-transparent border-0 p-0`,
        label: `text-sm`,
        removeIcon: `ml-1.5 w-3.5 h-3.5 cursor-pointer hover:text-primary-900 dark:hover:text-primary-100 transition-colors`
    },
    clearIcon: `absolute top-1/2 right-12 -translate-y-1/2 w-4 h-4 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 cursor-pointer`,
    dropdown: ({ state, props }) => ({
        class: `flex items-center justify-center shrink-0
            px-3 py-2 rounded-r-control
            bg-white dark:bg-gray-800
            border border-gray-300 dark:border-gray-600
            text-gray-500 dark:text-gray-400
            transition-all duration-200
            ${state.focused ? 'border-primary-500 dark:border-primary-400 ring-1 ring-primary-500 dark:ring-primary-400' : ''}
            ${!props.disabled ? 'hover:bg-gray-50 dark:hover:bg-gray-700' : ''}`
    }),
    dropdownIcon: `w-4 h-4 text-gray-600 dark:text-gray-400`,
    loadingIcon: `w-4 h-4 animate-spin text-gray-500 dark:text-gray-400`,
    overlay: `volt-multiselect-panel absolute rounded-surface mt-1 max-h-80 overflow-hidden z-50`,
    header: `px-3 py-2 border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800`,
    pcHeaderCheckbox: {
        root: `relative inline-flex items-center justify-center w-5 h-5 mr-2
            border-2 border-gray-300 dark:border-gray-600
            rounded-control
            bg-white dark:bg-gray-700
            transition-all duration-200
            hover:border-primary-500
            data-[p-checked=true]:bg-primary-600 data-[p-checked=true]:border-primary-600`,
        box: `flex items-center justify-center w-5 h-5
            border-2 border-gray-300 dark:border-gray-600
            rounded-control
            bg-white dark:bg-gray-700
            transition-all duration-200
            hover:border-primary-500
            peer-checked:bg-primary-600 peer-checked:border-primary-600
            peer-focus:ring-2 peer-focus:ring-primary-500 peer-focus:ring-offset-2`,
        input: `absolute appearance-none w-5 h-5 peer cursor-pointer opacity-0`,
        icon: `w-3.5 h-3.5 text-white`
    },
    pcFilterContainer: {
        root: `relative mb-2`
    },
    pcFilter: {
        root: `w-full px-3 py-2 pr-10
            bg-white dark:bg-gray-800
            border border-gray-300 dark:border-gray-600
            rounded-control
            text-gray-900 dark:text-gray-100
            placeholder:text-gray-400 dark:placeholder:text-gray-500
            focus:outline-none focus:ring-2 focus:ring-primary-500 dark:focus:ring-primary-400 focus:border-primary-500 dark:focus:border-primary-400
            transition-all duration-200`
    },
    pcFilterIconContainer: {
        root: `absolute top-1/2 right-3 -translate-y-1/2`
    },
    filterIcon: `w-4 h-4 text-gray-400 dark:text-gray-500`,
    listContainer: `overflow-auto bg-white dark:bg-gray-800`,
    list: `py-1 list-none m-0 p-0`,
    optionGroup: `px-3 py-2 bg-gray-50 dark:bg-gray-700/50 border-b border-gray-100 dark:border-gray-700 font-semibold text-sm text-gray-700 dark:text-gray-300`,
    option: ({ context }) => ({
        class: `relative flex items-center gap-2 px-3 py-2 m-1 mx-2 rounded-control cursor-pointer
            text-gray-900 dark:text-gray-100
            transition-all duration-200
            ${context.focused ? 'bg-primary-50 dark:bg-primary-900/30 text-primary-700 dark:text-primary-300' : ''}
            ${context.selected ? 'bg-primary-100 dark:bg-primary-900/40 text-primary-800 dark:text-primary-200 font-semibold' : ''}
            ${!context.disabled ? 'hover:bg-gray-100 dark:hover:bg-gray-700' : 'opacity-50 cursor-not-allowed'}
            ${context.focused && context.selected ? 'bg-primary-200 dark:bg-primary-800/50' : ''}`
    }),
    optionLabel: `flex-auto`,
    pcOptionCheckbox: {
        root: `relative inline-flex items-center justify-center w-5 h-5 mr-2
            border-2 border-gray-300 dark:border-gray-600
            rounded-control
            bg-white dark:bg-gray-700
            transition-all duration-200
            data-[p-checked=true]:bg-primary-600 data-[p-checked=true]:border-primary-600`,
        box: `flex items-center justify-center w-5 h-5
            border-2 border-gray-300 dark:border-gray-600
            rounded-control
            bg-white dark:bg-gray-700
            transition-all duration-200
            peer-checked:bg-primary-600 peer-checked:border-primary-600`,
        input: `absolute appearance-none w-5 h-5 peer cursor-pointer opacity-0`,
        icon: `w-3.5 h-3.5 text-white`
    },
    emptyMessage: `px-3 py-8 text-center text-gray-500 dark:text-gray-400 bg-white dark:bg-gray-800`,
    transition: {
        enterFromClass: 'opacity-0 scale-95',
        enterActiveClass: 'transition-all duration-100 ease-out',
        enterToClass: 'opacity-100 scale-100',
        leaveFromClass: 'opacity-100 scale-100',
        leaveActiveClass: 'transition-all duration-75 ease-in',
        leaveToClass: 'opacity-0 scale-95'
    }
};
</script>
