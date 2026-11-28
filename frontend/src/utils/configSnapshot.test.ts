import { test, expect } from 'bun:test'
import { configSnapshot } from './configSnapshot'
test('blocks editing an index when another page inserted or reordered rules', () => {
  const guard = configSnapshot(() => 'changed')
  const rules = [{ domain: ['one.test'] }, { domain: ['two.test'] }]
  guard.remember(rules)
  expect(() => guard.assertCurrent(JSON.parse(JSON.stringify(rules)))).not.toThrow()
  expect(() => guard.assertCurrent([rules[1], rules[0]])).toThrow('changed')
  expect(() => guard.assertCurrent([{ action: 'reject' }, ...rules])).toThrow('changed')
  rules[0]!.domain = ['edited.test']
  expect(() => guard.assertCurrent(rules)).toThrow('changed')
})
