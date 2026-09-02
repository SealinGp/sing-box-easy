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
import { ptViewMerge } from './utils';

interface Props extends /* @vue-ignore */ DialogProps {}
defineProps<Props>();

// Static PT theme — plain const so we don't pay for unnecessary reactivity.
// `w-full max-w-lg` is the default panel width; callers can override via
// `class` and the ptViewMerge merger will reconcile.
const theme: DialogPassThroughOptions = {
    mask: `fixed top-0 left-0 w-full h-full flex items-center justify-center
        bg-black/45 dark:bg-black/65 backdrop-blur-sm
        transition-all duration-200`,
    // `liquid-glass-float` supplies the fill, hairline border, and elevation, so
    // the dialog matches every other floating surface instead of the flat
    // bg-white/bg-gray-800 pair it used to hard-code.
    root: `liquid-glass-float flex flex-col w-full max-w-lg max-h-[90vh] rounded-surface
        transform transition-all duration-200`,
    // Compact density pass: header / content / footer were all `px-6 py-4`.
    // This is the only dialog implementation in the app — components/Modal.vue
    // used to wrap it with `:show-header="false"` and hand-roll its own header
    // and footer inside `content`, which meant half the app's dialogs never saw
    // the header/footer rules below. Its call sites now render this directly.
    header: `flex items-center justify-between px-4 py-3
        border-b border-border`,
    title: `text-lg font-semibold text-gray-900 dark:text-gray-100`,
    headerActions: `flex items-center gap-2`,
    pcCloseButton: {
        root: {
            'aria-label': 'Close',
            class: `flex items-center justify-center w-8 h-8 rounded-control
                text-gray-500 dark:text-gray-400
                hover:bg-black/5 dark:hover:bg-white/10
                hover:text-gray-700 dark:hover:text-gray-200
                transition-colors duration-200
                focus:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 dark:focus-visible:ring-offset-gray-900`,
        },
        icon: `w-5 h-5`,
    },
    content: `px-4 py-3 overflow-y-auto flex-1
        text-gray-900 dark:text-gray-100`,
    footer: `flex items-center justify-end gap-2 px-4 py-3
        border-t border-border`,
    transition: {
        enterFromClass: 'opacity-0 scale-95',
        enterActiveClass: 'transition-all duration-200 ease-out',
        enterToClass: 'opacity-100 scale-100',
        leaveFromClass: 'opacity-100 scale-100',
        leaveActiveClass: 'transition-all duration-150 ease-in',
        leaveToClass: 'opacity-0 scale-95'
    }
};
</script>
