<template>
    <Button
        unstyled
        :pt="theme"
        :ptOptions="{
            mergeProps: ptViewMerge
        }"
    >
        <template v-for="(_, slotName) in $slots" #[slotName]="slotProps">
            <slot :name="slotName" v-bind="slotProps ?? {}" />
        </template>
    </Button>
</template>

<script setup lang="ts">
import Button, { type ButtonPassThroughOptions, type ButtonProps } from 'primevue/button';
import { computed } from 'vue';
import { ptViewMerge } from './utils';

interface Props extends /* @vue-ignore */ ButtonProps {}
const props = defineProps<Props>();

const theme = computed<ButtonPassThroughOptions>(() => ({
    root: `inline-flex items-center justify-center gap-2 px-4 py-2 rounded-md font-medium
        transition-colors duration-200
        focus:outline-none focus:ring-2 focus:ring-offset-2 dark:focus:ring-offset-gray-800
        disabled:opacity-50 disabled:cursor-not-allowed
        ${getSeverityClasses(props.severity)}`,
    label: `flex-1 font-medium`,
    icon: `w-5 h-5`
}));

function getSeverityClasses(severity?: string): string {
    switch (severity) {
        case 'secondary':
            return `bg-white dark:bg-gray-700 text-gray-700 dark:text-gray-300
                border border-gray-300 dark:border-gray-600
                hover:bg-gray-50 dark:hover:bg-gray-600
                focus:ring-gray-500`;
        case 'success':
            return `bg-green-600 text-white
                hover:bg-green-700
                focus:ring-green-500`;
        case 'info':
            return `bg-violet-600 text-white
                hover:bg-violet-700
                focus:ring-violet-500`;
        case 'warn':
            return `bg-yellow-600 text-white
                hover:bg-yellow-700
                focus:ring-yellow-500`;
        case 'danger':
            return `bg-red-600 text-white
                hover:bg-red-700
                focus:ring-red-500`;
        case 'help':
            return `bg-purple-600 text-white
                hover:bg-purple-700
                focus:ring-purple-500`;
        case 'contrast':
            return `bg-gray-900 dark:bg-gray-100 text-white dark:text-gray-900
                hover:bg-gray-800 dark:hover:bg-gray-200
                focus:ring-gray-500`;
        default: // primary
            return `bg-violet-600 text-white
                hover:bg-violet-700
                focus:ring-violet-500`;
    }
}
</script>
