<template>
    <Toast
        unstyled
        :pt="theme"
        :ptOptions="{
            mergeProps: ptViewMerge
        }"
        :onMouseEnter="pauseOnHover"
        :onMouseLeave="pauseOnHover"
    >
        <template #closeicon>
            <TimesIcon />
        </template>
        <template v-for="(_, slotName) in $slots" #[slotName]="slotProps">
            <slot :name="slotName" v-bind="slotProps ?? {}" />
        </template>
    </Toast>
</template>

<script setup lang="ts">
import TimesIcon from '@primevue/icons/times';
import Toast, { type ToastPassThroughOptions, type ToastProps } from 'primevue/toast';
import { ptViewMerge } from './utils';

interface Props extends /* @vue-ignore */ ToastProps {}
defineProps<Props>();

// PrimeVue's ToastMessage only pauses its auto-dismiss timer on hover when an
// `onMouseEnter`/`onMouseLeave` handler is present (see handleMouseEnter's
// `if (this.onMouseEnter)` guard). We pass a no-op so the built-in pause/resume
// runs — the toast stays visible while the cursor is over it.
const pauseOnHover = () => {};

/*
 * Static PT theme — plain const, not a ref. `aria-label` on `closeButton`
 * is explicit because `unstyled` mode strips PrimeVue's default.
 *
 * Severity colouring (info/success/warn/error/secondary/contrast) lives in
 * `src/style/primevue.css`, keyed off the `data-p` attribute PrimeVue already
 * emits. It used to be expressed with `p-info:`-style variants from the
 * `tailwindcss-primeui` plugin — this component was that plugin's ONLY
 * consumer in the entire app, so moving ~70 variant utilities into ~30 lines
 * of CSS let the dependency go. It also let the palette switch from the
 * plugin's violet/green/yellow to the design system's own status tokens.
 */
const theme: ToastPassThroughOptions = {
    root: `volt-toast w-96 rounded-surface whitespace-pre-line break-words`,
    message: `volt-toast-message mb-4 border rounded-surface backdrop-blur-sm dark:backdrop-blur-md`,
    messageContent: `flex items-start p-3 gap-2`,
    messageIcon: `flex-shrink-0 text-lg w-[1.125rem] h-[1.125rem] mt-1`,
    // min-w-0 lets this flex child shrink below its content size so long,
    // unbroken strings (URLs, "ip:port->ip:port") wrap instead of overflowing
    // the fixed-width toast and spilling out of the colored background.
    messageText: `flex-auto min-w-0 flex flex-col gap-2`,
    summary: `font-medium text-base break-words`,
    detail: `font-medium text-sm opacity-90 break-words`,
    buttonContainer: ``,
    closeButton: {
        'aria-label': 'Close',
        class: `volt-toast-close flex items-center justify-center overflow-hidden relative cursor-pointer bg-transparent select-none
        transition-colors duration-200 text-inherit w-7 h-7 rounded-pill shrink-0 -mr-1 p-0 border-none
        focus-visible:outline focus-visible:outline-1 focus-visible:outline-offset-2`,
    },
    closeIcon: `text-base w-4 h-4`,
    transition: {
        enterFromClass: 'opacity-0 translate-y-1/2',
        enterActiveClass: 'transition-all duration-500',
        leaveFromClass: 'max-h-[1000px]',
        leaveActiveClass: 'transition-all duration-500',
        leaveToClass: 'max-h-0 opacity-0 mb-0 overflow-hidden'
    }
};
</script>
