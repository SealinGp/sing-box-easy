import { readonly, ref } from 'vue'
import { systemService } from '../services'

/**
 * The version of the sing-box binary actually installed on this host.
 *
 * The schema forms need this because the generated inventories describe the
 * sing-box LIBRARY this repo pins, not the binary that will parse the config.
 * The two diverge in practice — the machine this was written on pins 1.12.12
 * and runs 1.13.11 — and the divergence is only dangerous in one direction:
 * offering a field or type the installed binary has removed. See
 * `isRetired` in schemas/optionSchema.ts.
 *
 * Module-level state, fetched once per session and shared by every form. The
 * value changes only when the operator installs a different sing-box, which
 * reloads the panel anyway.
 */
const version = ref<string | undefined>(undefined)
const loaded = ref(false)
let inflight: Promise<void> | null = null

/**
 * Fetch once, collapsing concurrent callers.
 *
 * A failure is swallowed on purpose. Version gating is a refinement: with no
 * version we show everything, which is exactly the behaviour before this
 * existed. Blocking a form because a metadata endpoint was slow would be a
 * worse trade.
 */
function ensureVersion(): Promise<void> {
  if (loaded.value) return Promise.resolve()
  if (inflight) return inflight

  const request = systemService
    .getInfo()
    .then((info) => {
      const reported = info?.sing_box_version
      // The backend degrades every field to a zero value rather than failing,
      // so "unknown" and "" both mean "could not detect".
      version.value = reported && reported !== 'unknown' ? reported : undefined
    })
    .catch(() => {
      version.value = undefined
    })
    .finally(() => {
      loaded.value = true
      inflight = null
    })

  inflight = request
  return request
}

export function useSingBoxVersion() {
  void ensureVersion()
  return {
    /** e.g. "1.13.11", or undefined when detection failed. */
    singBoxVersion: readonly(version),
    loaded: readonly(loaded),
    ensureVersion,
  }
}
