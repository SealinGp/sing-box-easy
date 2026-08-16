/**
 * sing-box enum vocabularies shared by more than one schema domain.
 *
 * These are DATA, not machinery, so they do not belong in `optionSchema.ts` —
 * but they are not one domain's editorial choice either, so they do not belong
 * in a single `*Fields.ts`. `network_type` is the clearest case: it is a dialer
 * option on outbounds and DNS servers, a matcher on route rules, and a field of
 * the `direct` route action, so four curation files need the same list.
 *
 * Every list here is transcribed from a sing-box `MarshalJSON` or its
 * `map[T]string`, with the source noted. Recheck on a dependency bump — these
 * are the one thing in the schema layer the generator cannot supply, because
 * reflection sees the Go type and not the values it accepts.
 */

/**
 * `option.NetworkStrategy` — constant/network.go:42-48.
 *
 * Shipped as `kind: 'number'` until the generator learned this type, because the
 * underlying Go type is a uint8. The form rendered a number spinner and every
 * value an operator typed was rejected.
 */
export const NETWORK_STRATEGIES = ['default', 'fallback', 'hybrid'] as const

/** `option.InterfaceType` — constant/network.go:18-23. Same uint8 story. */
export const INTERFACE_TYPES = ['wifi', 'cellular', 'ethernet', 'other'] as const

/**
 * `option.DomainStrategy` — option/types.go:66-84.
 *
 * `as_is` is accepted on read but marshals back as "", so it is deliberately
 * absent: removing the field is the same thing and does not leave a value that
 * changes shape on save.
 */
export const DOMAIN_STRATEGIES = ['prefer_ipv4', 'prefer_ipv6', 'ipv4_only', 'ipv6_only'] as const
