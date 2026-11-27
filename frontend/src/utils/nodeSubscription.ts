/** Ownership suffix minted by subscription imports; aligned with nodehistory.SubscriptionID. */
export function nodeSubscriptionId(tag: string): string {
  const start = tag.lastIndexOf(' | sub_')
  return start >= 0 && start + 7 < tag.length ? tag.slice(start + 3) : ''
}

/** Keep the node's full label and fingerprint, omitting only its ownership suffix. */
export function nodeNameWithoutSubscription(tag: string): string {
  const id = nodeSubscriptionId(tag)
  return id ? tag.slice(0, -(id.length + 3)) : tag
}
