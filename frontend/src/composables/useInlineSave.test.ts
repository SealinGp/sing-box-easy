import { describe, expect, test } from 'bun:test'
import { useInlineSave } from './useInlineSave'

describe('inline persistence', () => {
  test('locks duplicate requests, retains a recoverable error, and allows retry', async () => {
    const state = useInlineSave()
    let reject!: (error: Error) => void
    let calls = 0
    const pending = state.persist(() => {
      calls++
      return new Promise((_, fail) => { reject = fail })
    }, 'saved', 'failed')
    expect(state.saving.value).toBe(true)
    expect(await state.persist(async () => { calls++ }, 'saved', 'failed')).toBe(false)
    expect(calls).toBe(1)
    reject(new Error('network unavailable'))
    expect(await pending).toBe(false)
    expect(state.saving.value).toBe(false)
    expect(state.saveError.value).toBe('network unavailable')
    expect(await state.persist(async () => {}, 'saved', 'failed')).toBe(true)
    expect(state.saveError.value).toBe('')
    expect(state.savedMessage.value).toBe('saved')
    state.clearSaveFeedback()
    expect(state.savedMessage.value).toBe('')
  })
})
