/**
 * DNS rule actions understood by the editor across supported core versions.
 *
 * The generated inventory is intentionally pinned to the Go helper library
 * (currently sing-box 1.12). Newer installed binaries are validated by the
 * binary itself, so their additive actions live in this compatibility overlay
 * until the helper dependency is upgraded and the schema is regenerated.
 */
import { DNS_RULE_ACTION_INVENTORY as GENERATED } from './dnsRuleActionInventory.generated'
import type { OptionFieldInfo } from './optionSchema'

export const DNS_RULE_ACTION_INVENTORY = {
  ...GENERATED,
  evaluate: {
    server: { kind: 'string' },
    tag: { kind: 'string' },
    timeout: { kind: 'duration' },
  },
  respond: {
    match_response: { kind: 'string' },
    response_rcode: { kind: 'string' },
    race: { kind: 'boolean' },
  },
} as const satisfies Record<string, Record<string, OptionFieldInfo>>

export type DNSRuleActionTypeName = keyof typeof DNS_RULE_ACTION_INVENTORY

export type DNSRuleActionFieldKey<T extends DNSRuleActionTypeName> = Extract<
  keyof (typeof DNS_RULE_ACTION_INVENTORY)[T],
  string
>

export const DNS_RULE_ACTION_TYPE_NAMES = Object.keys(
  DNS_RULE_ACTION_INVENTORY,
) as DNSRuleActionTypeName[]
