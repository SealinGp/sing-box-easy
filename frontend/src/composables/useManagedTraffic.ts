import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import Select from '../volt/Select.vue'
import Popover from '../volt/Popover.vue'
import {
  trafficPolicyService as service,
  type ManagedSnapshot, type ManagedGroup, type ManagedPolicy,
  type ManagedCommand, type ManagedPlan,
} from '../services/trafficPolicy'
import { managedPolicyDraft } from '../utils/managedPolicyDraft'
import { matchValues } from '../lab/trafficPolicyConfig'

export function useManagedTraffic() {
  const { t } = useI18n()
  const route = useRoute()
  const groupsPage = computed(() => route.path.includes('/dns/'))
  const snapshot = ref<ManagedSnapshot>()
  const loading = ref(true),
    busy = ref(false),
    error = ref(''),
    feedback = ref('')
  const feedbackPolicyId = ref<string | null>(null)
  const policy = ref<ManagedPolicy | null>(null)
  const conditions = ref<{ kind: string; value: string }[]>([])
  const plan = ref<ManagedPlan | null>(null)
  const panelOpen = ref(false)
  const panel = ref<InstanceType<typeof Popover>>()
  const panelKind = ref<'group' | 'deleteGroup' | 'policyAction'>('group')
  const groupDraft = ref<ManagedGroup>()
  const targetGroup = ref<ManagedGroup>()
  const targetPolicy = ref<ManagedPolicy>()
  const replacement = ref('')
  const creatingReplacement = ref(false)
  const replacementSearch = ref('')
  const replacementSelect = ref<InstanceType<typeof Select>>()
  const quick = ref(false),
    dnsSearch = ref('')
  const dnsAnchor = ref<HTMLElement>()
  const dnsSelect = ref<InstanceType<typeof Select>>()
  const dnsChoice = computed({
    get: () =>
      policy.value?.group_id ||
      (policy.value?.dns_server
        ? 'server:' + policy.value.dns_server
        : 'unchanged'),
    set: (value: string) => {
      if (!policy.value) return
      policy.value.group_id =
        value === 'unchanged' || value.startsWith('server:') ? '' : value
      policy.value.dns_server = value.startsWith('server:') ? value.slice(7) : ''
      policy.value.timeout ??= 3
      policy.value.fallback ??= 'strict'
    },
  })
  const dependencies = computed(
    () =>
      snapshot.value?.policies.filter(
        (p) => p.group_id === targetGroup.value?.id
      ) ?? []
  )
  const missingServers = (g: ManagedGroup) =>
    g.servers.filter((tag) => !snapshot.value?.servers.includes(tag))
  const serialized = (v: unknown) => JSON.stringify(v, null, 2)
  function fail(e: unknown) {
    error.value = e instanceof Error ? e.message : String(e)
  }
  async function load() {
    try {
      snapshot.value = await service.list()
    } catch (e) {
      fail(e)
    } finally {
      loading.value = false
    }
  }
  let previewGeneration = 0
  function cancelPolicy() {
    policy.value = null
    clearPlan()
  }
  function previewGroup(group: ManagedGroup) {
    groupDraft.value = group
    void preview({ action: 'save_group', group })
  }
  function clearPlan() {
    previewGeneration++
    plan.value = null
    error.value = ''
  }
  watch([policy, conditions], clearPlan, { deep: true })
  function openPolicy(p?: ManagedPolicy) {
    if (busy.value) return
    dnsSearch.value = ''
    clearPlan()
    feedback.value = ''
    feedbackPolicyId.value = null
    policy.value = p
      ? managedPolicyDraft(p, snapshot.value!)
      : {
        id: '',
        name: '',
        group_id: '',
        outbound: snapshot.value?.outbounds[0] ?? '',
        mode: 'or',
        conditions: [],
        route_index: snapshot.value?.route_rules.length ?? 0,
        dns_index: snapshot.value?.dns_rules.length ?? 0,
      }
    conditions.value = p
      ? p.conditions.map((c) => ({ kind: c.kind, value: c.values.join('\n') }))
      : [{ kind: 'domain_suffix', value: '' }]
  }
  function openGroup(event: Event, g?: ManagedGroup, isQuick = false) {
    clearPlan()
    quick.value = isQuick
    groupDraft.value = g
      ? JSON.parse(JSON.stringify(g))
      : {
        id: '',
        name: isQuick ? dnsSearch.value.trim() : '',
        servers: [],
        timeout: 3,
        fallback: 'strict',
      }
    panelKind.value = 'group'
    panelOpen.value = true
    panel.value?.showAt(
      (isQuick ? dnsAnchor.value : event.currentTarget) as HTMLElement
    )
  }
  async function quickGroup(event: Event) {
    const name = dnsSearch.value.trim()
    await dnsSelect.value?.hideAndWait()
    openGroup(event, undefined, true)
    if (groupDraft.value) groupDraft.value.name = name
  }
  async function createReplacement() {
    const name = replacementSearch.value.trim()
    await replacementSelect.value?.hideAndWait()
    clearPlan()
    creatingReplacement.value = true
    quick.value = false
    groupDraft.value = { id: '', name, servers: [], timeout: 3, fallback: 'strict' }
    panelKind.value = 'group'
  }
  function cancelGroup() {
    if (creatingReplacement.value) {
      creatingReplacement.value = false
      panelKind.value = 'deleteGroup'
      clearPlan()
    } else {
      panel.value?.hide()
    }
  }
  function deleteGroup(event: Event, g: ManagedGroup) {
    creatingReplacement.value = false
    replacementSearch.value = ''

    clearPlan()
    quick.value = false
    targetGroup.value = g
    replacement.value = ''
    panelKind.value = 'deleteGroup'
    panelOpen.value = true
    panel.value?.showAt(event.currentTarget as HTMLElement)
  }
  function policyAction(event: Event, p: ManagedPolicy) {
    clearPlan()
    targetPolicy.value = p
    panelKind.value = 'policyAction'
    panelOpen.value = true
    panel.value?.showAt(event.currentTarget as HTMLElement)
  }
  async function preview(command: ManagedCommand) {
    busy.value = true
    clearPlan()
    try {
      const generation = previewGeneration
      const result = await service.plan(command)
      if (generation === previewGeneration) plan.value = result
    } catch (e) {
      fail(e)
    } finally {
      busy.value = false
    }
  }
  async function previewPolicy() {
    if (!policy.value) return
    await preview({
      action: 'save_policy',
      policy: {
        ...policy.value,
        route_index: Number(policy.value.route_index),
        dns_index: Number(policy.value.dns_index),
        timeout: Number(policy.value.timeout || 3),
        conditions: conditions.value.map((c) => ({
          kind: c.kind,
          values: matchValues(c.value),
        })),
      },
    })
  }
  async function commit() {
    if (!plan.value) return
    busy.value = true
    error.value = ''
    const command = plan.value.command
    const creatingPolicy = command.action === 'save_policy' && !policy.value?.id
    try {
      await service.commit(command)
      await load()
      if (command.action === 'save_group' && quick.value && policy.value) {
        policy.value.group_id = command.group!.id
        policy.value.dns_server = ''
      }
      if (command.action === 'save_group' && creatingReplacement.value) {
        replacement.value = command.group!.id
        creatingReplacement.value = false
        panelKind.value = 'deleteGroup'
        plan.value = null
        feedback.value = t('managed.saved')
        return
      }
      if (command.action === 'save_policy') policy.value = null
      plan.value = null
      panel.value?.hide()
      feedbackPolicyId.value = command.action === 'save_policy' && !creatingPolicy ? command.policy!.id : null
      feedback.value = t('managed.saved')
    } catch (e) {
      fail(e)
      plan.value = null
    } finally {
      busy.value = false
    }
  }
  function panelClosed() {
    panelOpen.value = false
    plan.value = null
    error.value = ''
    if (quick.value)
      dnsAnchor.value?.querySelector<HTMLElement>('[role="combobox"]')?.focus()
  }
  onMounted(load)

  return {
    groupsPage, snapshot, loading, busy, error, feedbackPolicyId,
    feedback, policy, conditions, plan, panelOpen,
    panel, panelKind, groupDraft, targetGroup, targetPolicy,
    replacement, replacementSearch, replacementSelect,
    dnsSearch, dnsAnchor, dnsSelect, dnsChoice, dependencies,
    missingServers, serialized, load, cancelPolicy, previewGroup,
    clearPlan, openPolicy, openGroup, quickGroup, createReplacement,
    cancelGroup, deleteGroup, policyAction, preview, previewPolicy,
    commit, panelClosed,
  }
}
