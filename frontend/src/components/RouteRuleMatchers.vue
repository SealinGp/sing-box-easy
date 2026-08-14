<script setup lang="ts">
/**
 * Reusable editor for a route rule's *common matching criteria*. Self-contained:
 * pass any RouteRule via v-model and render it anywhere (route rules dialog,
 * init wizard, …). Updates are immutable — each change emits a new rule object
 * rather than mutating the bound one in place.
 *
 * Two behaviours are shared with the DNS rule form and live in
 * `composables/useMatcherFields.ts`, which carries the full reasoning:
 * show-only-what-is-filled, and the rule-set/content-matcher either-or.
 *
 * The route form splits its matchers where the DNS form did not need to, and
 * the split is the load-bearing decision here:
 *
 *   CONTENT matchers (domain, domain_suffix, geosite, geoip) describe *what*
 *   the traffic is. A rule set already expresses exactly these, so combining
 *   the two in one rule is the AND trap — "in the rule set AND also this
 *   domain" — and they are therefore presented as alternatives to `rule_set`.
 *
 *   CONTEXT matchers (inbound, protocol, network, port) describe *where the
 *   traffic came from*. A rule set cannot express them, so narrowing a rule set
 *   by `network: udp` or `port: 443` is a normal, correct thing to do. These
 *   are never folded away by the rule-set choice — only by their own emptiness.
 *
 * Collapsing every matcher against `rule_set` would have been the simpler code
 * and the wrong model: it would flag the most common correct route rule in
 * sing-box (a rule set narrowed by port or network) as a mistake.
 */
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { storeToRefs } from 'pinia'
import { MultiSelect } from '../volt'
import ChipsField from './ChipsField.vue'
import LabeledField from './LabeledField.vue'
import Alert from './Alert.vue'
import { PlusCircleIcon } from '@heroicons/vue/24/outline'
import { useExclusiveMatcherGroups, useOptionalFields } from '../composables/useMatcherFields'
import { useRouteStore } from '../stores/route'
import type { RouteRule } from '../types/api'

const model = defineModel<RouteRule>({ required: true })
const { t } = useI18n()

// Rule sets come from the config, so they are picked, never typed: a tag that
// does not exist in `route.rule_sets` makes sing-box reject the whole config,
// and a typo is invisible until the next validate. `ensureRuleSets` loads once
// per session and collapses concurrent callers, so reopening the dialog does
// not refetch.
const routeStore = useRouteStore()
const { ruleSets } = storeToRefs(routeStore)

onMounted(() => {
  routeStore.ensureRuleSets()
})

// Immutable field update: replace the rule with a shallow copy carrying the
// new value. Keeps the parent's reactivity simple and avoids prop mutation.
function update<K extends keyof RouteRule>(key: K, val: RouteRule[K]) {
  model.value = { ...model.value, [key]: val }
}

const networkOptions = computed(() => [
  { label: t('route.rules.networks.tcp'), value: 'tcp' },
  { label: t('route.rules.networks.udp'), value: 'udp' },
])

const protocolOptions = computed(() => [
  { label: t('route.rules.protocols.http'), value: 'http' },
  { label: t('route.rules.protocols.https'), value: 'tls' },
  { label: t('route.rules.protocols.quic'), value: 'quic' },
])

// GeoSite/GeoIP labels are technical geo codes that double as the wire values;
// only the few human-readable variants route through the catalog.
const geositeOptions = computed(() => [
  { label: 'Google', value: 'google' },
  { label: 'Netflix', value: 'netflix' },
  { label: 'YouTube', value: 'youtube' },
  { label: 'OpenAI', value: 'openai' },
  { label: 'Microsoft', value: 'microsoft' },
  { label: 'Apple', value: 'apple' },
  { label: 'Telegram', value: 'telegram' },
  { label: t('route.rules.geosite.geolocationCn'), value: 'geolocation-cn' },
  { label: t('route.rules.geosite.geolocationNotCn'), value: 'geolocation-!cn' },
  { label: 'CN', value: 'cn' },
  { label: 'Private', value: 'private' },
  { label: t('route.rules.geosite.categoryAds'), value: 'category-ads' },
])

