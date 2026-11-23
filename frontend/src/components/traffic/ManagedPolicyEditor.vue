<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Button, Card, Input, Textarea, IconButton } from '..'
import FieldHelp from '../FieldHelp.vue'
import Select from '../../volt/Select.vue'
import PolicyRuleSetSelect from './PolicyRuleSetSelect.vue'
import ManagedPlanPreview from './ManagedPlanPreview.vue'
import type { ManagedPolicy, ManagedSnapshot, ManagedPlan } from '../../services/trafficPolicy'

const props = defineProps<{ snapshot: ManagedSnapshot; busy: boolean; error: string; plan: ManagedPlan | null }>()
const policy = defineModel<ManagedPolicy>({ required: true })
interface ConditionDraft { kind: string; value: string; drafts?: Record<string, string> }
const conditions = defineModel<ConditionDraft[]>('conditions', { required: true })
const dnsChoice = defineModel<string>('dnsChoice', { required: true })
const dnsSearch = defineModel<string>('dnsSearch', { required: true })
defineEmits<{ preview: []; cancel: []; apply: []; back: []; createGroup: [event: Event] }>()
const { t } = useI18n()
const dnsAnchor = ref<HTMLElement>()
const dnsSelect = ref<InstanceType<typeof Select>>()
defineExpose({ dnsAnchor, dnsSelect })
const groupOptions = computed(() => [
  { id: 'unchanged', name: t('lab.leaveDnsUnchanged') },
  ...props.snapshot.groups,
  ...props.snapshot.servers.map(tag => ({ id: 'server:' + tag, name: tag })),
])
const kinds = computed(() => ['domain_suffix', 'domain', 'rule_set', 'ip_cidr'].map((value, i) => ({
  value, label: t(['lab.domainSuffix', 'lab.exactDomain', 'lab.ruleSet', 'lab.ipRange'][i]!),
})))
function changeConditionKind(condition: ConditionDraft, kind: string) {
  condition.drafts = { ...condition.drafts, [condition.kind]: condition.value }
  condition.value = condition.drafts[kind] ?? ''
  condition.kind = kind
}
const serialized = (value: unknown) => JSON.stringify(value, null, 2)
function dnsSearchKeydown(event: KeyboardEvent) {
  if (event.key === 'Tab' && !event.shiftKey) {
    event.preventDefault(); event.stopPropagation()
    document.getElementById('managed-dns-create')?.focus()
  }
}
function dnsCreateKeydown(event: KeyboardEvent) {
  if (event.key === 'Tab' && event.shiftKey) {
    event.preventDefault(); document.getElementById('managed-dns-search')?.focus()
  } else if (event.key === 'Escape') {
    event.preventDefault(); event.stopPropagation(); dnsSelect.value?.hide()
    dnsAnchor.value?.querySelector<HTMLElement>('[role="combobox"]')?.focus()
  }
}
</script>

<template>
  <Card class="scroll-mt-4">
    <div class="grid min-w-0 gap-5" :class="{ 'xl:grid-cols-2': plan }">
          <form class="min-w-0 space-y-5" :inert="busy" @submit.prevent="$emit('preview')">
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
                      :model-value="c.kind"
                      @update:model-value="changeConditionKind(c, $event)"
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
                  <PolicyRuleSetSelect v-if="c.kind === 'rule_set'"
                    v-model="c.value" :options="snapshot.rule_sets"
                    :label="t('lab.matchValues', { n: i + 1 })" />
                  <Textarea v-else
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
                        @click="$emit('createGroup', $event)"
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
                    @click="$emit('createGroup', $event)"
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

                <div class="flex items-center gap-1 text-sm">
                  {{ t('managed.priorityLabel') }}
                  <FieldHelp :label="t('managed.priorityLabel')" :text="t('managed.priority')" />
                </div>
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
            <p v-if="error" role="alert" class="text-sm text-danger">{{ error }}</p>
            <div class="flex justify-end gap-2">
              <Button
                variant="ghost"
                :disabled="busy"
                @click="$emit('cancel')"
                >{{ t('lab.cancel') }}</Button
              ><Button type="submit" :loading="busy">{{
                t('managed.preview')
              }}</Button>
            </div>
          </form>
      <ManagedPlanPreview v-if="plan" :plan="plan" :busy="busy"
        @apply="$emit('apply')" @back="$emit('back')" />
    </div>
  </Card>
</template>
