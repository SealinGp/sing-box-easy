import type { RuntimeGroup, RuntimeProxies } from '../types/runtime'
import type { NodeQualityRules } from '../utils/nodeQuality'
import { nodeDelayPoints } from '../utils/nodeDelayHistory'

function demoCandidates(group: RuntimeGroup, rules: NodeQualityRules, now: number) {
  return group.members.flatMap(member => {
    if (member.group || !member.type || /^(direct|reject|block|dns|selector|urltest|fallback)$/i.test(member.type)) return []
    const points = nodeDelayPoints(member.history).slice(-rules.samples)
    const last = points.at(-1)
    if (points.length < rules.samples || !last?.reachable || last.avg_ms <= 0) return []
    const age = now - Date.parse(last.at)
    if (age > 20 * 60000 || age < -60000) return []
    const delays = points.filter(p => p.reachable && p.avg_ms > 0).map(p => p.avg_ms).sort((a, b) => a - b)
    const passed = delays.filter(d => d <= rules.max_delay_ms).length
    return [{ member, passed, successes: delays.length, p95: delays[Math.ceil(delays.length * .95) - 1]! }]
  })
}

/** Browser-only fixture behavior, not a background production scheduler. */
export function evaluateDemoSelection(view: RuntimeProxies, rules: NodeQualityRules) {
  const now = Date.now()
  for (const group of view.groups) {
    const state = group.auto_selection
    if (!group.switchable || !state?.enabled || (state.next_check && Date.parse(state.next_check) > now)) continue
    state.last_checked = new Date(now).toISOString()
    state.next_check = new Date(now + state.interval_seconds * 1000).toISOString()
    const measured = demoCandidates(group, rules, now)
    if (!measured.length) {
      state.reason = 'no_candidate'
      continue
    }
    const best = Math.max(...measured.map(c => c.passed))
    const fallback = best * 100 < rules.good_percent * rules.samples
    const candidates = measured.filter(c => fallback
      ? (best === 0 || c.passed > 0) && (best - c.passed) * 100 <= rules.samples
      : c.passed * 100 >= rules.good_percent * rules.samples)
    const incumbent = candidates.find(c => c.member.name === group.now)
    const selected = (!fallback && incumbent) || candidates.sort((a, b) => {
      if (!fallback && a.passed !== b.passed) return b.passed - a.passed
      if (!best && a.successes !== b.successes) return b.successes - a.successes
      return a.p95 - b.p95 || Number(b === incumbent) - Number(a === incumbent) || a.member.name.localeCompare(b.member.name)
    })[0]!
    if (selected.member.name === group.now) {
      state.reason = fallback ? 'fallback_retained' : 'retained'
    } else {
      group.now = selected.member.name
      group.delay = selected.member.delay
      state.reason = fallback ? 'fallback_switched' : 'switched'
      state.last_switched = state.last_checked
    }
    state.selected = group.now
  }
}
