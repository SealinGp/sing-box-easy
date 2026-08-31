/**
 * Guards a provider-supplied string on its way into an `href`.
 *
 * A subscription's official link is auto-filled from the feed — a
 * `profile-web-page-url` header or an "官网：…" entry — so it is third-party
 * text that becomes a clickable link. `javascript:` and `data:` URLs execute
 * on click, so only http and https survive here.
 *
 * The backend applies the same rule when the value is stored
 * (`subscription.NormalizeOfficialURL`). This is the second half of that pair,
 * not a duplicate of it: a row written before the rule existed, or by any
 * other client of the API, still reaches this render path.
 *
 * @returns the URL to link to, or `null` when nothing safe can be made of it.
 */
export function safeExternalUrl(raw?: string | null): string | null {
  const candidate = (raw ?? '').trim()
  if (!candidate) return null

  let parsed: URL
  try {
    parsed = new URL(candidate)
  } catch {
    return null
  }

  // Scheme comparison is on the parsed protocol, never on the raw string:
  // "JavaScript:", " javascript:" and "java\tscript:" all normalize to the
  // same protocol here, and a substring check on the input would miss them.
  if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') return null

  return parsed.toString()
}

/**
 * Whether a string typed into the "official site" field is something the
 * backend will accept.
 *
 * Looser than `safeExternalUrl` on purpose: the backend promotes a schemeless
 * value to https (`subscription.NormalizeOfficialURL`), so rejecting
 * "example.com" here would refuse in the form exactly what the same field
 * accepts when a provider's feed supplies it.
 *
 * The promotion is expressed as "does it parse once https:// is prepended?"
 * rather than as a copy of the backend's host heuristics — two hand-written
 * host validators drift, one reused parser cannot. It stays safe because a
 * hostile scheme cannot survive the prefix: "https://javascript:alert(1)" has
 * "alert(1)" where a port belongs, so it fails to parse at all.
 */
export function isLinkableSiteInput(raw?: string | null): boolean {
  const candidate = (raw ?? '').trim()
  if (!candidate) return false
  if (safeExternalUrl(candidate)) return true

  const promoted = safeExternalUrl(`https://${candidate}`)
  if (!promoted) return false
  // A host with no dot is a word, not a site — and it is what a smuggled
  // scheme ("javascript:8080") degrades into once the prefix is applied.
  return new URL(promoted).hostname.includes('.')
}

/**
 * A short, human-readable form of a link — the host, without scheme or a
 * trailing slash. Used where the full URL would crowd the row out.
 */
export function displayHost(raw?: string | null): string {
  const safe = safeExternalUrl(raw)
  if (!safe) return ''
  try {
    return new URL(safe).host
  } catch {
    return ''
  }
}
