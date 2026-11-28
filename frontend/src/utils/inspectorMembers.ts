import type { RuntimeMember } from '../types/runtime'

/** Stable partition: keep the active member first without mutating runtime data. */
export function inspectorMembers(members: RuntimeMember[], current: string, query: string): RuntimeMember[] {
  const needle = query.trim().toLocaleLowerCase()
  const matches = members.filter(member => !needle || member.name.toLocaleLowerCase().includes(needle))
  return [...matches.filter(member => member.name === current), ...matches.filter(member => member.name !== current)]
}
