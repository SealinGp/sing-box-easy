// Outbound Node Rules types — mirror app/pkg/noderules domain structs.

export type MatcherType = 'keyword' | 'code' | 'emoji'

export interface Matcher {
  type: MatcherType
  value: string
}

export type FilterOutboundType = 'urltest' | 'selector'

// Filter = the user's "Region Node": a keyword/tag-matched bucket of endpoints.
export interface Filter {
  id: string
  name: string
  matchers: Matcher[]
  // excludes: deny-list — a node matched by `matchers` is still kept out of this
  // filter when it also matches an exclude (e.g. match code "US", except the
  // node tagged "relay_bwh_us1").
  excludes: Matcher[]
  outbound_type: FilterOutboundType
  priority: number
  is_fallback: boolean
  // urltest health-check settings (ignored when outbound_type is 'selector').
  test_url: string
  test_interval: string
  test_tolerance: number
  created_at?: string
  updated_at?: string
}

// Group = the user's "Group Node": an ordered set of Filters.
export interface Group {
  id: string
  name: string
  filter_ids: string[]
  priority: number
  created_at?: string
  updated_at?: string
}

export interface KeywordEntry {
  code: string
  label: string
  synonyms: string[]
}

export interface FilterTemplate {
  id: string
  name: string
  description: string
  outbound_type: FilterOutboundType
  matchers: Matcher[]
}

export interface PreviewFilter {
  id: string
  name: string
  outbound_type: string
  is_fallback: boolean
  member_count: number
  members: string[]
}

export interface PreviewResult {
  endpoints: number
  filters: PreviewFilter[]
  unmatched: string[]
}

// Payloads for create/update.
export interface FilterInput {
  name: string
  matchers: Matcher[]
  excludes: Matcher[]
  outbound_type: FilterOutboundType
  priority: number
  test_url: string
  test_interval: string
  test_tolerance: number
}

// Defaults for a new urltest filter (mirror app/pkg/noderules defaults).
export const URLTEST_DEFAULTS = {
  test_url: 'http://www.gstatic.com/generate_204',
  test_interval: '10s',
  test_tolerance: 200,
} as const

export interface GroupInput {
  name: string
  filter_ids: string[]
  priority: number
}