const geoipOptions = [
  { label: 'Private', value: 'private' },
  { label: 'CN', value: 'cn' },
  { label: 'US', value: 'us' },
  { label: 'JP', value: 'jp' },
  { label: 'HK', value: 'hk' },
  { label: 'TW', value: 'tw' },
  { label: 'SG', value: 'sg' },
]

const inbound = computed({
  get: () => model.value.inbound,
  set: (v) => update('inbound', v),
})

const protocol = computed({
  get: () => model.value.protocol,
  set: (v) => update('protocol', v),
})

const network = computed({
  get: () => model.value.network,
  set: (v) => update('network', v),
})

const domain = computed({
  get: () => model.value.domain,
  set: (v) => update('domain', v),
})

const domainSuffix = computed({
  get: () => model.value.domain_suffix,
  set: (v) => update('domain_suffix', v),
})

const geosite = computed({
  get: () => model.value.geosite,
  set: (v) => update('geosite', v),
})

const geoip = computed({
  get: () => model.value.geoip,
  set: (v) => update('geoip', v),
})

const ruleSet = computed({
  get: () => {
    const rs = model.value.rule_set
    return Array.isArray(rs) ? rs : rs ? [rs] : undefined
  },
  set: (v) => update('rule_set', v),
})

const ruleSetOptions = computed(() => {
  const options = ruleSets.value
    .filter((rs) => !!rs.tag)
    .map((rs) => ({
      value: rs.tag,
      label: t('route.rules.ruleSetLabel', {
        tag: rs.tag,
        type: rs.type || 'local',
        format: rs.format || 'source',
      }),
    }))

  // A rule can name a rule set that was since renamed or deleted. MultiSelect
  // renders nothing for a value it has no option for, so that tag would vanish
  // from the form while still living in the rule — and the next save would look
  // like the operator had removed it. Surface it as a flagged option instead, so
  // it stays visible, stays selected, and can be deliberately removed.
  const known = new Set(options.map((o) => o.value))
  for (const tag of ruleSet.value ?? []) {
    if (known.has(tag)) continue
    known.add(tag)
    options.push({ value: tag, label: t('route.rules.ruleSetMissing', { tag }) })
  }

  return options
})

// Chips works with strings; ports are numbers on the wire. Round-trip with the
// rule that "8080-8090" range syntax stays a string while plain ports become
// numbers.
const port = computed({
  get: () => model.value.port?.map((p) => String(p)),
  set: (v: string[] | undefined) => {
    const ports = v?.map((p) => (/^\d+$/.test(p) ? Number(p) : p)) as any
    update('port', ports)
  },
})

// ── Field registry ──────────────────────────────────────────────────────────
type ContentKey = 'domain' | 'domainSuffix' | 'geosite' | 'geoip'
type ContextKey = 'inbound' | 'protocol' | 'network' | 'port'
type MatcherKey = ContentKey | ContextKey

const CONTENT_KEYS: readonly ContentKey[] = ['domain', 'domainSuffix', 'geosite', 'geoip']
const CONTEXT_KEYS: readonly ContextKey[] = ['inbound', 'protocol', 'network', 'port']

// One place to read every field's current value, so "is this filled?" cannot
// disagree with what the template renders. Typed as plain `unknown` because
// sing-box accepts a scalar OR an array for these fields (`"geosite": "cn"` is
// equivalent to `["cn"]`), and a rule loaded straight from config.json can
// carry either shape — see `normalizeRouteRule` in RoutingRules.vue.
const values: Record<MatcherKey, { value: unknown }> = {
  domain,
  domainSuffix,
  geosite,
  geoip,
  inbound,
  protocol,
  network,
  port,
}

// Labels live here so the "add" buttons and the field headers can never drift.
const labelKeys: Record<MatcherKey, string> = {
  domain: 'route.rules.fields.domain',
  domainSuffix: 'route.rules.fields.domainSuffix',
  geosite: 'route.rules.fields.geosite',
  geoip: 'route.rules.fields.geoip',
  inbound: 'route.rules.fields.inbound',
  protocol: 'route.rules.fields.protocol',
  network: 'route.rules.fields.network',
  port: 'route.rules.fields.port',
}

// A bare scalar counts as filled: hiding a field that holds `"geosite": "cn"`
// would drop a live condition out of the form.
const isFilled = (key: MatcherKey) => {
  const value = values[key].value
  if (Array.isArray(value)) return value.length > 0
  return value !== undefined && value !== null && value !== ''
}

