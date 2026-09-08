/** Release tags may include a v prefix; prerelease labels remain significant. */
export function isDifferentRelease(current: string, target: string): boolean {
  const normalize = (version: string) => version.trim().replace(/^v/i, '')
  const candidate = normalize(target)
  return candidate !== '' && candidate !== normalize(current)
}
