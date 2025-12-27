<template>
    <Dialog
        unstyled
        :pt="theme"
        :ptOptions="{
            mergeProps: ptViewMerge
        }"
    >
        <template v-for="(_, slotName) in $slots" #[slotName]="slotProps">
            <slot :name="slotName" v-bind="slotProps ?? {}" />
        </template>
    </Dialog>
</template>

<script setup lang="ts">
import Dialog, { type DialogPassThroughOptions, type DialogProps } from 'primevue/dialog';
import { ref } from 'vue';
import { ptViewMerge } from './utils';

interface Props extends /* @vue-ignore */ DialogProps {}
defineProps<Props>();

const theme = ref<DialogPassThroughOptions>({
    mask: `fixed top-0 left-0 w-full h-full flex items-center justify-center
        bg-black/40 backdrop-blur-sm
        transition-all duration-200`,
    root: `flex flex-col max-h-[90vh] rounded-lg shadow-xl
        bg-white dark:bg-gray-800
        border border-gray-200 dark:border-gray-700
        transform transition-all duration-200`,
    header: `flex items-center justify-between px-6 py-4 rounded-t-lg
        border-b border-gray-200 dark:border-gray-700
        bg-white dark:bg-gray-800`,
    title: `text-lg font-semibold text-gray-900 dark:text-gray-100`,
    headerActions: `flex items-center gap-2`,
    pcCloseButton: {
        root: `flex items-center justify-center w-8 h-8 rounded-md
            text-gray-500 dark:text-gray-400
            hover:bg-gray-100 dark:hover:bg-gray-700
            hover:text-gray-700 dark:hover:text-gray-200
            transition-colors duration-200
            focus:outline-none focus:ring-2 focus:ring-violet-500 focus:ring-offset-2 dark:focus:ring-offset-gray-800`,
        icon: `w-5 h-5`
    },
    content: `px-6 py-4 overflow-y-auto flex-1
        text-gray-900 dark:text-gray-100`,
    footer: `flex items-center justify-end gap-3 px-6 py-4 rounded-b-lg
        border-t border-gray-200 dark:border-gray-700
        bg-gray-50 dark:bg-gray-800/50`,
    transition: {
        enterFromClass: 'opacity-0 scale-95',
        enterActiveClass: 'transition-all duration-200 ease-out',
        enterToClass: 'opacity-100 scale-100',
        leaveFromClass: 'opacity-100 scale-100',
        leaveActiveClass: 'transition-all duration-150 ease-in',
        leaveToClass: 'opacity-0 scale-95'
    }
});
</script>
