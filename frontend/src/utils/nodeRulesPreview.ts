import type { PreviewFilter, PreviewResult } from '../types/noderules'

/**
 * Makes a node-rules preview safe to render, at the one place it enters the app.
 *
 * The types say `members: string[]` and `unmatched: string[]`. The wire did not
 * always agree: Go encodes a nil slice as `null`, so a filter that matched no
 * nodes arrived as `members: null`, and the Node Rules page — which iterates
 * that field on every render — threw before it could draw anything, leaving
 * only the dialog's backdrop on screen (#14). The page was then unusable in
 * exactly the situation it is needed to repair.
 *
 * The backend now always sends arrays, but a panel is upgraded less often than
 * its frontend is reloaded, and "never trust the wire" is cheaper to state once
 * here than to remember at every `v-for`.
 */
export function normalizePreview(raw: PreviewResult | null | undefined): PreviewResult | null {
  if (!raw) return null
  const filters: PreviewFilter[] = (Array.isArray(raw.filters) ? raw.filters : []).map((filter) => {
    const members = Array.isArray(filter.members) ? filter.members : []
    // The count is derived, not trusted: the two came from the same slice on
    // the server, so if `members` had to be repaired the count describes it.
    return { ...filter, members, member_count: members.length }
  })
  return {
    ...raw,
    filters,
    unmatched: Array.isArray(raw.unmatched) ? raw.unmatched : [],
    optional: Array.isArray(raw.optional) ? raw.optional : [],
  }
}
