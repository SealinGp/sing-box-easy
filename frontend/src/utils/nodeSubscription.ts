/** Ownership suffix minted by subscription imports; aligned with nodehistory.SubscriptionID. */
export function nodeSubscriptionId(tag: string): string {
  const start = tag.lastIndexOf(' | sub_')
  return start >= 0 && start + 7 < tag.length ? tag.slice(start + 3) : ''
}
