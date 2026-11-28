import { computed, ref, watch } from 'vue'

export type HostsMap = Record<string, string | string[]>
export interface HostRow { id: number; domain: string; addresses: string }

/** Input rows retain their identity and unfinished text across v-model echoes. */
export function useHostsDraft(
  value: () => HostsMap | undefined,
  update: (value: HostsMap | undefined) => void,
  validity: (valid: boolean) => void,
) {
  let receiving = false
  let nextId = 0
  let published = JSON.stringify(value())
  const load = (map?: HostsMap): HostRow[] => Object.entries(map ?? {}).map(([domain, addresses]) => ({
    id: nextId++, domain, addresses: (Array.isArray(addresses) ? addresses : [addresses]).join(', '),
  }))
  const rows = ref<HostRow[]>(load(value()))
  const errors = computed(() => {
    const names = rows.value.map(row => row.domain.trim().toLowerCase().replace(/\.$/, ''))
    return names.map(name => !name ? 'hostsDomainRequired' : names.filter(other => other === name).length > 1 ? 'hostsDomainDuplicate' : '')
  })
  watch(value, incoming => {
    if (JSON.stringify(incoming) === published) return
    published = JSON.stringify(incoming)
    receiving = true
    rows.value = load(incoming)
    receiving = false
  }, { deep: true })
  validity(errors.value.every(error => !error))
  watch(rows, () => {
    const valid = errors.value.every(error => !error)
    validity(valid)
    if (!valid || receiving) return
    const entries = rows.value.map(row => {
      const addresses = row.addresses.split(',').map(part => part.trim()).filter(Boolean)
      return [row.domain.trim(), addresses.length === 1 ? addresses[0]! : addresses] as const
    })
    const next = entries.length ? Object.fromEntries(entries) : undefined
    const signature = JSON.stringify(next)
    if (signature === published) return
    published = signature
    update(next)
  }, { deep: true, flush: 'sync' })
  function edit(id: number, patch: Partial<Pick<HostRow, 'domain' | 'addresses'>>) {
    rows.value = rows.value.map(row => row.id === id ? { ...row, ...patch } : row)
  }
  function add() { rows.value = [...rows.value, { id: nextId++, domain: '', addresses: '' }] }
  function remove(id: number) { rows.value = rows.value.filter(row => row.id !== id) }
  return { rows, errors, edit, add, remove }
}
