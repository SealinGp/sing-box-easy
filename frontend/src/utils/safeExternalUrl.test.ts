import { describe, expect, it } from 'bun:test'
import { displayHost, isLinkableSiteInput, safeExternalUrl } from './safeExternalUrl'

describe('safeExternalUrl', () => {
  it('keeps http and https links', () => {
    expect(safeExternalUrl('https://example.com')).toBe('https://example.com/')
    expect(safeExternalUrl('http://example.com/buy')).toBe('http://example.com/buy')
    expect(safeExternalUrl('  https://example.com/#/register  ')).toBe(
      'https://example.com/#/register',
    )
  })

  it('rejects anything that could execute on click', () => {
    // The value is provider-controlled and lands in an href, so these are the
    // cases that matter more than any formatting nicety.
    for (const hostile of [
      'javascript:alert(1)',
      'JavaScript:alert(1)',
      '  javascript:alert(1)',
      'java\tscript:alert(1)',
      'data:text/html,<script>alert(1)</script>',
      'vbscript:msgbox(1)',
      'file:///etc/passwd',
      'tg://resolve?domain=x',
    ]) {
      expect(safeExternalUrl(hostile)).toBeNull()
    }
  })

  it('rejects empty and unparseable values', () => {
    expect(safeExternalUrl('')).toBeNull()
    expect(safeExternalUrl('   ')).toBeNull()
    expect(safeExternalUrl(undefined)).toBeNull()
    expect(safeExternalUrl(null)).toBeNull()
    // No scheme: the backend promotes a bare domain when it stores one, so a
    // value arriving here without a scheme is not a link this can vouch for.
    expect(safeExternalUrl('example.com')).toBeNull()
  })
})

describe('displayHost', () => {
  it('reduces a link to its host', () => {
    expect(displayHost('https://www.example.com/user/login')).toBe('www.example.com')
    expect(displayHost('http://example.com:8080/x')).toBe('example.com:8080')
  })

  it('is empty for anything unsafe', () => {
    expect(displayHost('javascript:alert(1)')).toBe('')
    expect(displayHost('')).toBe('')
  })
})

describe('isLinkableSiteInput', () => {
  it('accepts what the backend stores, including a bare domain', () => {
    // The backend promotes these to https, so the form must not refuse them.
    for (const input of [
      'https://example.com',
      'http://example.com/buy',
      'example.com',
      'www.example.com',
      'example.com/buy',
      'example.com:8443/buy',
      '  example.com  ',
    ]) {
      expect(isLinkableSiteInput(input)).toBe(true)
    }
  })

  it('still refuses anything that could execute, and non-sites', () => {
    for (const input of [
      'javascript:alert(1)',
      'JavaScript:alert(1)',
      'data:text/html,<script>alert(1)</script>',
      'vbscript:msgbox(1)',
      'file:///etc/passwd',
      // Degrades to a hostname with no dot once https:// is prepended.
      'javascript:8080',
      'localhost',
      '',
      '   ',
    ]) {
      expect(isLinkableSiteInput(input)).toBe(false)
    }
  })
})
