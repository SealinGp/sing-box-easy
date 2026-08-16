import {
  applyTypeDefaults,
  pruneForType,
  USER_FIELDS,
  INBOUND_INVENTORY,
  type InboundTypeName,
} from '../schemas/inboundFields'
import { generateShadowsocksPassword } from './credentials'

/**
 * Save-time validation for an inbound.
 *
 * This is the ONLY place required-ness lives. The form's core/typical/advanced
 * tiers are about what to *show* — sing-box marks almost nothing Required (for
 * a `mixed` inbound, only `listen`), so tiering could never double as
 * validation. The rules here are the ones sing-box will actually reject, plus
 * the ones that produce a server nobody can connect to.
 *
 * Defaults and pruning now come from the schema (`schemas/inboundFields.ts`);
 * this module keeps only the checks and the shadowsocks password coupling,
 * which is a rule about two fields together rather than about either alone.
 */
type MutableInbound = Record<string, unknown>

export interface InboundValidationError {
  key: string
}

/** Types whose `users` list is what makes the inbound usable at all. */
const USERS_REQUIRED: InboundTypeName[] = ['vmess', 'vless', 'trojan', 'tuic', 'anytls']

function isInboundType(value: unknown): value is InboundTypeName {
  return typeof value === 'string' && value in INBOUND_INVENTORY
}

export const validateInboundRequiredFields = (
  inbound: MutableInbound,
): InboundValidationError | null => {
  const tag = inbound.tag
  if (typeof tag !== 'string' || !tag.trim()) {
    return { key: 'inbounds.validation.tagRequired' }
  }

  if (!isInboundType(inbound.type)) {
    return { key: 'inbounds.validation.typeRequired' }
  }

  const type = inbound.type
  const inventory = INBOUND_INVENTORY[type] as Record<string, unknown>

  // Driven by the generated inventory rather than a hardcoded `type !== 'tun'`:
  // tun genuinely has no listen_port field, and neither do any future types
  // that skip the shared listen options.
  if ('listen_port' in inventory && !inbound.listen_port) {
    return { key: 'inbounds.validation.listenPortRequired' }
  }

  if (type === 'shadowsocks') {
    if (!inbound.method) return { key: 'inbounds.validation.ssMethodRequired' }
    if (inbound.method !== 'none' && !inbound.password) {
      return { key: 'inbounds.validation.ssPasswordRequired' }
    }
  }

  if (USERS_REQUIRED.includes(type)) {
    const users = inbound.users
    if (!Array.isArray(users) || users.length === 0) {
      return { key: 'inbounds.validation.usersRequired' }
    }

    // The identity field is what a client authenticates with — a user row with
    // an empty uuid/password is accepted by the config parser and then rejects
    // every connection, which is far harder to diagnose than a save error.
    const identity = USER_FIELDS[type]?.find((field) => field.identity)
    if (identity) {
      const incomplete = users.some((user) => {
        const value = (user as Record<string, unknown>)[identity.key]
        return value === undefined || value === null || value === ''
      })
      if (incomplete) return { key: 'inbounds.validation.userIdentityRequired' }
    }
  }

  return null
}

/**
 * Prepare a NEW inbound after a type change.
 *
 * Two things happen, and the order matters: prune first so the previous type's
 * fields are gone, then seed defaults for the new one.
 *
 * Pruning fixes a real bug. The old watcher applied defaults but never removed
 * anything, so building a shadowsocks inbound and switching to trojan carried
 * `method` and `password` into the payload. sing-box decodes inbound options
 * strictly, so the save failed naming a field the form no longer displayed.
 */
export const prepareInboundForType = (
  inbound: MutableInbound,
  type: InboundTypeName,
): MutableInbound => {
  const pruned = pruneForType(inbound, type)
  const seeded = applyTypeDefaults(pruned, type)

  // Shadowsocks couples password format to the chosen method, so a generated
  // password has to follow the method rather than be seeded blindly as a
  // schema default.
  if (type === 'shadowsocks') {
    const method = seeded.method as string | undefined
    if (method === 'none') {
      delete seeded.password
    } else if (!seeded.password) {
      seeded.password = generateShadowsocksPassword(method)
    }
  }

  return seeded
}

/**
 * Prepare an EXISTING inbound for editing.
 *
 * Deliberately does not seed defaults. The previous implementation called the
 * same defaults path on edit-open, which injected a `users` array into a config
 * that had legitimately omitted one — mutating the operator's config before
 * they touched anything, and making the diff on save look like their doing.
 * Fields the config does not set stay unset; the form still renders any field
 * that HAS a value, so nothing is hidden.
 */
export const prepareInboundForEdit = (inbound: MutableInbound): MutableInbound => ({ ...inbound })

export { generateVmessUUID, generateShadowsocksPassword } from './credentials'
