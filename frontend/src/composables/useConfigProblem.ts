import { readonly, ref } from 'vue'
import { parseConfigProblems, type ConfigProblem } from '../utils/configProblems'

/**
 * Shared state behind the app-wide config-problem dialog.
 *
 * Module-level, like `useConfirm`: the report comes from the API client — one
 * place that sees every response — and the dialog is mounted once in App.vue,
 * so neither can own the state and neither should have to know the other.
 * Nothing here imports the API client, which is what lets the client import
 * this without a cycle.
 */
const problems = ref<ConfigProblem[]>([])
const message = ref('')
const visible = ref(false)

/**
 * Called by the API client for every failed response. Opens the dialog when
 * the body describes broken outbound references and returns whether it did.
 *
 * Why a dialog and not only the caller's toast: this error is raised from a
 * dozen places — saving a route, applying node rules, deleting an outbound,
 * starting the service — and the fix is always on a DIFFERENT page from the
 * one that triggered it. A toast can say what is wrong; it cannot take the
 * operator there.
 */
export function reportConfigProblem(data: unknown, text: string): boolean {
  const found = parseConfigProblems(data)
  if (found.length === 0) return false
  problems.value = found
  message.value = text
  visible.value = true
  return true
}

export function useConfigProblem() {
  const dismiss = () => {
    visible.value = false
  }
  return { problems: readonly(problems), message: readonly(message), visible, dismiss }
}
