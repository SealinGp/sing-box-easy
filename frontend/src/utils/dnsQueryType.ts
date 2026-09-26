/**
 * The DNS rule `query_type` condition.
 *
 * On the wire it is `badoption.Listable[DNSQueryType]` (sing-box
 * `option/types.go`), and DNSQueryType has an asymmetric JSON shape worth
 * pinning down:
 *
 * - decode accepts a NUMBER, or a NAME found in miekg/dns `StringToType`.
 *   A numeric STRING ("32768") is neither, and fails with
 *   "unknown DNS query type".
 * - encode writes the name when one exists and the bare number otherwise.
 *
 * The form edits everything as strings, so the two helpers below convert at
 * the boundary: numbers become strings on read, numeric strings become
 * numbers on write.
 *
 * The field is not new in 1.14 — it is present in the pinned 1.12.12 option
 * struct — so it is offered on every core without a version gate.
 */

/** The types worth offering by name, most common first. */
export const COMMON_DNS_QUERY_TYPES = [
  'A',
  'AAAA',
  'HTTPS',
  'SVCB',
  'CNAME',
  'MX',
  'TXT',
  'NS',
  'PTR',
  'SRV',
  'SOA',
  'CAA',
  'ANY',
] as const

export interface QueryTypeOption {
  value: string
  label: string
}

const NUMERIC = /^\d+$/

/** Wire value -> the string list the form binds. */
export function readQueryTypes(raw: unknown): string[] {
  if (raw === undefined || raw === null) return []
  const list = Array.isArray(raw) ? raw : [raw]
  return list.map((entry) => String(entry))
}

/** Form strings -> wire values sing-box will decode. */
export function writeQueryTypes(values: readonly string[]): Array<string | number> {
  const seen = new Set<string>()
  const out: Array<string | number> = []
  for (const value of values) {
    const trimmed = value.trim()
    if (!trimmed || seen.has(trimmed)) continue
    seen.add(trimmed)
    out.push(NUMERIC.test(trimmed) ? Number(trimmed) : trimmed)
  }
  return out
}

/**
 * The select's options: the common types, plus any selected value that is not
 * among them. Without the second part a rule carrying, say, `32768` or `DS`
 * would show nothing while still matching on it, and the next save would look
 * like the operator had removed it.
 */
export function queryTypeOptions(selected: readonly string[]): QueryTypeOption[] {
  const known = new Set<string>(COMMON_DNS_QUERY_TYPES)
  const extras = selected.filter((value) => !known.has(value))
  return [...COMMON_DNS_QUERY_TYPES, ...new Set(extras)].map((value) => ({ value, label: value }))
}
