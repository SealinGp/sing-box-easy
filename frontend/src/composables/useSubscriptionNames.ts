import { onMounted, readonly, shallowRef } from 'vue'
import { subscriptionService } from '../services'

// One list read serves every visible node, including cards and nested panels.
const names = shallowRef<Record<string, string>>({})
let fetchedAt = 0
let pending: Promise<void> | null = null
async function load() {
  if (pending) return pending
  if (Date.now() - fetchedAt < 30_000) return
  pending = subscriptionService.getSubscriptions().then(({ data }) => {
    names.value = Object.fromEntries(data.subscriptions.map(sub => [sub.id, sub.name]))
    fetchedAt = Date.now()
  }).catch(() => {
    // Ownership IDs remain visible when names cannot be loaded.
  }).finally(() => { pending = null })
  return pending
}
export function useSubscriptionNames() {
  onMounted(load)
  return readonly(names)
}
