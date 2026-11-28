/** Detect a stale page before applying an index-based or whole-section edit. */
export function configSnapshot(message: () => string) {
  let baseline: string | undefined
  return {
    remember(value: unknown) { baseline = JSON.stringify(value) },
    assertCurrent(value: unknown) {
      if (baseline === undefined || baseline !== JSON.stringify(value)) throw new Error(message())
    },
  }
}
