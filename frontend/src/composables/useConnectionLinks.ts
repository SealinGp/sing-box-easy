import { computed, ref } from 'vue'
import { runtimeService } from '../services'
import { useRouteStore } from '../stores/route'

/**
 * What a connection's details may link to.
 *
 * A chain entry is only worth linking when it is a proxy GROUP (a plain node
 * or `direct` has no card on the Proxies page), and a rule-set name only when
 * the config defines it. Both lists are fetched once and shared: the details
 * panel is mounted per expanded row, and a request per row would be absurd.
 *
 * Failure is silent on purpose. Without the lists the names simply render as
 * the plain text they were before; nothing on the Connections page depends on
 * them.
 */
const groupNames = ref<ReadonlySet<string>>(new Set())
let groupsInFlight: Promise<void> | null = null

const loadGroups = (): Promise<void> => {
  if (!groupsInFlight) {
    groupsInFlight = runtimeService
      .getProxies()
      .then((response) => {
        groupNames.value = new Set(response.data.groups.map((group) => group.name))
      })
      .catch(() => undefined)
      .finally(() => {
        groupsInFlight = null
      })
  }
  return groupsInFlight
}

export function useConnectionLinks() {
  const routeStore = useRouteStore()
  const ruleSetTags = computed<ReadonlySet<string>>(
    () => new Set(routeStore.ruleSets.map((ruleSet) => ruleSet.tag)),
  )

  /** Refresh both lists. Called by the page, not by each details panel. */
  const refresh = () => {
    void loadGroups()
    void routeStore.ensureRuleSets()
  }

  const isGroup = (name: string) => groupNames.value.has(name)

  return { ruleSetTags, isGroup, refresh }
}
