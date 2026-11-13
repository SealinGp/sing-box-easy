<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import {
  Button,
  Card,
  Input,
  Textarea,
  Skeleton,
  IconButton,
  StatusIndicator,
} from '../../components'
import Select from '../../volt/Select.vue'
import Popover from '../../volt/Popover.vue'
import DNSGroupForm from '../../lab/DNSGroupForm.vue'
import {
  trafficPolicyService as service,
  type ManagedSnapshot,
  type ManagedGroup,
  type ManagedPolicy,
  type ManagedCommand,
  type ManagedPlan,
} from '../../services/trafficPolicy'
import { matchValues } from '../../lab/trafficPolicyConfig'
const { t } = useI18n()
const route = useRoute()
const groupsPage = computed(() => route.path.includes('/dns/'))
const snapshot = ref<ManagedSnapshot>()
const loading = ref(true),
  busy = ref(false),
  error = ref(''),
  feedback = ref('')
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
const groupOptions = computed(() => [
  { id: 'unchanged', name: t('lab.leaveDnsUnchanged') },
  ...(snapshot.value?.groups ?? []),
  ...(snapshot.value?.servers ?? []).map((tag) => ({
    id: 'server:' + tag,
    name: tag,
  })),
])
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
const kinds = computed(() =>
  ['domain_suffix', 'domain', 'rule_set', 'ip_cidr'].map((value, i) => ({
    value,
    label: t(
      ['lab.domainSuffix', 'lab.exactDomain', 'lab.ruleSet', 'lab.ipRange'][i]!
    ),
  }))
)
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
function cancelPolicy(){policy.value=null;clearPlan()}
function previewGroup(group:ManagedGroup){groupDraft.value=group;void preview({action:'save_group',group})}
function clearPlan() {
  previewGeneration++
  plan.value = null
  error.value = ''
}
watch([policy, conditions], clearPlan, { deep: true })
function openPolicy(p?: ManagedPolicy) {
  dnsSearch.value = ''
  clearPlan()
  feedback.value = ''
  policy.value = p
    ? JSON.parse(JSON.stringify(p))
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
function dnsSearchKeydown(event: KeyboardEvent) {
  if (event.key === 'Tab' && !event.shiftKey) {
    event.preventDefault()
    event.stopPropagation()
    document.getElementById('managed-dns-create')?.focus()
  }
}
function dnsCreateKeydown(event: KeyboardEvent) {
  if (event.key === 'Tab' && event.shiftKey) {
    event.preventDefault()
    document.getElementById('managed-dns-search')?.focus()
  } else if (event.key === 'Escape') {
    event.preventDefault()
    event.stopPropagation()
    dnsSelect.value?.hide()
    dnsAnchor.value?.querySelector<HTMLElement>('[role="combobox"]')?.focus()
  }
}
async function quickGroup(event: Event) {
  const name=dnsSearch.value.trim()
  await dnsSelect.value?.hideAndWait()
  openGroup(event, undefined, true)
  if(groupDraft.value)groupDraft.value.name=name
}
async function createReplacement() {
 const name=replacementSearch.value.trim()
 await replacementSelect.value?.hideAndWait()
 clearPlan();creatingReplacement.value=true;quick.value=false
 groupDraft.value={id:'',name,servers:[],timeout:3,fallback:'strict'}
 panelKind.value='group'
}
function cancelGroup() {
 if(creatingReplacement.value){creatingReplacement.value=false;panelKind.value='deleteGroup';clearPlan()}
 else panel.value?.hide()
}
function deleteGroup(event: Event, g: ManagedGroup) {
 creatingReplacement.value=false
 replacementSearch.value=''

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
  try {
    await service.commit(command)
    await load()
    if (command.action === 'save_group' && quick.value && policy.value) {
      policy.value.group_id = command.group!.id
      policy.value.dns_server = ''
    }
    if (command.action === 'save_group' && creatingReplacement.value) {
 replacement.value=command.group!.id;creatingReplacement.value=false;panelKind.value='deleteGroup';plan.value=null;feedback.value=t('managed.saved');return
 }
 if (command.action === 'save_policy') policy.value = null
    plan.value = null
    panel.value?.hide()
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
</script>
<template>
  <section class="space-y-5">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="text-xl font-semibold">
          {{ t(groupsPage ? 'lab.resolverGroups' : 'lab.trafficPolicies') }}
        </h1>
        <p class="mt-2 text-sm text-text-secondary">
          {{ t('managed.description') }}
        </p>
      </div>
      <div class="flex gap-2">
        <Button variant="ghost" :disabled="busy" @click="load">{{
          t('managed.refresh')
        }}</Button
        ><Button v-if="groupsPage" @click="openGroup($event)">{{
          t('lab.createDnsGroup')
        }}</Button
        ><Button v-else @click="openPolicy()">{{
          t('lab.addTrafficPolicy')
        }}</Button>
      </div>
    </header>
    <p v-if="feedback" role="status" class="text-sm text-success">
      {{ feedback }}
    </p>
    <p v-if="error" role="alert" class="text-sm text-danger">{{ error }}</p>
    <div v-if="loading" class="space-y-3" aria-busy="true">
      <Skeleton v-for="n in 3" :key="n" class="h-24 w-full" />
    </div>
    <template v-else-if="snapshot">
      <Card v-if="snapshot.recovery"
        ><h2 class="font-semibold">{{ t('managed.recovery') }}</h2>
        <p class="my-3 text-sm">{{ t('managed.recoveryHelp') }}</p>
        <div class="flex flex-wrap gap-2">
          <Button
            :disabled="busy"
            @click="preview({ action: 'recover_accept' })"
            >{{ t('managed.recoverAccept') }}</Button
          ><Button
            variant="secondary"
            :disabled="busy"
            @click="preview({ action: 'recover_discard' })"
            >{{ t('managed.recoverDiscard') }}</Button
          >
        </div></Card
      >
      <template v-if="groupsPage">
        <Card v-if="!snapshot.groups.length"
          ><p>{{ t('managed.noGroups') }}</p></Card
        >
        <Card v-for="g in snapshot.groups" :key="g.id"
          ><div class="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h2 class="font-semibold">{{ g.name }}</h2>
              <p class="mt-2 text-sm text-text-secondary">
                {{ g.servers.join(' + ') }} ·
                {{ t('lab.seconds', { n: g.timeout }) }}
              </p>
              <p class="mt-2 text-sm">
                {{
                  t('managed.usedBy', {
                    n: snapshot.policies.filter((p) => p.group_id === g.id)
                      .length,
                  })
                }}
              </p>
              <p
                v-if="missingServers(g).length"
                role="status"
                class="mt-2 text-sm text-danger"
              >
                {{
                  t('managed.missingServers', {
                    names: missingServers(g).join(', '),
                  })
                }}
              </p>
            </div>
            <div class="flex gap-1">
              <IconButton
                action="edit"
                :label="t('lab.editNamed', { name: g.name })"
                @click="openGroup($event, g)"
              /><IconButton
                action="delete"
                :label="t('managed.deleteNamed', { name: g.name })"
                @click="deleteGroup($event, g)"
              />
            </div></div
        ></Card>
        <RouterLink
          class="inline-block text-sm text-primary underline"
          to="/dashboard/route/policies"
          >{{ t('lab.trafficPolicies') }} →</RouterLink
        >
      </template>
      <template v-else>
        <Card v-if="policy" class="scroll-mt-4">
          <form class="space-y-5" :inert="busy" @submit.prevent="previewPolicy">
            <div class="grid gap-5 md:grid-cols-2">
              <div class="space-y-4">
                <Input
                  v-model="policy.name"
                  :label="t('managed.policyName')"
                  required
                />
                <Select
                  v-model="policy.mode"
                  :aria-label="t('lab.conditionLogic')"
                  :options="[
                    { label: t('lab.matchAnyRowOr'), value: 'or' },
                    { label: t('lab.matchAllRowsAnd'), value: 'and' },
                  ]"
                  option-label="label"
                  option-value="value"
                  class="w-full"
                />
                <div
                  v-for="(c, i) in conditions"
                  :key="i"
                  class="space-y-2 rounded-control border border-border p-3"
                >
                  <div class="flex gap-2">
                    <Select
                      v-model="c.kind"
                      :options="kinds"
                      option-label="label"
                      option-value="value"
                      :aria-label="t('lab.numberedCondition', { n: i + 1 })"
                      class="min-w-0 flex-1"
                    /><IconButton
                      v-if="conditions.length > 1"
                      action="delete"
                      :label="t('lab.removeCondition', { n: i + 1 })"
                      @click="conditions.splice(i, 1)"
                    />
                  </div>
                  <Textarea
                    v-model="c.value"
                    :label="t('lab.matchValues', { n: i + 1 })"
                    :rows="3"
                    :hint="
                      t('lab.oneValuePerLineOrSeparateWithCommasMatchesAny')
                    "
                  />
                </div>
                <Button
                  variant="ghost"
                  @click="conditions.push({ kind: 'domain_suffix', value: '' })"
                  >{{ t('managed.addCondition') }}</Button
                >
              </div>
              <div class="space-y-4">
                <label class="block text-sm font-medium"
                  >{{ t('lab.outbound')
                  }}<Select
                    v-model="policy.outbound"
                    :options="snapshot.outbounds"
                    filter
                    :aria-label="t('lab.outbound')"
                    class="mt-2 w-full"
                /></label>
                <div ref="dnsAnchor">
                  <label class="mb-2 block text-sm font-medium">{{
                    t('managed.resolveWith')
                  }}</label
                  ><Select
                    ref="dnsSelect"
                    v-model="dnsChoice"
                    filter-input-id="managed-dns-search"
                    :filter-input-label="t('managed.searchGroups')"
                    :filter-keydown="dnsSearchKeydown"
                    :options="groupOptions"
                    option-label="name"
                    option-value="id"
                    filter
                    :aria-label="t('managed.resolveWith')"
                    :filter-placeholder="t('managed.searchGroups')"
                    class="w-full"
                    @filter="dnsSearch = $event.value"
 @hide="dnsSearch = ''"
                    ><template #footer
                      ><button
                        type="button"
                        id="managed-dns-create"
                        @keydown="dnsCreateKeydown"
                        class="w-full border-t border-border p-3 text-left text-sm text-primary"
                        @click="quickGroup"
                      >
                        +
                        {{
                          dnsSearch.trim()
                            ? t('lab.createNamedGroup', {
                                name: dnsSearch.trim(),
                              })
                            : t('lab.createDnsGroup')
                        }}
                      </button></template
                    ></Select
                  ><button
                    type="button"
                    class="mt-2 text-sm text-primary underline"
                    @click="quickGroup"
                  >
                    {{ t('lab.createDnsGroup') }}
                  </button>
                </div>
                <div v-if="policy.dns_server" class="space-y-3">
                  <Input
                    v-model="policy.timeout"
                    type="number"
                    :min="1"
                    :max="30"
                    :label="t('lab.timeoutSeconds')"
                  /><Select
                    v-model="policy.fallback"
                    :aria-label="t('lab.ifResolutionFails')"
                    :options="[
                      {
                        label: t('lab.stopWithinThisProfile'),
                        value: 'strict',
                      },
                      {
                        label: t('lab.allowExistingDnsFallback'),
                        value: 'general',
                      },
                    ]"
                    option-label="label"
                    option-value="value"
                    class="w-full"
                  />
                </div>
                <p class="text-xs text-text-secondary">
                  {{ t('managed.priority') }}
                </p>
                <Input
                  v-model="policy.route_index"
                  type="number"
                  :min="0"
                  :max="snapshot.route_rules.length"
                  :label="t('managed.routePosition')"
                /><Input
                  v-if="policy.group_id || policy.dns_server"
                  v-model="policy.dns_index"
                  type="number"
                  :min="0"
                  :max="snapshot.dns_rules.length"
                  :label="t('managed.dnsPosition')"
                />
                <details class="text-xs">
                  <summary class="cursor-pointer text-primary">
                    {{ t('managed.existingRules') }}
                  </summary>
                  <pre class="mt-2 max-h-72 overflow-auto">{{
                    serialized({
                      route: snapshot.route_rules,
                      dns: snapshot.dns_rules,
                    })
                  }}</pre>
                </details>
              </div>
            </div>
            <div class="flex justify-end gap-2">
              <Button
                variant="ghost"
                :disabled="busy"
                @click="cancelPolicy"
                >{{ t('lab.cancel') }}</Button
              ><Button type="submit" :disabled="busy">{{
                t('managed.preview')
              }}</Button>
            </div>
          </form>
        </Card>
        <Card v-if="!snapshot.policies.length && !policy"
          ><p>{{ t('managed.noPolicies') }}</p></Card
        >
        <Card v-for="p in snapshot.policies" :key="p.id"
          ><div class="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h2 class="font-semibold">{{ p.name }}</h2>
              <p class="mt-2 text-sm text-text-secondary">
                {{
                  p.conditions
                    .map((c) => c.kind + ': ' + c.values.join(', '))
                    .join(p.mode === 'and' ? ' AND ' : ' OR ')
                }}
              </p>
              <p class="mt-1 text-sm">
                {{ p.outbound }} ·
                {{
                  snapshot.groups.find((g) => g.id === p.group_id)?.name ||
                  p.dns_server ||
                  t('lab.leaveDnsUnchanged')
                }}
              </p>
              <StatusIndicator
                :variant="p.status === 'synced' ? 'success' : 'warning'"
                class="mt-2"
                >{{ t('managed.' + p.status) }}</StatusIndicator
              >
            </div>
            <div class="flex gap-1">
              <IconButton
                v-if="p.status === 'synced' || p.status === 'missing_dependency'"
                action="edit"
                :label="t('lab.editNamed', { name: p.name })"
                @click="openPolicy(p)"
              /><Button
                v-if="p.status !== 'synced'"
                size="sm"
                variant="secondary"
                @click="policyAction($event, p)"
                >{{ t('managed.review') }}</Button
              ><IconButton
                action="delete"
                :label="t('managed.deleteNamed', { name: p.name })"
                @click="policyAction($event, p)"
              />
            </div></div
        ></Card>
        <RouterLink
          class="inline-block text-sm text-primary underline"
          to="/dashboard/dns/groups"
          >{{ t('lab.resolverGroups') }} →</RouterLink
        >
      </template>
      <div
        v-if="plan && !panelOpen"
        class="space-y-3 rounded-surface border border-border bg-bg-secondary p-4"
      >
        <h2 class="font-semibold">{{ t('managed.proposed') }}</h2>
        <p class="text-sm text-text-secondary">{{ t('managed.validation') }}</p>
        <div class="grid min-w-0 gap-3 lg:grid-cols-2">
          <div>
            <h3 class="text-sm">{{ t('managed.before') }}</h3>
            <pre class="mt-2 max-h-96 overflow-auto text-xs">{{
              serialized(plan.before)
            }}</pre>
          </div>
          <div>
            <h3 class="text-sm">{{ t('managed.after') }}</h3>
            <pre class="mt-2 max-h-96 overflow-auto text-xs">{{
              serialized(plan.after)
            }}</pre>
          </div>
        </div>
        <Button :disabled="busy" @click="commit">{{
          t('managed.apply')
        }}</Button>
      </div>
    </template>
    <Popover
      ref="panel"
      panel-class="w-[min(44rem,calc(100vw-2rem))]"
      @hide="panelClosed"
      ><div class="w-[min(42rem,calc(100vw-4rem))] space-y-4">
        <DNSGroupForm
          v-if="panelKind === 'group' && groupDraft && !plan"
          :key="groupDraft.id + groupDraft.name"
          :initial="groupDraft"
          :groups="snapshot?.groups ?? []"
          :servers="snapshot?.servers ?? []"
          :mode="groupDraft.id ? 'edit' : 'create'"
          live
          :busy="busy"
          @cancel="cancelGroup"
          @save="previewGroup"
        />
        <template v-if="panelKind === 'deleteGroup' && !plan"
          ><h2 class="font-semibold">
            {{ t('managed.deleteNamed', { name: targetGroup?.name }) }}
          </h2>
          <p class="text-sm">{{ t('managed.deleteExplanation') }}</p>
          <p>{{ t('managed.usedBy', { n: dependencies.length }) }}</p>
          <ul class="list-inside list-disc text-sm">
            <li v-for="p in dependencies" :key="p.id">
              {{ p.name }} · {{ t('managed.' + p.status) }}
            </li>
          </ul>
          <Select
            v-if="dependencies.length"
            ref="replacementSelect"
 v-model="replacement"
 @filter="replacementSearch=$event.value"
            :options="snapshot?.groups.filter((g) => g.id !== targetGroup?.id)"
            option-label="name"
            option-value="id"
            filter
            :aria-label="t('managed.replacement')"
            :placeholder="t('managed.replacement')"
            class="w-full"
            @change="clearPlan"
          ><template #footer><button type="button" class="w-full border-t border-border p-3 text-left text-primary" @click="createReplacement">+ {{ t('lab.createDnsGroup') }}</button></template></Select>
          <button v-if="dependencies.length" type="button" class="block text-sm text-primary underline" @click="createReplacement">{{ t('lab.createDnsGroup') }}</button>
          <Button
            :disabled="busy || (dependencies.length > 0 && !replacement)"
            @click="
              preview({
                action: 'delete_group',
                id: targetGroup?.id,
                replacement,
              })
            "
            >{{ t('managed.preview') }}</Button
          >
          <Button
            variant="ghost"
            :disabled="busy"
            @click="preview({ action: 'detach_group', id: targetGroup?.id })"
            >{{ t('managed.detachGroup') }}</Button
          >
        </template>
        <template v-if="panelKind === 'policyAction' && targetPolicy && !plan"
          ><h2 class="font-semibold">{{ targetPolicy.name }}</h2>
          <p class="text-sm">{{ t('managed.driftHelp') }}</p>
          <details class="text-xs">
            <summary>{{ t('managed.savedRules') }}</summary>
            <pre class="max-h-64 overflow-auto">{{
              serialized({
                route: targetPolicy.route_members,
                dns: targetPolicy.dns_members,
              })
            }}</pre>
          </details>
          <div class="flex flex-wrap gap-2">
            <Button
              v-if="targetPolicy.status === 'synced'"
              :disabled="busy"
              @click="preview({ action: 'delete_policy', id: targetPolicy.id })"
              >{{ t('managed.removeOwned') }}</Button
            ><Button
              v-if="targetPolicy.status === 'modified'"
              :disabled="busy"
              @click="preview({ action: 'accept_policy', id: targetPolicy.id })"
              >{{ t('managed.accept') }}</Button
            ><Button
              v-if="targetPolicy.status === 'missing'"
              :disabled="busy"
              @click="
                preview({ action: 'restore_policy', id: targetPolicy.id })
              "
              >{{ t('managed.restore') }}</Button
            ><Button
              variant="secondary"
              :disabled="busy"
              @click="preview({ action: 'detach_policy', id: targetPolicy.id })"
              >{{ t('managed.detach') }}</Button
            >
          </div></template
        >
        <p v-if="error" role="alert" class="text-sm text-danger">{{ error }}</p>
        <template v-if="plan"
          ><h2 class="font-semibold">{{ t('managed.proposed') }}</h2>
          <p class="text-sm">
            {{ t('managed.affected') }}:
            {{ plan.affected.join(', ') || t('managed.metadataOnly') }}
          </p>
          <p class="text-xs text-text-secondary">
            {{ t('managed.validation') }}
          </p>
          <pre
            v-if="plan.command.group"
            class="max-h-40 overflow-auto text-xs"
            >{{ serialized(plan.command.group) }}</pre
          >
          <details open>
            <summary class="text-sm">{{ t('managed.after') }}</summary>
            <pre class="max-h-64 overflow-auto text-xs">{{
              serialized(plan.after)
            }}</pre>
          </details>
          <details>
            <summary class="text-sm">{{ t('managed.before') }}</summary>
            <pre class="max-h-64 overflow-auto text-xs">{{
              serialized(plan.before)
            }}</pre>
          </details>
          <div class="flex justify-end gap-2">
            <Button variant="ghost" :disabled="busy" @click="plan = null">{{
              t('managed.back')
            }}</Button
            ><Button :disabled="busy" @click="commit">{{
              t('managed.apply')
            }}</Button>
          </div></template
        >
      </div></Popover
    >
  </section>
</template>
