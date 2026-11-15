import { describe, expect, test } from 'bun:test'
import { Fragment, h } from 'vue'
import { elements, nodeText, containsCheckbox } from './tableVNodes'

describe('collection row structure', () => {
  test('keeps keyed loop rows and nested conditional cells in order', () => {
    const rows = elements([h(Fragment, [h('tr', { key: 'one' }, [
      h('td', 'Name'), h(Fragment, [h('td', 'UDP'), h('td', '53')]),
    ])])], 'tr')
    expect(rows.map(row => row.key)).toEqual(['one'])
    expect(elements(rows[0]!.children, 'td').map(cell => nodeText(cell.children))).toEqual(['Name', 'UDP', '53'])
  })
  test('ignores absent conditional rows and does not descend into nested detail tables', () => {
    expect(elements([null, false, h('div', [h('tr')]), h('tr', { key: 'detail' }, [h('td', { colspan: 5 }, 'Details')])], 'tr').map(row => row.key)).toEqual(['detail'])
  })
  test('recognizes business-owned selection even without a special header class', () => {
    expect(containsCheckbox([h('label', [h('input', { type: 'checkbox' })])])).toBe(true)
    expect(containsCheckbox([h('button', 'Sort')])).toBe(false)
  })
  test('derives translated custom header labels without markup', () => {
    expect(nodeText([h('button', ['服务器', h('span', ' ↑')])])).toBe('服务器 ↑')
  })
})
