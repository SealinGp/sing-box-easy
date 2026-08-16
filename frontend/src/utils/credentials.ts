/**
 * Credential generation for inbound users and passwords.
 *
 * Pulled out of Inbounds.vue, where `generatePassword` and `generateVMessUUID`
 * were defined inside the view and therefore reachable only from that one
 * modal. The users editor needs both, for every inbound type that takes
 * credentials rather than just the two the old template happened to cover.
 *
 * Everything here uses `crypto.getRandomValues`. `Math.random` is not a CSPRNG
 * and these values are the only thing standing between a proxy and the open
 * internet.
 */

const UUID_TEMPLATE = '10000000-1000-4000-8000-100000000000'
const SECRET_ALPHABET = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
const DEFAULT_SECRET_LENGTH = 32

function randomBytes(length: number): Uint8Array {
  const bytes = new Uint8Array(length)
  const source = globalThis.crypto
  if (!source?.getRandomValues) {
    // Refuse rather than silently degrading to Math.random: a password that
    // looks random but is predictable is worse than a visible failure.
    throw new Error('Secure random source unavailable; cannot generate credentials')
  }
  source.getRandomValues(bytes)
  return bytes
}

/** RFC 4122 v4 UUID — the identity format VMess, VLESS and TUIC users use. */
export function generateVmessUUID(): string {
  const source = globalThis.crypto
  if (source && typeof source.randomUUID === 'function') {
    return source.randomUUID()
  }

  // randomUUID needs a secure context; a panel served over plain HTTP on a LAN
  // does not have one, so fall back to a hand-assembled v4 from getRandomValues.
  return UUID_TEMPLATE.replace(/[018]/g, (c) => {
    const digit = Number(c)
    const rand = randomBytes(1)[0] as number
    return (digit ^ (rand & (15 >> (digit / 4)))).toString(16)
  })
}

/** Opaque random string, for password-style credentials with no format rules. */
export function generateSecret(length = DEFAULT_SECRET_LENGTH): string {
  const bytes = randomBytes(length)
  let out = ''
  for (let i = 0; i < bytes.length; i++) {
    out += SECRET_ALPHABET.charAt((bytes[i] as number) % SECRET_ALPHABET.length)
  }
  return out
}

/**
 * Shadowsocks passwords are method-dependent, and getting it wrong is a startup
 * failure rather than a weak key:
 *
 *   - the 2022-blake3-* ciphers take a base64-encoded key of an EXACT byte
 *     length — 16 for the -128- variant, 32 for the others — and sing-box
 *     rejects anything else outright;
 *   - `none` takes no password at all;
 *   - the legacy AEAD ciphers accept any string.
 */
export function generateShadowsocksPassword(method: string | undefined): string {
  if (method === 'none') return ''

  if (method?.startsWith('2022-blake3-')) {
    const keyLength = method.includes('128') ? 16 : 32
    const bytes = randomBytes(keyLength)
    let binary = ''
    for (let i = 0; i < bytes.length; i++) {
      binary += String.fromCharCode(bytes[i] as number)
    }
    // Bare `btoa`, not `window.btoa`: the latter is undefined anywhere without
    // a window object, which is how this first showed up under the test runner.
    return btoa(binary)
  }

  return generateSecret()
}
