<script setup lang="ts">
import { ListBulletIcon, Squares2X2Icon } from '@heroicons/vue/24/outline'
import { useI18n } from 'vue-i18n'
import TooltipTarget from './TooltipTarget.vue'
defineProps<{ modelValue: 'list' | 'cards' }>()
const emit = defineEmits<{ 'update:modelValue': [value: 'list' | 'cards'] }>()
const { t } = useI18n()
</script>
<template>
  <div class="collection-view-toggle" :class="{ 'is-cards': modelValue === 'cards' }" role="group" :aria-label="t('common.collection.view')">
    <span class="collection-view-slider" aria-hidden="true" />
    <TooltipTarget as="button" v-for="view in (['list', 'cards'] as const)" :key="view" type="button"
      :aria-label="t(`common.collection.${view}`)" :text="t(`common.collection.${view}`)"
      :aria-pressed="modelValue === view" @click="emit('update:modelValue', view)">
      <component :is="view === 'list' ? ListBulletIcon : Squares2X2Icon" class="h-5 w-5" aria-hidden="true" />
    </TooltipTarget>
  </div>
</template>
<style>
.collection-view-toggle { position: relative; isolation: isolate; display: inline-flex; flex-shrink: 0; padding: 3px; gap: 2px; border: 1px solid var(--color-border); border-radius: var(--radius-control); background: var(--color-bg-tertiary); }
.collection-view-slider { position: absolute; left: 3px; top: 3px; width: 32px; height: 28px; border-radius: max(2px, calc(var(--radius-control) - 3px)); background: var(--color-bg-secondary); box-shadow: 0 1px 3px rgb(0 0 0 / 10%), inset 0 0 0 1px var(--color-border); pointer-events: none; transition: transform 220ms var(--motion-continuous, cubic-bezier(.4, 0, .2, 1)); }
.collection-view-toggle.is-cards .collection-view-slider { transform: translateX(34px); }
.collection-view-toggle button { position: relative; z-index: 1; display: inline-flex; align-items: center; justify-content: center; width: 32px; height: 28px; border-radius: max(2px, calc(var(--radius-control) - 3px)); background: transparent; cursor: pointer; color: var(--color-text-secondary); transition: color 140ms ease; }
.collection-view-toggle button:hover { color: var(--color-text-primary); }
.collection-view-toggle button:focus-visible { outline: 2px solid var(--color-primary); outline-offset: 2px; }
.collection-view-toggle button[aria-pressed="true"] { color: var(--color-primary); }
@media (prefers-reduced-motion: reduce) { .collection-view-toggle button, .collection-view-slider { transition: none; } }
</style>
