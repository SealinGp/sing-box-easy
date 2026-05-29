<script setup lang="ts">
// Reusable editor for a route rule's *common matching criteria* (inbound,
// protocol, network, domain(s), geosite/geoip, rule_set, port). Self-contained:
// pass any RouteRule via v-model and render it anywhere (route rules dialog,
// DNS rules, init wizard, …). Updates are immutable — each change emits a new
// rule object rather than mutating the bound one in place.
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Chips, MultiSelect } from '../volt'
import type { RouteRule } from '../types/api'

const model = defineModel<RouteRule>({ required: true })
const { t } = useI18n()

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
</script>

<template>
  <div class="space-y-4">
    <div>
      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ t('route.rules.fields.inbound') }}</label>
      <Chips
        v-model="inbound"
        :placeholder="t('route.rules.placeholders.inbound')"
        class="w-full"
      />
    </div>

    <div>
      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ t('route.rules.fields.protocol') }}</label>
      <MultiSelect
        v-model="protocol"
        :options="protocolOptions"
        optionLabel="label"
        optionValue="value"
        :placeholder="t('route.rules.placeholders.protocol')"
        display="chip"
        class="w-full"
      />
    </div>

    <div>
      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ t('route.rules.fields.network') }}</label>
      <MultiSelect
        v-model="network"
        :options="networkOptions"
        optionLabel="label"
        optionValue="value"
        :placeholder="t('route.rules.placeholders.network')"
        display="chip"
        class="w-full"
      />
    </div>

    <div>
      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ t('route.rules.fields.domain') }}</label>
      <Chips
        v-model="domain"
        :placeholder="t('route.rules.placeholders.domain')"
        class="w-full"
      />
    </div>

    <div>
      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ t('route.rules.fields.domainSuffix') }}</label>
      <Chips
        v-model="domainSuffix"
        :placeholder="t('route.rules.placeholders.domainSuffix')"
        class="w-full"
      />
    </div>

    <div>
      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ t('route.rules.fields.geosite') }}</label>
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
    </div>

    <div>
      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ t('route.rules.fields.geoip') }}</label>
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
    </div>

    <div>
      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ t('route.rules.fields.ruleSet') }}</label>
      <Chips
        v-model="ruleSet"
        :placeholder="t('route.rules.placeholders.ruleSet')"
        class="w-full"
      />
    </div>

    <div>
      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ t('route.rules.fields.port') }}</label>
      <Chips
        v-model="port"
        :placeholder="t('route.rules.placeholders.port')"
        class="w-full"
      />
    </div>
  </div>
</template>
