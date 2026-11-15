/** Only one tooltip, including a pending hover, may own the screen. */
let active: { owner: object; dismiss: () => void } | undefined
export function claimTooltip(owner: object, dismiss: () => void) {
  if (active?.owner === owner) return
  const previous = active
  active = { owner, dismiss }
  previous?.dismiss()
}
export function releaseTooltip(owner: object) {
  if (active?.owner === owner) active = undefined
}
