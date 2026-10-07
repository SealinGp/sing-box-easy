import type { PackageInstallResult, VersionStatus } from '../types/version'

/**
 * Deciding when a panel-driven opkg install is over.
 *
 * The request that starts the install is answered by a process the install
 * then stops, so the outcome has to be read off whichever panel answers next.
 * Two signals, because either can be missing:
 *
 *  - `last_package_install`, written by the install helper. Authoritative, but
 *    a target release older than this feature does not report it.
 *  - the running version, which covers that case.
 */
export type PackageInstallVerdict = 'pending' | 'succeeded' | 'failed'

/** "v1.5.4" and "1.5.4" are the same release. */
const normalize = (version: string | undefined | null): string =>
  (version ?? '').trim().replace(/^v/i, '')

const sameVersion = (a: string | undefined | null, b: string | undefined | null): boolean =>
  normalize(a) !== '' && normalize(a) === normalize(b)

export interface PackageInstallProbe {
  /** The `GET /version` answer, from whichever panel process replied. */
  status: Pick<VersionStatus, 'current_version' | 'last_package_install'>
  /** The version the install was asked for. */
  target: string
  /** The version that was running when the install was started. */
  from: string
  /**
   * True once a probe has failed, i.e. the panel was seen to go away. Only
   * needed when `from` and `target` are the same release (a reinstall): there
   * the version cannot change, and the OLD panel answering with it would read
   * as finished before the install began. It must not be REQUIRED otherwise —
   * the panel is down for about two seconds, which a two-second probe misses
   * as often as not (seen on a real router: the page waited forever on an
   * install that had long since succeeded).
   */
  sawOutage: boolean
}

export function packageInstallVerdict(probe: PackageInstallProbe): PackageInstallVerdict {
  const result = probe.status.last_package_install
  // A result for some other run (an earlier install this boot) says nothing
  // about this one.
  if (result && sameVersion(result.to_version, probe.target)) {
    if (result.state === 'succeeded') return 'succeeded'
    if (result.state === 'failed' || result.state === 'interrupted') return 'failed'
    return 'pending'
  }
  if (sameVersion(probe.status.current_version, probe.target)) {
    const versionChanged = !sameVersion(probe.from, probe.target)
    if (versionChanged || probe.sawOutage) return 'succeeded'
  }
  return 'pending'
}

/** What to show for a failed install: opkg's own output when there is any. */
export function packageInstallFailureText(result: PackageInstallResult | null | undefined): string {
  if (!result) return ''
  const header =
    result.exit_code >= 0 ? `opkg exited with ${result.exit_code}.` : 'The install did not finish.'
  return result.log_tail ? `${header}\n${result.log_tail}` : header
}
