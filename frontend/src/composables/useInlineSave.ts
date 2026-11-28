import { getCurrentInstance, inject, ref } from 'vue'
import { workspaceKey } from './workspaceContext'

/** Saving keeps the form mounted and errors next to the submitting control. */
export function useInlineSave() {
  const workspace = getCurrentInstance() ? inject(workspaceKey, null) : null
  const saving = ref(false)
  const saveError = ref('')
  const savedMessage = ref('')

  function clearSaveFeedback() {
    saveError.value = ''
    savedMessage.value = ''
  }

  async function persist(operation: () => Promise<unknown>, success: string, fallback: string) {
    if (saving.value) return false
    clearSaveFeedback()
    const version = workspace?.editVersion.value
    saving.value = true
    try {
      await operation()
      if (workspace && workspace.editVersion.value === version) workspace.inlineDirty.value = false
      savedMessage.value = success
      return true
    } catch (error) {
      saveError.value = error instanceof Error && error.message.trim() ? error.message : fallback
      return false
    } finally {
      saving.value = false
    }
  }

  return { saving, saveError, savedMessage, clearSaveFeedback, persist }
}
