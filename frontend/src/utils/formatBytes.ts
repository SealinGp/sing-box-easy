/** Binary units — this formats filesystem sizes, which `df -h` also reports in KiB steps. */
const UNITS = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'] as const

const PLACEHOLDER = '—'

/**
 * Formats a byte count for display, e.g. 31350000000 -> "29.2 GB".
 *
 * Returns a placeholder rather than "NaN B" for values that are not usable
 * numbers, so a backend that could not stat a filesystem never renders garbage.
 */
export function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes < 0) return PLACEHOLDER
  if (bytes < 1024) return `${bytes} B`

  let value = bytes
  let unit = 0
  while (value >= 1024 && unit < UNITS.length - 1) {
    value /= 1024
    unit += 1
  }

  // One decimal is enough to distinguish "0.4 GB left" from "4 GB left";
  // a trailing .0 is noise, so it is dropped.
  const rounded = value.toFixed(1).replace(/\.0$/, '')
  return `${rounded} ${UNITS[unit]}`
}
