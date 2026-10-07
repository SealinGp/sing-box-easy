/**
 * Builds the "Report a bug" link: a GitHub new-issue URL that opens the
 * repository's bug-report form with the About information already filled in.
 *
 * WHY A LINK AND NOT AN API CALL
 * ──────────────────────────────
 * The panel never files anything. It opens GitHub in the operator's own
 * browser, signed in as themselves, with a form they read and submit — so
 * there is no token to hold, nothing leaves the machine without being seen
 * first, and the issue belongs to the person who can answer follow-ups.
 *
 * GitHub issue forms accept a pre-filled value for any field through a query
 * parameter named after the field's `id`. Those ids are therefore a contract
 * with `.github/ISSUE_TEMPLATE/bug_report.yml`; `bugReport.test.ts` reads that
 * file and fails if one this module uses is missing from it.
 *
 * WHAT IS DELIBERATELY NOT SENT
 * ─────────────────────────────
 * The hostname, any address, and anything from config.json. An issue is
 * public and permanent; a hostname can name a person or a network, and nothing
 * in it helps diagnose a bug that the fields below do not already say.
 */

export const PROJECT_URL = 'https://github.com/SealinGp/sing-box-easy'

/** File name of the issue form this link opens. */
export const BUG_TEMPLATE = 'bug_report.yml'

/** Field ids in the issue form that this module pre-fills. */
export const BUG_FIELDS = {
  panelVersion: 'panel_version',
  coreVersion: 'core_version',
  environment: 'environment',
  browser: 'browser',
} as const

/** What a report is built from. Every field is optional: the fetch may have failed. */
export interface Diagnostics {
  panelVersion?: string
  coreVersion?: string
  /** "iStoreOS 24.10.8", or '' when the host reports none. */
  distribution?: string
  /** "openwrt" / "debian" / "unknown". */
  systemType?: string
  os?: string
  arch?: string
  cpuCores?: number
  kernel?: string
  serviceBackend?: string
  /** "tarball" / "opkg". */
  installMethod?: string
  /** Whether login is required on this deployment. */
  authEnabled?: boolean
  userAgent?: string
}

/**
 * GitHub answers a new-issue URL that is too long with an error page rather
 * than a truncated form, which would lose the report entirely. The fields here
 * are short, but a kernel string or user agent is not bounded by us.
 */
const FIELD_MAX = 300
const clip = (value: string) => (value.length > FIELD_MAX ? `${value.slice(0, FIELD_MAX - 1)}…` : value)
const clean = (value: string | undefined) => clip((value ?? '').replace(/\s+/g, ' ').trim())

/**
 * The environment as `Label: value` lines. Lines with nothing to say are
 * omitted, so a host that reports little produces a short block rather than a
 * column of blanks. English labels always: the block is read by maintainers.
 */
export function environmentLines(diagnostics: Diagnostics): string[] {
  const lines: string[] = []
  const push = (label: string, value: string) => {
    if (value) lines.push(`${label}: ${value}`)
  }

  const distribution = clean(diagnostics.distribution)
  const family = clean(diagnostics.systemType)
  const knownFamily = family && family !== 'unknown' ? family : ''
  push('System', distribution && knownFamily ? `${distribution} (${knownFamily})` : distribution || knownFamily)

  const os = clean(diagnostics.os)
  const arch = clean(diagnostics.arch)
  const target = [os, arch].filter(Boolean).join('/')
  const cores = diagnostics.cpuCores && diagnostics.cpuCores > 0 ? `${diagnostics.cpuCores} cores` : ''
  push('Platform', [target, cores].filter(Boolean).join(', '))

  push('Kernel', clean(diagnostics.kernel))
  push('Service manager', clean(diagnostics.serviceBackend))
  push('Install method', clean(diagnostics.installMethod))
  if (diagnostics.authEnabled !== undefined) push('Login required', diagnostics.authEnabled ? 'yes' : 'no')
  return lines
}

/**
 * The same information as plain text, for pasting into an existing issue or a
 * chat — and the fallback when the link cannot pre-fill (see `bugReportUrl`).
 */
export function diagnosticsText(diagnostics: Diagnostics): string {
  const lines: string[] = []
  const panel = clean(diagnostics.panelVersion)
  const core = clean(diagnostics.coreVersion)
  if (panel) lines.push(`sing-box-easy: ${panel}`)
  if (core) lines.push(`sing-box: ${core}`)
  lines.push(...environmentLines(diagnostics))
  const browser = clean(diagnostics.userAgent)
  if (browser) lines.push(`Browser: ${browser}`)
  return lines.join('\n')
}

/**
 * The new-issue URL with the form pre-filled.
 *
 * A value that is unknown is left out rather than sent empty, so the form
 * shows its placeholder and its "required" check still does its job.
 */
export function bugReportUrl(diagnostics: Diagnostics): string {
  const params = new URLSearchParams({ template: BUG_TEMPLATE })
  const set = (field: string, value: string) => {
    if (value) params.set(field, value)
  }
  set(BUG_FIELDS.panelVersion, clean(diagnostics.panelVersion))
  set(BUG_FIELDS.coreVersion, clean(diagnostics.coreVersion))
  set(BUG_FIELDS.environment, environmentLines(diagnostics).join('\n'))
  set(BUG_FIELDS.browser, clean(diagnostics.userAgent))
  return `${PROJECT_URL}/issues/new?${params.toString()}`
}
