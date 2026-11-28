import { ref } from 'vue'

/** Saving keeps the form mounted and errors next to the submitting control. */
export function useInlineSave() {
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
    saving.value = true
    try {
      await operation()
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
