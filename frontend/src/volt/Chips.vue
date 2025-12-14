<template>
    <Chips
        unstyled
        :pt="theme"
        :ptOptions="{
            mergeProps: ptViewMerge
        }"
    >
        <template v-for="(_, slotName) in $slots" #[slotName]="slotProps">
            <slot :name="slotName" v-bind="slotProps ?? {}" />
        </template>
    </Chips>
</template>

<script setup lang="ts">
import Chips, { type ChipsPassThroughOptions, type ChipsProps } from 'primevue/chips';
import { ref } from 'vue';
import { ptViewMerge } from './utils';

interface Props extends /* @vue-ignore */ ChipsProps {}
defineProps<Props>();

const theme = ref<ChipsPassThroughOptions>({
    root: ({ props }) => ({
        class: `w-full p-2
            bg-white dark:bg-gray-700
            border border-gray-300 dark:border-gray-600
            rounded-md
            transition-colors duration-200
            ${props.disabled ? 'opacity-50 cursor-not-allowed bg-gray-50 dark:bg-gray-800' : ''}
            focus-within:ring-2 focus-within:ring-blue-500 focus-within:border-blue-500`
    }),
    input: `flex items-center flex-wrap gap-2 list-none m-0 p-0`,
    inputitem: `flex-1 inline-flex`,
    inputitemfield: ({ props }) => ({
        class: `w-full border-0 outline-none bg-transparent p-1 m-0
            text-gray-900 dark:text-gray-100
            placeholder:text-gray-400 dark:placeholder:text-gray-500
            ${props.disabled ? 'cursor-not-allowed' : ''}`
    }),
    chipitem: `mr-2`,
    pcChip: {
        root: `inline-flex items-center gap-2 px-3 py-1.5 rounded-md
            bg-blue-100 dark:bg-blue-900/30
            text-blue-700 dark:text-blue-300
            text-sm border-0`,
        label: `text-sm font-medium`,
        removeIcon: `w-4 h-4 ml-1.5 cursor-pointer hover:text-blue-900 dark:hover:text-blue-100 transition-colors`
    }
});
</script>
