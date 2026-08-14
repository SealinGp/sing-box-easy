<template>
    <InputChips
        unstyled
        :modelValue="modelValue"
        :separator="separator"
        :addOnBlur="addOnBlur"
        :pt="theme"
        :ptOptions="{
            mergeProps: ptViewMerge
        }"
        @update:modelValue="onUpdate"
    >
        <template v-for="(_, slotName) in $slots" #[slotName]="slotProps">
            <slot :name="slotName" v-bind="slotProps ?? {}" />
        </template>
    </InputChips>
</template>

<script setup lang="ts">
// PrimeVue v4 deprecated `Chips` in favour of `InputChips`. We keep the
// wrapper's exported name as `Chips` (via index.ts) so existing call sites do
// not need to change.
//
// PT SECTION KEYS: this component's entry field is themed through `inputItem` /
// `inputItemField` — the keys `InputChips.vue` actually calls `cx()`/`ptm()`
// with. An earlier revision used `inputToken` / `inputTokenField` (keys from a
// different PrimeVue line), which silently matched nothing: the <input> ended up
// with no theme at all, so the browser painted its own default focus ring inside
// our already-focused shell. If the entry field ever looks unstyled again, check
// these key names against `node_modules/primevue/inputchips/InputChips.vue`
// first.
import InputChips, {
    type InputChipsProps,
} from 'primevue/inputchips';
import { ptViewMerge } from './utils';

interface Props extends /* @vue-ignore */ InputChipsProps {
    modelValue?: string[];
    // Typing this character commits the current entry, and a paste is split on
    // it — so "a.com,b.com" becomes two chips.
    separator?: string;
    // Commit whatever is typed when focus leaves. Without this, typing an entry
    // and clicking "Save" straight away silently discards it.
    addOnBlur?: boolean;
}

withDefaults(defineProps<Props>(), {
    separator: ',',
    addOnBlur: true,
});

const emit = defineEmits<{
    'update:modelValue': [value: string[]];
}>();

// Sanitize centrally: entries are trimmed, blanks dropped, duplicates removed.
// Pasting " a.com, b.com" would otherwise store a chip with a leading space,
// which silently never matches anything in sing-box. Always emits a NEW array.
const onUpdate = (value: unknown) => {
    const incoming = Array.isArray(value) ? value : [];
    const seen = new Set<string>();
    const cleaned: string[] = [];

    for (const raw of incoming) {
        const entry = String(raw).trim();
        if (entry === '' || seen.has(entry)) continue;
        seen.add(entry);
        cleaned.push(entry);
    }

    emit('update:modelValue', cleaned);
};

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
    input: `flex items-center flex-wrap gap-1.5 list-none m-0 p-0 w-full`,
    // `flex-1 min-w-0` lets the typing area grow to fill the row (and the full
    // root width when empty); `inline-flex` alone shrank to the input's
    // intrinsic size, leaving a gap to the right of the box.
    inputItem: `flex-1 flex min-w-0 basis-40`,
    inputItemField: ({ props }: { props: any }) => ({
        class: `w-full border-0 outline-none ring-0 shadow-none bg-transparent p-1 m-0
            focus:border-0 focus:outline-none focus:ring-0 focus:shadow-none
            text-gray-900 dark:text-gray-100
            placeholder:text-gray-400 dark:placeholder:text-gray-500
            ${props.disabled ? 'cursor-not-allowed' : ''}`,
    }),
    // Spacing between chips comes from the list's `gap-2`; a margin here would
    // double it.
    chipItem: ``,
    // Compact pills: a rule can hold a dozen domains, and full-size chips turned
    // the field into a wall of boxes. Small enough to scan, large enough to hit.
    pcChip: {
        root: `inline-flex items-center gap-1 px-2 py-0.5 rounded-pill
            bg-primary-100 dark:bg-primary-900/40
            text-primary-700 dark:text-primary-200
            text-xs border-0`,
        label: `text-xs font-medium`,
        removeIcon: `w-3.5 h-3.5 shrink-0 cursor-pointer opacity-60 hover:opacity-100 hover:text-red-600 dark:hover:text-red-400 transition`,
    },
};
</script>
