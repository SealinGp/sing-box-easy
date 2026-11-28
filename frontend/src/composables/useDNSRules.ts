import { configSnapshot } from '../utils/configSnapshot'
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import type { DNSRule } from '../types/api'
import { configService, dnsService } from '../services'
import { useToast } from 'primevue'
import { useDragReorder } from '../composables/useDragReorder'
import { useDNSStore } from '../stores/dns'
import { useRouteStore } from '../stores/route'
import { storeToRefs } from 'pinia'
import { findOrphanRespondRules } from '../utils/parallelResolve'
import { readDNSRuleDraft, writeDNSRuleDraft, invalidDNSConditionGroup } from '../utils/dnsRuleDraft'
import {
  actionOf,
  applyActionDefaults,
  pruneForeignFields,
  resolveDNSRuleActionFields,
  validateDNSRuleAction,
  isTerminalAction,
  type DNSRuleActionTypeName,
} from '../schemas/dnsRuleActionFields'

import { dnsConditionKeys, dnsRuleOutcome } from '../utils/dnsRulePresentation'

export function useDNSRules() {
  const toast = useToast()
  const { t } = useI18n()
  const snapshot = configSnapshot(() => t('nav.stalePage'))
  const dnsStore = useDNSStore()
  const routeStore = useRouteStore()
  // The server list is no longer read here — the `dns-server` control in
  // SchemaFieldControl.vue owns that picker now. The store is still prefetched on
  // mount (below) so the select is populated before it is first opened.
  const { ruleSets } = storeToRefs(routeStore)

  // Local state for DNS rules
  const loading = ref(false)
  const dnsRules = ref<DNSRule[]>([])
  const coreCapabilities = ref({ dns_evaluate: false, dns_respond: false })

  // Modal state
  const showRuleModal = ref(false)
  // Bumped on every modal open; used as the conditions block's :key so it
  // remounts and re-derives which matcher style this rule uses.
  const conditionsKey = ref(0)
  const isEditMode = ref(false)
  const editingIndex = ref(-1)
  /**
   * The rule being edited, as a whole.
   *
   * This used to be a fixed 9-key object rebuilt from the loaded rule, with
   * everything else stashed in a `preservedFields` side-channel and re-merged on
   * save — but only while the action was unchanged. That allowlist caused three
   * separate data-loss bugs: `no_drop` was listed but never rendered, so it was
   * read into nothing and written back as nothing; `strategy` was neither listed
   * nor rendered, so a real rule in this repo's test config had
   * `"strategy": "ipv4_only"` erased just by opening it; and a logical rule lost
   * its `type`/`mode`/`rules` the moment its action changed.
   *
   * Now the whole rule is the model. `SchemaFieldsEditor` only touches keys it
   * renders, and `pruneForeignFields` removes only keys some OTHER action owns.
   * Anything neither knows about — every match condition, and any field a future
   * sing-box adds — survives by construction.
   */
  const currentRule = ref<Record<string, any>>(emptyRuleForm())

  /** The action driving the schema. Absent means route — see `actionOf`. */
  const currentAction = computed<DNSRuleActionTypeName>(() => actionOf(currentRule.value))

  /** The fields this action owns, resolved from the generated inventory. */
  const actionFields = computed(() => resolveDNSRuleActionFields(currentAction.value))

  /**
   * Everything on the rule that is NOT the action — i.e. its conditions.
   *
   * Derived by subtraction rather than from a list, because there is no generated
   * matcher inventory for DNS rules (the conditions half stayed hand-written).
   * Subtracting means a condition the form has no control for still SHOWS in the
   * preview, which is the honest behaviour: the rule is matching on it either way.
   */
  const conditionKeys = computed(() => dnsConditionKeys(currentRule.value))
  const flowOutcome = computed(() => dnsRuleOutcome(currentRule.value, t))
  const flowContinues = computed(() => !isTerminalAction(currentAction.value))

  /**
   * Switching the action prunes the previous one's fields and seeds the new one's
   * defaults — the analogue of `changeType()` in DNSServers.vue.
   *
   * There was no handler here at all before, which is why switching `route` to
   * `route-options` carried `server` along: the save-time `switch` deleted
   * method/no_drop/rcode but not server, and DNSRouteOptionsActionOptions has no
   * field for it, so the save failed on a field the form had stopped showing.
   */
  function changeAction(action: DNSRuleActionTypeName) {
    const pruned = pruneForeignFields(currentRule.value, action)
    // Defaults are for NEW records only. On an existing rule the operator is
    // changing the action deliberately, so the new action's core fields do need
    // seeding — but nothing already set is overwritten.
    currentRule.value = applyActionDefaults(pruned, action)
  }



  // Delete confirmation
  const showDeleteConfirm = ref(false)
  const deletingIndex = ref(-1)

  // Fetch DNS rules
  const fetchDNSRules = async () => {
    loading.value = true
    try {
      const { data } = await dnsService.getDNSRules()
      dnsRules.value = data.rules || []
      snapshot.remember(data.rules || [])
      reorder.syncKeys(dnsRules.value.length)
    } catch (err: any) {
      toast.add({
        severity: 'error',
        summary: t('common.error'),
        detail: err.message || t('dns.rules.toast.fetchFailed'),
        life: 3000
      })
    } finally {
      loading.value = false
    }
  }

  const actionTypes = computed(() => [
    { value: 'route', label: t('dns.rules.actionTypes.route') },
    { value: 'route-options', label: t('dns.rules.actionTypes.routeOptions') },
    { value: 'reject', label: t('dns.rules.actionTypes.reject') },
    // `predefined` is the only way to answer a query yourself in 1.12 — there is
    // no "block" DNS server type. Without it, blocking a domain at the DNS layer
    // is unreachable from this UI.
    { value: 'predefined', label: t('dns.rules.actionTypes.predefined') },
    ...(coreCapabilities.value.dns_evaluate || currentAction.value === 'evaluate'
      ? [{ value: 'evaluate', label: t('dns.rules.actionTypes.evaluate') }]
      : []),
    ...(coreCapabilities.value.dns_respond || currentAction.value === 'respond'
      ? [{ value: 'respond', label: t('dns.rules.actionTypes.respond') }]
      : []),
  ])

  async function fetchCoreCapabilities() {
    try {
      const { data } = await configService.getCoreInfo()
      coreCapabilities.value = data.capabilities
    } catch {
      // Older panel backends do not expose /core. Existing newer actions remain
      // editable through the currentAction exceptions above.
    }
  }

  // The rcode vocabulary, the reject methods and the DNS server picker all used to
  // be built here. They now live in schemas/dnsRuleActionFields.ts (the first two,
  // as curated `select` options) and in SchemaFieldControl.vue (the third, as the
  // `dns-server` control) — next to the field definitions they belong to, and
  // reachable from any future form that needs them.

  const ruleSetOptions = computed(() => {
    const options: { value: string; label: string }[] = []
    if (ruleSets.value) {
      ruleSets.value.forEach(ruleSet => {
        if (ruleSet.tag) {
          const type = (ruleSet as any).type || 'local'
          const format = (ruleSet as any).format || 'source'
          options.push({
            value: ruleSet.tag,
            label: t('dns.rules.ruleSetLabel', { tag: ruleSet.tag, type, format })
          })
        }
      })
    }
    return options
  })

  /**
   * A blank rule. Only the conditions are seeded, as empty arrays for the Chips
   * v-model; the action's own fields come from `applyActionDefaults`, so their
   * defaults live in the curation file rather than being restated here.
   */
  function emptyRuleForm(): Record<string, any> {
    return applyActionDefaults(
      {
        rule_set: [] as string[],
        domain: [] as string[],
        domain_suffix: [] as string[],
        domain_keyword: [] as string[],
        geosite: [] as string[],
        query_type: [] as string[],
      },
      'route',
    )
  }

  const openAddRuleModal = () => {
    isEditMode.value = false
    currentRule.value = emptyRuleForm()
    conditionsKey.value++
    showRuleModal.value = true
  }

  const openEditRuleModal = (index: number, rule: DNSRule) => {
    isEditMode.value = true
    editingIndex.value = index

    const raw = rule as Record<string, unknown>

    // The whole rule, not a fixed subset. Every key survives — including ones this
    // form has no control for, and `type`/`mode`/`rules` on a logical rule.
    //
    // Note what is NOT done here: no default is seeded. `applyActionDefaults` runs
    // on create and on an action change, never on open. Seeding one here is
    // exactly how a predefined rule with no `rcode` — which means NOERROR — used
    // to be silently rewritten to NXDOMAIN by opening it and pressing Update.
    currentRule.value = readDNSRuleDraft(raw)
    conditionsKey.value++
    showRuleModal.value = true
  }

  const closeRuleModal = () => {
    showRuleModal.value = false
    currentRule.value = emptyRuleForm()
  }

  const handleSaveRule = async () => {
    if (loading.value) return
    // Keep only the fields valid for the selected action. sing-box strict-parses
    // DNS rules and rejects any field that does not belong to the action —
    // `method` is reject-only ("unknown field method" on a route rule).
    //
    // This replaces a hand-written switch that had to be updated by hand for every
    // action, and which missed `server` on the route -> route-options edge.
    // pruneForeignFields derives the same answer from the generated inventory, and
    // — unlike the switch and the EDITED_FIELDS allowlist it worked with — keeps
    // every key no action owns: all the match conditions, and `type`/`mode`/`rules`
    // on a logical rule.
    const processedRule = writeDNSRuleDraft(pruneForeignFields(currentRule.value, currentAction.value))

    // Three constraints sing-box only enforces while decoding, which `sing-box
    // check` reports as an opaque upstream string. Caught here so the operator
    // sees which field is wrong. See validateDNSRuleAction for each one's source.
    const invalid = invalidDNSConditionGroup(processedRule) ? 'dns.rules.logical.empty' : validateDNSRuleAction(processedRule)
    if (invalid) {
      toast.add({
        severity: 'error',
        summary: t('common.error'),
        detail: t(invalid),
        life: 4000,
      })
      return
    }

    loading.value = true
    try {
      if (isEditMode.value) {
        snapshot.assertCurrent((await dnsService.getDNSRules()).data.rules || [])
        await dnsService.updateDNSRule(editingIndex.value, processedRule)
        toast.add({
          severity: 'success',
          summary: t('common.success'),
          detail: t('dns.rules.toast.updatedOk'),
          life: 3000
        })
      } else {
        await dnsService.addDNSRule(processedRule)
        toast.add({
          severity: 'success',
          summary: t('common.success'),
          detail: t('dns.rules.toast.addedOk'),
          life: 3000
        })
      }
      await fetchDNSRules()
      closeRuleModal()
    } catch (err: any) {
      toast.add({
        severity: 'error',
        summary: t('common.error'),
        detail: err.message || t('dns.rules.toast.saveFailed'),
        life: 3000
      })
    } finally {
      loading.value = false
    }
  }

  const openDeleteConfirm = (index: number) => {
    deletingIndex.value = index
    showDeleteConfirm.value = true
  }

  const closeDeleteConfirm = () => {
    showDeleteConfirm.value = false
    deletingIndex.value = -1
  }

  const handleDeleteRule = async () => {
    if (deletingIndex.value === -1) return
    loading.value = true
    try {
      snapshot.assertCurrent((await dnsService.getDNSRules()).data.rules || [])
      await dnsService.deleteDNSRule(deletingIndex.value)
      toast.add({
        severity: 'success',
        summary: t('common.success'),
        detail: t('dns.rules.toast.deletedOk'),
        life: 3000
      })
      await fetchDNSRules()
      closeDeleteConfirm()
    } catch (err: any) {
      toast.add({
        severity: 'error',
        summary: t('common.error'),
        detail: err.message || t('dns.rules.toast.deleteFailed'),
        life: 3000
      })
    } finally {
      loading.value = false
    }
  }

  /* ── Parallel resolution group ───────────────────────────────────────────────
   *
   * A separate flow because it writes SEVERAL rules that only work as a set. See
   * DNSParallelGroupDialog and utils/parallelResolve for why this is generated
   * rather than typed: sing-box fails a query outright when a `respond` is
   * reached with no preceding `evaluate` for its tag, so a group entered by hand
   * in the wrong order breaks resolution instead of degrading it.
   */
  const showParallelModal = ref(false)

  /**
   * Offered only when the running core has both actions.
   *
   * `evaluate` and `respond` arrived in sing-box 1.14. On an older core the
   * button would build a config that fails `sing-box check`, so it is absent
   * rather than disabled — there is nothing the operator could do about it here.
   */
  const canBuildParallelGroup = computed(
    () => coreCapabilities.value.dns_evaluate && coreCapabilities.value.dns_respond,
  )

  /**
   * Written in ONE request. Repeated single POSTs would pass through states that
   * are not partial versions of the target but broken configs.
   */
  async function handleCreateParallelGroup(rules: Record<string, any>[]) {
    loading.value = true
    try {
      const { data } = await dnsService.addDNSRulesBatch(rules)
      toast.add({
        severity: 'success',
        summary: t('common.success'),
        detail: t('dns.rules.parallel.toast.created', { count: data.added }),
        life: 3000,
      })
      await fetchDNSRules()
      showParallelModal.value = false
    } catch (err: any) {
      toast.add({
        severity: 'error',
        summary: t('common.error'),
        detail: err.message || t('dns.rules.toast.saveFailed'),
        life: 4000,
      })
    } finally {
      loading.value = false
    }
  }

  /**
   * Rules whose `respond` has no preceding `evaluate` to answer from.
   *
   * Shown DURING reorder, which is when it can actually help. sing-box rejects
   * this arrangement at `check` time, so a save cannot land one — but reorder is
   * a batch edit that sorts live and writes only on Save, so without this the
   * operator drags rules into a broken order, presses Save, and meets
   * "dns rule[31]: undefined evaluate tag: …" naming an index that has moved
   * since. The marker appears on the row the moment a drag separates a group.
   */
  const orphanRespondIndices = computed(() => new Set(findOrphanRespondRules(dnsRules.value as any)))

  /* ── Reorder ────────────────────────────────────────────────────────────────
   *
   * DNS rules match top-down exactly as route rules do, so their order is policy
   * too — a `geosite-cn -> local` rule under a catch-all never runs. Same
   * composable, same live-sorting drag; only the endpoint differs.
   */
  const reorder = useDragReorder(dnsRules, persistOrder)

  /**
   * Sends the permutation. The table already shows the result, so this only has
   * to handle the failure: refetch, which restores whatever the config actually
   * says rather than leaving the rows lying about it.
   */
  async function persistOrder(order: number[]) {
    try {
      snapshot.assertCurrent((await dnsService.getDNSRules()).data.rules || [])
      await dnsService.reorderDNSRules(order)
      snapshot.remember((await dnsService.getDNSRules()).data.rules || [])
      toast.add({
        severity: 'success',
        summary: t('common.success'),
        detail: t('dns.rules.toast.reordered'),
        life: 2000,
      })
    } catch (err: any) {
      toast.add({
        severity: 'error',
        summary: t('common.error'),
        detail: err.message || t('dns.rules.toast.reorderFailed'),
        life: 3000,
      })
      await fetchDNSRules()
      throw err
    }
  }

  // Load data on mount
  onMounted(() => {
    fetchCoreCapabilities()
    fetchDNSRules()
    routeStore.fetchRuleSets() // Fetch shared rule sets
    dnsStore.fetchDNSServers() // Fetch shared DNS servers
  })

  return { loading, dnsRules, showRuleModal, conditionsKey, isEditMode, currentRule, currentAction,
   actionFields, conditionKeys, flowOutcome, flowContinues, changeAction, actionTypes, ruleSetOptions,
   openAddRuleModal, openEditRuleModal, closeRuleModal, handleSaveRule, showDeleteConfirm,
   deletingIndex, openDeleteConfirm, closeDeleteConfirm, handleDeleteRule, showParallelModal,
   canBuildParallelGroup, handleCreateParallelGroup, orphanRespondIndices, reorder }
}
