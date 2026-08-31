import { describe, expect, it } from 'bun:test'
import { apiErrorMessage } from './apiErrorMessage'
import { ApiError } from '../types/api'

describe('apiErrorMessage', () => {
  it('prefers the backend message', () => {
    expect(apiErrorMessage(new ApiError(3, 'subscription server returned http 404'), 'fallback')).toBe(
      'subscription server returned http 404',
    )
    expect(apiErrorMessage(new Error('boom'), 'fallback')).toBe('boom')
  })

  it('falls back when there is nothing useful to show', () => {
    expect(apiErrorMessage(new Error('   '), 'fallback')).toBe('fallback')
    expect(apiErrorMessage(undefined, 'fallback')).toBe('fallback')
    expect(apiErrorMessage('a string, not an Error', 'fallback')).toBe('fallback')
    expect(apiErrorMessage({ msg: 'not an Error either' }, 'fallback')).toBe('fallback')
  })
})
