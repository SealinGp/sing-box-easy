import type { RuntimeProxies } from '../types/runtime'
import { rateNode, type NodeQualityRules } from '../utils/nodeQuality'

/** Browser-only fixture behavior, not a background production scheduler. */
export function evaluateDemoSelection(view: RuntimeProxies, rules: NodeQualityRules) {
  for (const group of view.groups) {
    const state = group.auto_selection
    if (!state?.enabled || (state.next_check && Date.parse(state.next_check) > Date.now())) continue
    state.last_checked = new Date().toISOString()
    state.next_check = new Date(Date.now() + state.interval_seconds * 1000).toISOString()
    const eligible = group.members.filter(m => !m.group && rateNode(m.history, rules).level === 'good' && m.history?.at(-1)?.success)
    if (eligible.some(m => m.name === group.now)) state.reason = 'retained'
    else if (eligible.length) {
      group.now = eligible[0]!.name
      state.reason = 'switched'; state.last_switched = new Date().toISOString()
    } else state.reason = 'no_candidate'
    state.selected = group.now
  }
}
