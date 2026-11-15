<script setup lang="ts">
import { ref, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { EllipsisHorizontalIcon } from '@heroicons/vue/24/outline'
import TooltipTarget from './TooltipTarget.vue'
import Popover from '../volt/Popover.vue'
const { t } = useI18n()
const panel = ref<InstanceType<typeof Popover>>()
const open = ref(false)
async function act(event: MouseEvent) {
  const button = (event.target as Element).closest('button')
  // Confirmation popovers keep their trigger available until resolved.
  if (!button || button.disabled || button.hasAttribute('aria-haspopup')) return
  await nextTick()
  panel.value?.hide()
}
function navigate(event: KeyboardEvent) {
  if (!['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(event.key)) return
  const buttons = [...(event.currentTarget as HTMLElement).querySelectorAll<HTMLButtonElement>('button:not(:disabled)')]
  if (!buttons.length) return
  event.preventDefault()
  const index = buttons.indexOf(document.activeElement as HTMLButtonElement)
  const next = event.key === 'Home' ? 0 : event.key === 'End' ? buttons.length - 1
    : (index + (event.key === 'ArrowDown' ? 1 : -1) + buttons.length) % buttons.length
  buttons[next]?.focus()
}
</script>
<template>
  <div class="card-actions">
    <TooltipTarget as="button" type="button" class="card-actions-trigger" :text="t('common.moreActions')" :aria-label="t('common.moreActions')"
      aria-haspopup="dialog" :aria-expanded="open" @click.stop="panel?.toggle($event)" @mousedown.stop>
      <EllipsisHorizontalIcon class="h-5 w-5" aria-hidden="true" />
    </TooltipTarget>
    <Popover ref="panel" interaction="menu" :aria-label="t('common.moreActions')" panel-class="card-actions-panel" content-class="p-1.5"
      @show="open = true" @hide="open = false">
      <div class="card-actions-items" role="group" :aria-label="t('common.moreActions')" @click="act" @keydown="navigate"><slot /></div>
    </Popover>
  </div>
</template>
<style>
.card-actions { position: absolute; right: 16px; top: 16px; }
.card-actions-trigger { width: 32px; height: 32px; display: flex; align-items: center; justify-content: center; color: var(--color-text-secondary); border-radius: var(--radius-control); cursor: pointer; }
.card-actions-trigger:hover, .card-actions-trigger[aria-expanded="true"] { background: var(--color-bg-tertiary); color: var(--color-text-primary); }
.card-actions-trigger:focus-visible { outline: 2px solid var(--color-primary); outline-offset: 2px; }
.card-actions-panel { width: 208px !important; }
.card-actions-items > div { display: flex; flex-direction: column; align-items: stretch; gap: 2px; }
.card-actions-items button { justify-content: flex-start; gap: 8px; width: 100%; min-height: 36px; height: auto; padding: 6px 10px; font-size: 13px; box-shadow: none; }
.card-actions-items > div > span { display: flex; }
.card-actions-items .semantic-action-label { display: inline; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.card-actions-items .copy-action { width: 100%; height: auto; }
.card-actions-items .copy-action::after { content: attr(aria-label); }
.card-actions-items button:has(> svg:only-child)::after,
.card-actions-items button:has(> .ui-control-content > svg:only-child)::after { content: attr(aria-label); }
</style>
