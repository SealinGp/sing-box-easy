<template>
    <Timeline
        unstyled
        :pt="theme"
        :ptOptions="{
            mergeProps: ptViewMerge
        }"
    >
        <template v-for="(_, slotName) in $slots" #[slotName]="slotProps">
            <slot :name="slotName" v-bind="slotProps ?? {}" />
        </template>
    </Timeline>
</template>

<script setup lang="ts">
import Timeline, { type TimelinePassThroughOptions, type TimelineProps } from 'primevue/timeline';
import { ptViewMerge } from './utils';

interface Props extends /* @vue-ignore */ TimelineProps {}
defineProps<Props>();

/*
 * PrimeVue runs unstyled here (see plugins/primevue.ts), so the whole visual
 * treatment lives in these pass-through classes.
 *
 * The marker is deliberately left unstyled beyond its box: callers render their
 * own dot through the #marker slot, because a DNS step's colour carries meaning
 * (confirmed / predicted / failed) and only the caller knows which applies.
 *
 * `min-h-*` on the connector keeps short steps from collapsing the line into
 * disconnected dashes, which is what makes a timeline read as a sequence at all.
 */
const theme: TimelinePassThroughOptions = {
    root: {
        class: 'flex grow flex-col'
    },
    event: {
        class: `flex relative min-h-10
            [&:last-child]:min-h-0`
    },
    eventSeparator: {
        class: 'flex flex-col items-center shrink-0'
    },
    eventMarker: {
        class: 'flex items-center justify-center shrink-0'
    },
    eventConnector: {
        class: `grow w-px min-h-4 my-1
            bg-gray-200 dark:bg-gray-700`
    },
    eventContent: {
        class: 'grow min-w-0 pb-4 ps-3 [&:last-child]:pb-0'
    },
    eventOpposite: {
        class: 'hidden'
    }
};
</script>
