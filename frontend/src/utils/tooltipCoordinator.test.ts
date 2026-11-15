import { expect, test } from 'bun:test'
import { claimTooltip, releaseTooltip } from './tooltipCoordinator'

test('a new tooltip cancels the previous owner, including a pending hover', () => {
  const first = {}, second = {}
  let dismissed = 0
  claimTooltip(first, () => { dismissed++; releaseTooltip(first) })
  claimTooltip(second, () => { dismissed++ })
  expect(dismissed).toBe(1)
  claimTooltip(second, () => { dismissed++ })
  expect(dismissed).toBe(1)
  releaseTooltip(first)
  claimTooltip(first, () => {})
  expect(dismissed).toBe(2)
  releaseTooltip(first)
})
