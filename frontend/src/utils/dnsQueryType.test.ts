import { describe, expect, it } from 'bun:test'
import {
  COMMON_DNS_QUERY_TYPES,
  queryTypeOptions,
  readQueryTypes,
  writeQueryTypes,
} from './dnsQueryType'

describe('readQueryTypes', () => {
  it('returns [] for an absent field', () => {
    expect(readQueryTypes(undefined)).toEqual([])
    expect(readQueryTypes(null)).toEqual([])
  })

  it('wraps a scalar — badoption.Listable collapses a one-entry list', () => {
    expect(readQueryTypes('AAAA')).toEqual(['AAAA'])
    expect(readQueryTypes(65)).toEqual(['65'])
  })

  it('stringifies numbers so the select can bind them', () => {
    // sing-box writes a type it has no name for as a bare number.
    expect(readQueryTypes(['A', 32768])).toEqual(['A', '32768'])
  })
})

describe('writeQueryTypes', () => {
  it('sends numeric entries as JSON numbers', () => {
    // DNSQueryType.UnmarshalJSON tries a number first, then a NAME from
    // StringToType — the string "32768" is neither and fails decode.
    expect(writeQueryTypes(['A', '32768'])).toEqual(['A', 32768])
  })

  it('keeps names as names', () => {
    expect(writeQueryTypes(['HTTPS', 'AAAA'])).toEqual(['HTTPS', 'AAAA'])
  })

  it('drops blanks and duplicates', () => {
    expect(writeQueryTypes(['A', ' ', 'A'])).toEqual(['A'])
  })
})

describe('queryTypeOptions', () => {
  it('offers the common types', () => {
    const values = queryTypeOptions([]).map((o) => o.value)
    expect(values).toEqual([...COMMON_DNS_QUERY_TYPES])
  })

  it('keeps a selected value that is not in the list, so it stays visible', () => {
    // A MultiSelect renders nothing for a value it has no option for — the
    // type would vanish from the form while still living in the rule.
    const values = queryTypeOptions(['A', '32768']).map((o) => o.value)
    expect(values).toContain('32768')
    expect(values.filter((v) => v === 'A')).toHaveLength(1)
  })
})
