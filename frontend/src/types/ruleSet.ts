/**
 * Rule-set status, shared by the DNS and route probes.
 *
 * One definition rather than two near-identical ones, because the two probes
 * answer the same kind of question and the backend deliberately emits the same
 * shape for both. Two copies would drift, and the drift would be invisible:
 * both sides would keep compiling while one quietly stopped rendering a field.
 */

/** Why a rule set could not be consulted. Keys, translated by the UI. */
export type RuleSetReason =
  | 'unknown_tag'
  | 'not_cached'
  | 'cache_unavailable'
  | 'cache_disabled'
  | 'file_missing'
  | 'unsupported_srs_version'
  | 'parse_error'

/**
 * Which layer decided the set.
 *
 * `sing-box` means the panel could not decode the set itself and asked the
 * installed binary — which is the only thing that can read a cache written by
 * a newer sing-box than the one this panel is built against. Worth surfacing:
 * it is the sole visible sign that the two versions have drifted apart.
 */
export type RuleSetTier = '' | 'decoded' | 'sing-box'

export interface RuleSetStatus {
  tag: string
  /** The set's own verdict against the probed domain or address. */
  state: 'matched' | 'not_matched' | 'unevaluated'
  reason?: RuleSetReason
  detail?: string
  /** When sing-box last downloaded the set. A surprise is often a stale set. */
  updated_at_unix?: number
  tier?: RuleSetTier
}
