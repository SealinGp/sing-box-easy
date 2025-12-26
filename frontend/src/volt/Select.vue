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
import { ref } from 'vue';
import { ptViewMerge } from './utils';

interface Props extends /* @vue-ignore */ SelectProps {}
defineProps<Props>();

const theme = ref<SelectPassThroughOptions>({
    root: ({ props }) => ({
        class: `relative inline-flex cursor-pointer select-none
            ${props.disabled ? 'opacity-50 cursor-not-allowed' : ''}
            ${props.invalid ? 'border-red-500 dark:border-red-400 focus-within:border-red-500 dark:focus-within:border-red-400 focus-within:ring-red-500 dark:focus-within:ring-red-400' : ''}`
    }),
    label: ({ props, state }) => ({
        class: `flex-auto block overflow-hidden text-ellipsis whitespace-nowrap cursor-pointer
            px-3 py-2 rounded-l-md
            bg-white dark:bg-gray-800
            border border-r-0 border-gray-300 dark:border-gray-600
            text-gray-900 dark:text-gray-100
            transition-all duration-200
            ${state.focused ? 'border-blue-500 dark:border-blue-400 ring-1 ring-blue-500 dark:ring-blue-400' : ''}
            ${!props.modelValue || (Array.isArray(props.modelValue) && props.modelValue.length === 0) ? 'text-gray-400 dark:text-gray-500' : ''}`
    }),
    dropdown: ({ state, props }) => ({
        class: `flex items-center justify-center shrink-0
            px-3 py-2 rounded-r-md
            bg-white dark:bg-gray-800
            border border-gray-300 dark:border-gray-600
            text-gray-500 dark:text-gray-400
            transition-all duration-200
            ${state.focused ? 'border-blue-500 dark:border-blue-400 ring-1 ring-blue-500 dark:ring-blue-400' : ''}
            ${!props.disabled ? 'hover:bg-gray-50 dark:hover:bg-gray-700' : ''}`
    }),
    dropdownIcon: `w-4 h-4 text-gray-600 dark:text-gray-400`,
    overlay: `absolute bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-600 rounded-lg shadow-lg dark:shadow-gray-900/50 mt-1 max-h-80 overflow-hidden z-50`,
    header: `px-3 py-2 border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800`,
    pcFilterContainer: {
        root: `relative`
    },
    pcFilter: {
        root: `w-full px-3 py-2 pr-10
            bg-white dark:bg-gray-800
            border border-gray-300 dark:border-gray-600
            rounded-md
            text-gray-900 dark:text-gray-100
            placeholder:text-gray-400 dark:placeholder:text-gray-500
            focus:outline-none focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-400 focus:border-blue-500 dark:focus:border-blue-400
            transition-all duration-200`
    },
    pcFilterIconContainer: {
        root: `absolute top-1/2 right-3 -translate-y-1/2`
    },
    filterIcon: `w-4 h-4 text-gray-400 dark:text-gray-500`,
    listContainer: `overflow-auto bg-white dark:bg-gray-800`,
    list: `py-1 list-none m-0 p-0`,
    optionGroup: `px-3 py-2 bg-gray-50 dark:bg-gray-700/50 border-b border-gray-100 dark:border-gray-700`,
    optionGroupLabel: `font-semibold text-sm text-gray-700 dark:text-gray-300`,
    option: ({ context }) => ({
        class: `relative flex items-center gap-2 px-3 py-2 m-1 mx-2 rounded-md cursor-pointer
            text-gray-900 dark:text-gray-100
            transition-all duration-200
            ${context.focused ? 'bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300' : ''}
            ${context.selected ? 'bg-blue-100 dark:bg-blue-900/40 text-blue-800 dark:text-blue-200 font-semibold' : ''}
            ${!context.disabled ? 'hover:bg-gray-100 dark:hover:bg-gray-700' : 'opacity-50 cursor-not-allowed'}
            ${context.focused && context.selected ? 'bg-blue-200 dark:bg-blue-800/50' : ''}`
    }),
    optionLabel: `flex-auto`,
    optionCheckIcon: `w-4 h-4 text-blue-600 dark:text-blue-400`,
    optionBlankIcon: `w-4 h-4`,
    emptyMessage: `px-3 py-8 text-center text-gray-500 dark:text-gray-400 bg-white dark:bg-gray-800`,
    clearIcon: `absolute top-1/2 right-12 -translate-y-1/2 w-4 h-4 text-gray-400 hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-300 cursor-pointer transition-colors duration-200`,
    loadingIcon: `w-4 h-4 animate-spin text-gray-500 dark:text-gray-400`,
    transition: {
        enterFromClass: 'opacity-0 scale-95',
        enterActiveClass: 'transition-all duration-100 ease-out',
        enterToClass: 'opacity-100 scale-100',
        leaveFromClass: 'opacity-100 scale-100',
        leaveActiveClass: 'transition-all duration-75 ease-in',
        leaveToClass: 'opacity-0 scale-95'
    }
});
</script>
