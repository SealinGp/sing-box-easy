<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Dialog } from '../volt'
import Button from './Button.vue'
import { useConfirm } from '../composables/useConfirm'

/**
 * Global confirmation dialog. Mount exactly once (in App.vue). It renders the
 * shared state owned by useConfirm() and resolves the in-flight promise when
 * the user confirms or cancels. Replaces native window.confirm() across the app.
 */

const { t } = useI18n()
const { state, handleConfirm, handleCancel } = useConfirm()

// The volt Dialog passes `update:visible` through from PrimeVue. Any close
// gesture (mask click, ESC, header X) flows here and is treated as a cancel.
function onVisibilityChange(visible: boolean) {
  if (!visible) handleCancel()
}
</script>

<template>
  <Dialog
    :visible="state.visible"
    @update:visible="onVisibilityChange"
    modal
    :header="state.title || t('common.confirmTitle')"
    class="w-full max-w-md"
  >
    <p class="text-sm text-gray-700 dark:text-gray-300 whitespace-pre-line">
      {{ state.message }}
    </p>

    <template #footer>
      <Button
        :label="state.cancelLabel || t('common.cancel')"
        severity="secondary"
        @click="handleCancel"
      />
      <Button
        :label="state.confirmLabel || t('common.confirm')"
        :severity="state.tone === 'danger' ? 'danger' : 'primary'"
        @click="handleConfirm"
      />
    </template>
  </Dialog>
</template>
