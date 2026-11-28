import { describe, expect, test } from 'bun:test'
import { effectScope, nextTick, ref } from 'vue'
import { useHostsDraft, type HostsMap } from './useHostsDraft'
function fixture(initial: HostsMap = { 'keel.example.com': '192.0.2.2', 'git.example.com': ['192.0.2.3', '2001:db8::1'] }) {
  const scope = effectScope()
  const model = ref<HostsMap | undefined>(initial)
  let valid = true
  const draft = scope.run(() => useHostsDraft(() => model.value, value => { model.value = value }, value => { valid = value }))!
  return { ...draft, model, valid: () => valid, stop: () => scope.stop() }
}
describe('hosts input drafts', () => {
  test('add, clear, paste and parent echo preserve rows and identities', async () => {
    const f = fixture(); const originals = f.rows.value.map(row => row.id)
    f.add(); f.add()
    const added = f.rows.value[2]!.id; const other = f.rows.value[3]!.id
    expect(f.valid()).toBe(false)
    f.edit(added, { domain: 'new.example.com', addresses: '192.0.2.4' }); f.remove(other)
    await nextTick()
    expect(f.rows.value.map(row => row.id)).toEqual([...originals, added])
    f.edit(added, { domain: '' }); await nextTick()
    expect(f.rows.value).toHaveLength(3); expect(f.valid()).toBe(false)
    f.edit(added, { domain: ' pasted.example.com ' }); await nextTick()
    expect(f.rows.value.map(row => row.id)).toEqual([...originals, added])
    expect(f.rows.value[2]!.domain).toBe(' pasted.example.com ')
    expect(f.model.value?.['pasted.example.com']).toBe('192.0.2.4')
    expect(f.model.value?.['keel.example.com']).toBe('192.0.2.2'); f.stop()
  })
  test('duplicates stay editable without overwriting existing mappings', async () => {
    const f = fixture(); f.add(); const added = f.rows.value[2]!.id
    f.edit(added, { domain: 'KEEL.example.com.', addresses: '192.0.2.99' }); await nextTick()
    expect(f.rows.value).toHaveLength(3); expect(f.valid()).toBe(false)
    expect(f.model.value?.['keel.example.com']).toBe('192.0.2.2')
    f.edit(added, { domain: 'other.example.com' }); await nextTick()
    expect(f.valid()).toBe(true); expect(f.model.value?.['other.example.com']).toBe('192.0.2.99'); f.stop()
  })
  test('address drafts retain commas and external resets discard old drafts', async () => {
    const f = fixture(); const row = f.rows.value[0]!
    f.edit(row.id, { addresses: '192.0.2.2, ' }); await nextTick()
    expect(f.rows.value[0]!.addresses).toBe('192.0.2.2, ')
    f.edit(row.id, { addresses: '192.0.2.2, 2001:db8::2' }); await nextTick()
    expect(f.model.value?.[row.domain]).toEqual(['192.0.2.2', '2001:db8::2'])
    f.add(); f.model.value = { 'replacement.example.com': '192.0.2.8' }; await nextTick()
    expect(f.rows.value).toHaveLength(1); expect(f.valid()).toBe(true)
    f.remove(f.rows.value[0]!.id); await nextTick()
    expect(f.model.value).toBeUndefined(); f.stop()
  })
})
