import { INBOUND_INVENTORY } from '../schemas/inboundInventory.generated'
import { isRetired, type OptionFieldInfo } from '../schemas/optionSchema'
import type { Inbound } from '../types/api'

/** Retain old-core options, but omit fields the detected binary has removed. */
export function compatibleInbound(config: Partial<Inbound>, version: string | undefined): Inbound {
  const fields: Record<string, OptionFieldInfo> = INBOUND_INVENTORY[config.type as keyof typeof INBOUND_INVENTORY] ?? {}
  return Object.fromEntries(Object.entries(config).filter(([key]) => {
    const field = fields[key]
    return !field || !isRetired(field, version)
  })) as Inbound
}
