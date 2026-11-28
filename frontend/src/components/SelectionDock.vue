<script setup lang="ts">
import { XMarkIcon, ClipboardDocumentIcon } from '@heroicons/vue/24/outline'
import { useI18n } from 'vue-i18n'
import TooltipTarget from './TooltipTarget.vue'

withDefaults(defineProps<{ count: number; busy?: boolean; copied?: boolean; error?: string }>(), { busy: false })
defineEmits<{ clear: []; copy: [] }>()
const { t } = useI18n()
</script>

<template>
  <Teleport to="body">
    <Transition name="selection-dock">
      <section v-if="count" class="selection-dock liquid-glass-float" role="region"
        :aria-label="t('common.collection.actions')" :aria-busy="busy">
        <div class="selection-dock-controls">
          <span class="shrink-0 text-sm font-medium" role="status">{{ t('common.collection.selected', { count }) }}</span>
          <div class="selection-dock-actions"><slot>
            <TooltipTarget as="button" type="button" class="collection-tool" :disabled="busy"
              :text="copied ? t('common.copied') : t('common.copy')" :aria-label="t('common.copy')" @click="$emit('copy')">
              <ClipboardDocumentIcon aria-hidden="true" />
              <span v-if="copied" role="status">{{ t('common.copied') }}</span>
            </TooltipTarget>
          </slot></div>
          <TooltipTarget as="button" type="button" class="collection-tool shrink-0" :disabled="busy"
            :text="t('common.collection.clear')" :aria-label="t('common.collection.clear')" @click="$emit('clear')">
            <XMarkIcon aria-hidden="true" />
          </TooltipTarget>
        </div>
        <p v-if="error" role="alert" class="px-2 pb-1 text-sm text-red-600 dark:text-red-400">{{ error }}</p>
      </section>
    </Transition>
  </Teleport>
</template>

<style>
.selection-dock { position: fixed; z-index: 40; left: 50%; bottom: max(20px, env(safe-area-inset-bottom)); transform: translateX(-50%); width: max-content; max-width: calc(100vw - 24px); border: 1px solid var(--color-border); border-radius: 18px; padding: 8px 12px; color: var(--color-text-primary); box-shadow: 0 12px 40px #0003; }
.selection-dock-controls, .selection-dock-actions { display: flex; align-items: center; gap: 12px; min-width: 0; }
.selection-dock-actions { flex: 1; flex-wrap: wrap; }
.selection-dock .collection-tool:disabled { opacity: .45; cursor: wait; }
.selection-dock-enter-active, .selection-dock-leave-active { transition: opacity var(--motion-feedback) var(--motion-continuous), translate var(--motion-feedback) var(--motion-continuous); }
.selection-dock-enter-from, .selection-dock-leave-to { opacity: 0; translate: 0 10px; }
@media (max-width: 520px) { .selection-dock-controls { flex-wrap: wrap; gap: 8px; } .selection-dock-actions { gap: 8px; } }
@media (prefers-reduced-motion: reduce) { .selection-dock-enter-active, .selection-dock-leave-active { transition: none; } }
</style>
