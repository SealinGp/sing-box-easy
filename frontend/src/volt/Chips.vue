<template>
    <InputChips
        unstyled
        :pt="theme"
        :ptOptions="{
            mergeProps: ptViewMerge
        }"
    >
        <template v-for="(_, slotName) in $slots" #[slotName]="slotProps">
            <slot :name="slotName" v-bind="slotProps ?? {}" />
        </template>
    </InputChips>
</template>

<script setup lang="ts">
// PrimeVue v4 deprecated `Chips` in favour of `InputChips`. The PT
// contract is mostly identical; only the token slot keys changed:
//   inputItem      → inputToken
//   inputItemField → inputTokenField
// We keep the wrapper's exported name as `Chips` (via index.ts) so existing
// call sites do not need to change.
import InputChips, {
    type InputChipsProps,
} from 'primevue/inputchips';
import { ptViewMerge } from './utils';

interface Props extends /* @vue-ignore */ InputChipsProps {}
defineProps<Props>();

const theme = {
    root: ({ props }: { props: any }) => ({
        class: `w-full p-2
            bg-white dark:bg-gray-700
            border border-gray-300 dark:border-gray-600
            rounded-control
            transition-colors duration-200
            ${props.disabled ? 'opacity-50 cursor-not-allowed bg-gray-50 dark:bg-gray-800' : ''}
            focus-within:ring-2 focus-within:ring-primary-500 focus-within:border-primary-500`,
    }),
    input: `flex items-center flex-wrap gap-2 list-none m-0 p-0 w-full`,
    // `flex-1 min-w-0` lets the typing area grow to fill the row (and the full
    // root width when empty); `inline-flex` alone shrank to the input's
    // intrinsic size, leaving a gap to the right of the box.
    inputToken: `flex-1 flex min-w-0 basis-32`,
    inputTokenField: ({ props }: { props: any }) => ({
        class: `w-full border-0 outline-none bg-transparent p-1 m-0
            text-gray-900 dark:text-gray-100
            placeholder:text-gray-400 dark:placeholder:text-gray-500
            ${props.disabled ? 'cursor-not-allowed' : ''}`,
    }),
    chipItem: `mr-2`,
    pcChip: {
        root: `inline-flex items-center gap-2 px-3 py-1.5 rounded-control
            bg-primary-100 dark:bg-primary-900/30
            text-primary-700 dark:text-primary-300
            text-sm border-0`,
        label: `text-sm font-medium`,
        removeIcon: `w-4 h-4 ml-1.5 cursor-pointer hover:text-primary-900 dark:hover:text-primary-100 transition-colors`,
    },
};
</script>