const content = useOptionalFields(CONTENT_KEYS, isFilled)
const context = useOptionalFields(CONTEXT_KEYS, isFilled)

const hasRuleSet = computed(() => (ruleSet.value?.length ?? 0) > 0)
const hasMatchers = computed(() => CONTENT_KEYS.some(isFilled))

const {
  showRuleSet,
  showMatchers,
  expandRuleSet,
  expandMatchers,
  collapseRuleSet,
  collapseMatchers,
  canHideRuleSet,
  canHideMatchers,
  showMixWarning,
} = useExclusiveMatcherGroups({ hasRuleSet, hasMatchers })
</script>

<template>
  <div class="space-y-4">
    <Alert v-if="showMixWarning" type="warning" :title="t('route.rules.mixing.title')">
      {{ t('route.rules.mixing.warning') }}
    </Alert>

    <!-- ── Rule set ─────────────────────────────────────────────────────── -->
    <div v-if="showRuleSet">
      <div class="flex items-center justify-between mb-1">
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('route.rules.fields.ruleSet') }}
        </label>
        <button
          v-if="canHideRuleSet"
          type="button"
          class="text-xs text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 transition-colors"
          @click="collapseRuleSet"
        >
          {{ t('route.rules.mixing.hide') }}
        </button>
      </div>
      <MultiSelect
        v-model="ruleSet"
        :options="ruleSetOptions"
        optionLabel="label"
        optionValue="value"
        :placeholder="t('route.rules.placeholders.ruleSet')"
        :emptyMessage="t('route.rules.ruleSetEmpty')"
        display="chip"
        filter
        class="w-full"
      />
      <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
        {{ t('route.rules.ruleSetHelp') }}
      </p>
    </div>

    <button
      v-else
      type="button"
      class="w-full flex items-center gap-2 px-3 py-2 rounded-control border border-dashed border-gray-300 dark:border-gray-600 text-left text-xs text-gray-500 dark:text-gray-400 hover:border-primary-400 hover:text-primary-600 dark:hover:text-primary-300 transition-colors"
      @click="expandRuleSet"
    >
      <PlusCircleIcon class="h-4 w-4 shrink-0" />
      <span class="flex-1">{{ t('route.rules.mixing.ruleSetCollapsed') }}</span>
      <span class="font-medium">{{ t('route.rules.mixing.show') }}</span>
    </button>

    <!-- ── Content conditions: the alternative to a rule set ─────────────── -->
    <template v-if="showMatchers">
      <div v-if="canHideMatchers" class="flex items-center justify-between">
        <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('route.rules.mixing.matchersGroup') }}
        </span>
        <button
          type="button"
          class="text-xs text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 transition-colors"
          @click="collapseMatchers"
        >
          {{ t('route.rules.mixing.hide') }}
        </button>
      </div>

      <ChipsField
        v-if="content.isShown('domain')"
        v-model="domain"
        :label="t('route.rules.fields.domain')"
        :placeholder="t('route.rules.placeholders.domain')"
        :removable="content.isRemovable('domain')"
        @remove="content.remove('domain')"
      />

      <ChipsField
        v-if="content.isShown('domainSuffix')"
        v-model="domainSuffix"
        :label="t('route.rules.fields.domainSuffix')"
        :placeholder="t('route.rules.placeholders.domainSuffix')"
        :removable="content.isRemovable('domainSuffix')"
        @remove="content.remove('domainSuffix')"
      />

      <LabeledField
        v-if="content.isShown('geosite')"
        :label="t('route.rules.fields.geosite')"
        :removable="content.isRemovable('geosite')"
        @remove="content.remove('geosite')"
      >
        <MultiSelect
          v-model="geosite"
          :options="geositeOptions"
          optionLabel="label"
          optionValue="value"
          :placeholder="t('route.rules.placeholders.geosite')"
          display="chip"
          filter
          class="w-full"
        />
      </LabeledField>

      <LabeledField
        v-if="content.isShown('geoip')"
        :label="t('route.rules.fields.geoip')"
        :removable="content.isRemovable('geoip')"
        @remove="content.remove('geoip')"
      >
        <MultiSelect
          v-model="geoip"
          :options="geoipOptions"
          optionLabel="label"
          optionValue="value"
          :placeholder="t('route.rules.placeholders.geoip')"
          display="chip"
          filter
          class="w-full"
        />
      </LabeledField>

      <div v-if="content.hidden.value.length" class="flex flex-wrap items-center gap-2">
        <span class="text-xs text-gray-500 dark:text-gray-400">
          {{ t('route.rules.matchers.addContent') }}
        </span>
        <button
          v-for="key in content.hidden.value"
          :key="key"
          type="button"
          class="inline-flex items-center gap-1 px-2.5 py-1 rounded-pill border border-dashed border-gray-300 dark:border-gray-600 text-xs text-gray-600 dark:text-gray-300 hover:border-primary-400 hover:text-primary-600 dark:hover:text-primary-300 transition-colors"
          @click="content.add(key)"
        >
          <PlusCircleIcon class="h-3.5 w-3.5" />
          {{ t(labelKeys[key]) }}
        </button>
      </div>
    </template>

    <button
      v-else
      type="button"
      class="w-full flex items-center gap-2 px-3 py-2 rounded-control border border-dashed border-gray-300 dark:border-gray-600 text-left text-xs text-gray-500 dark:text-gray-400 hover:border-primary-400 hover:text-primary-600 dark:hover:text-primary-300 transition-colors"
      @click="expandMatchers"
    >
      <PlusCircleIcon class="h-4 w-4 shrink-0" />
      <span class="flex-1">{{ t('route.rules.mixing.matchersCollapsed') }}</span>
      <span class="font-medium">{{ t('route.rules.mixing.show') }}</span>
    </button>

    <!-- ── Context conditions ───────────────────────────────────────────────
         Never folded by the rule-set choice: a rule set cannot express where
         traffic came from, so narrowing one by network/port/inbound is correct,
         not an accidental intersection. -->
    <div class="border-t border-gray-200 dark:border-gray-700 pt-4 space-y-4">
      <ChipsField
        v-if="context.isShown('inbound')"
        v-model="inbound"
        :label="t('route.rules.fields.inbound')"
        :placeholder="t('route.rules.placeholders.inbound')"
        :removable="context.isRemovable('inbound')"
        @remove="context.remove('inbound')"
      />

      <LabeledField
        v-if="context.isShown('protocol')"
        :label="t('route.rules.fields.protocol')"
        :removable="context.isRemovable('protocol')"
        @remove="context.remove('protocol')"
      >
        <MultiSelect
          v-model="protocol"
          :options="protocolOptions"
          optionLabel="label"
          optionValue="value"
          :placeholder="t('route.rules.placeholders.protocol')"
          display="chip"
          class="w-full"
        />
      </LabeledField>

      <LabeledField
        v-if="context.isShown('network')"
        :label="t('route.rules.fields.network')"
        :removable="context.isRemovable('network')"
        @remove="context.remove('network')"
      >
        <MultiSelect
          v-model="network"
          :options="networkOptions"
          optionLabel="label"
          optionValue="value"
          :placeholder="t('route.rules.placeholders.network')"
          display="chip"
          class="w-full"
        />
      </LabeledField>

      <ChipsField
        v-if="context.isShown('port')"
        v-model="port"
        :label="t('route.rules.fields.port')"
        :placeholder="t('route.rules.placeholders.port')"
        :removable="context.isRemovable('port')"
        @remove="context.remove('port')"
      />

      <div v-if="context.hidden.value.length" class="flex flex-wrap items-center gap-2">
        <span class="text-xs text-gray-500 dark:text-gray-400">
          {{ t('route.rules.matchers.addContext') }}
        </span>
        <button
          v-for="key in context.hidden.value"
          :key="key"
          type="button"
          class="inline-flex items-center gap-1 px-2.5 py-1 rounded-pill border border-dashed border-gray-300 dark:border-gray-600 text-xs text-gray-600 dark:text-gray-300 hover:border-primary-400 hover:text-primary-600 dark:hover:text-primary-300 transition-colors"
          @click="context.add(key)"
        >
          <PlusCircleIcon class="h-3.5 w-3.5" />
          {{ t(labelKeys[key]) }}
        </button>
      </div>
    </div>
  </div>
</template>
