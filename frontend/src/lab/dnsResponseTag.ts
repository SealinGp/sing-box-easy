const alphabet = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_-'

/** A 60-bit random draft ID; the rule generator appends each resolver's index.
 * Backend integration must still check tags against the destination config.
 */
export function createDNSResponsePrefix(): string {
  const bytes = crypto.getRandomValues(new Uint8Array(10))
  return `dns_${Array.from(bytes, byte => alphabet[byte & 63]).join('')}`
}
