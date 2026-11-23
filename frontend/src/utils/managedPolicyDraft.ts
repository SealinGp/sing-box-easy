import type { ManagedPolicy, ManagedSnapshot } from '../services/trafficPolicy'

function canonical(value: unknown): string {
  if (Array.isArray(value)) return `[${value.map(canonical).join(',')}]`
  if (value && typeof value === 'object') {
    return `{${Object.entries(value).sort(([a], [b]) => a.localeCompare(b))
      .map(([key, item]) => `${JSON.stringify(key)}:${canonical(item)}`).join(',')}}`
  }
  return JSON.stringify(value) ?? 'null'
}

// Stored insertion positions become stale when another policy is inserted above.
// Editing must start from the block's current position, not its creation index.
export function managedPolicyDraft(policy: ManagedPolicy, snapshot: ManagedSnapshot): ManagedPolicy {
  const draft: ManagedPolicy = JSON.parse(JSON.stringify(policy))
  for (const [rules, members, key] of [
    [snapshot.route_rules, policy.route_members, 'route_index'],
    [snapshot.dns_rules, policy.dns_members, 'dns_index'],
  ] as const) {
    if (!members?.length) continue
    const matches = rules.flatMap((_, index) =>
      canonical(rules.slice(index, index + members.length)) === canonical(members) ? [index] : [])
    if (matches.length === 1) draft[key] = matches[0]!
  }
  return draft
}
