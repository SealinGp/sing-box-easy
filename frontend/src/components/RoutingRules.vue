<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import Card from './Card.vue'
import { Dialog, Button, Select, MultiSelect, Chips } from '../volt'
import RoutingRuleItem from './RoutingRuleItem.vue'
import type { RouteRule, Outbound } from '../types/api'
import { routeService, outboundService } from '../services'
import { useToast } from 'primevue'

// sing-box accepts scalar OR array on the wire for every list-like matcher
// (e.g. "inbound": "dns-in" is equivalent to ["dns-in"]). The backend
// round-trips whatever shape lives in config.json, so coerce to arrays at the
// boundary. Without this: `.join()` crashes on strings, and `[...str]`
// silently splits "dns-in" into characters in startEditRule.
function toArray<T>(v: T | T[] | undefined | null): T[] | undefined {
  if (v === undefined || v === null) return undefined
  return Array.isArray(v) ? v : [v]
}

// Input is the raw wire payload (scalar-or-array), output matches RouteRule
// (post-normalization, arrays only). Cast on entry because the wire shape is
// intentionally not captured in the TS contract — see the note on RouteRule.
function normalizeRouteRule(rule: RouteRule): RouteRule {
  const raw = rule as Record<string, unknown>
  return {
    ...rule,
    inbound: toArray(raw.inbound as string | string[] | undefined),
    protocol: toArray(raw.protocol as string | string[] | undefined),
    network: toArray(raw.network as string | string[] | undefined),
    domain: toArray(raw.domain as string | string[] | undefined),
    domain_suffix: toArray(raw.domain_suffix as string | string[] | undefined),
    domain_keyword: toArray(raw.domain_keyword as string | string[] | undefined),
    domain_regex: toArray(raw.domain_regex as string | string[] | undefined),
    geosite: toArray(raw.geosite as string | string[] | undefined),
    source_geoip: toArray(raw.source_geoip as string | string[] | undefined),
    geoip: toArray(raw.geoip as string | string[] | undefined),
    ip_cidr: toArray(raw.ip_cidr as string | string[] | undefined),
    source_ip_cidr: toArray(raw.source_ip_cidr as string | string[] | undefined),
    source_port: toArray(raw.source_port as number | number[] | undefined),
    port: toArray(raw.port as number | number[] | undefined),
    rule_set: toArray(raw.rule_set as string | string[] | undefined),
    sniffer: toArray(raw.sniffer as string | string[] | undefined),
  }
}

const toast = useToast()

// Local state
const loading = ref(false)
const rules = ref<RouteRule[]>([])
const outbounds = ref<Outbound[]>([])

// State for dialog
const showAddRuleDialog = ref(false)
const editingRule = ref<{ index: number; rule: RouteRule } | null>(null)

// Form data
const ruleForm = ref<RouteRule>({ action: 'route', outbound: '' })

// Fetch data
const fetchRouteRules = async () => {
  loading.value = true
  try {
    const { data } = await routeService.getRouteRules()
    rules.value = (data.rules || []).map(normalizeRouteRule)
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.message || 'Failed to fetch route rules',
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

const fetchOutbounds = async () => {
  try {
    const { data } = await outboundService.getOutbounds()
    outbounds.value = data.outbounds || []
  } catch (err: any) {
    console.error('Failed to fetch outbounds:', err)
  }
}

// Handlers
const handleAddRule = async (rule: RouteRule) => {
  loading.value = true
  try {
    await routeService.addRouteRule(rule)
    toast.add({
      severity: 'success',
      summary: 'Success',
      detail: 'Route rule added successfully',
      life: 3000
    })
    await fetchRouteRules()
    showAddRuleDialog.value = false
    ruleForm.value = { action: 'route', outbound: '' }
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.message || 'Failed to add route rule',
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

const handleEditRule = async (index: number, rule: RouteRule) => {
  loading.value = true
  try {
    await routeService.updateRouteRule(index, rule)
    toast.add({
      severity: 'success',
      summary: 'Success',
      detail: 'Route rule updated successfully',
      life: 3000
    })
    await fetchRouteRules()
    editingRule.value = null
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.message || 'Failed to update route rule',
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

const handleDeleteRule = async (index: number) => {
  if (!confirm('Are you sure you want to delete this rule?')) return

  loading.value = true
  try {
    await routeService.deleteRouteRule(index)
    toast.add({
      severity: 'success',
      summary: 'Success',
      detail: 'Route rule deleted successfully',
      life: 3000
    })
    await fetchRouteRules()
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.message || 'Failed to delete route rule',
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

// Options
const outboundOptions = computed(() =>
  outbounds.value.map(o => ({ label: o.tag || '', value: o.tag || '' }))
)

const networkOptions = [
  { label: 'TCP', value: 'tcp' },
  { label: 'UDP', value: 'udp' },
]

const actionOptions = [
  { label: 'Route', value: 'route' },
  { label: 'Reject', value: 'reject' },
  { label: 'Route Options', value: 'route-options' },
  { label: 'Sniff', value: 'sniff' },
  { label: 'Resolve', value: 'resolve' },
  { label: 'Hijack DNS', value: 'hijack-dns' },
]

const rejectMethodOptions = [
  { label: 'Default', value: 'default' },
  { label: 'Drop', value: 'drop' },
]

const networkStrategyOptions = [
  { label: 'Prefer IPv4', value: 'prefer_ipv4' },
  { label: 'Prefer IPv6', value: 'prefer_ipv6' },
  { label: 'IPv4 Only', value: 'ipv4_only' },
  { label: 'IPv6 Only', value: 'ipv6_only' },
]

const dnsStrategyOptions = [
  { label: 'Prefer IPv4', value: 'prefer_ipv4' },
  { label: 'Prefer IPv6', value: 'prefer_ipv6' },
  { label: 'IPv4 Only', value: 'ipv4_only' },
  { label: 'IPv6 Only', value: 'ipv6_only' },
]

const protocolOptions = [
  { label: 'HTTP', value: 'http' },
  { label: 'HTTPS', value: 'tls' },
  { label: 'QUIC', value: 'quic' },
]

const geositeOptions = [
  { label: 'Google', value: 'google' },
  { label: 'Netflix', value: 'netflix' },
  { label: 'YouTube', value: 'youtube' },
  { label: 'OpenAI', value: 'openai' },
  { label: 'Microsoft', value: 'microsoft' },
  { label: 'Apple', value: 'apple' },
  { label: 'Telegram', value: 'telegram' },
  { label: 'Geolocation- CN', value: 'geolocation-cn' },
  { label: 'Geolocation- !CN', value: 'geolocation-!cn' },
  { label: 'CN', value: 'cn' },
  { label: 'Private', value: 'private' },
  { label: 'Category-Ads', value: 'category-ads' },
]

const geoipOptions = [
  { label: 'Private', value: 'private' },
  { label: 'CN', value: 'cn' },
  { label: 'US', value: 'us' },
  { label: 'JP', value: 'jp' },
  { label: 'HK', value: 'hk' },
  { label: 'TW', value: 'tw' },
  { label: 'SG', value: 'sg' },
]

// Computed properties for v-model
const currentRuleOutbound = computed({
  get: () => editingRule.value ? editingRule.value.rule.outbound : ruleForm.value.outbound,
  set: (val) => {
    if (editingRule.value) {
      editingRule.value.rule.outbound = val
    } else {
      ruleForm.value.outbound = val
    }
  }
})

const currentRuleInbound = computed({
  get: () => editingRule.value ? editingRule.value.rule.inbound : ruleForm.value.inbound,
  set: (val) => {
    if (editingRule.value) {
      editingRule.value.rule.inbound = val
    } else {
      ruleForm.value.inbound = val
    }
  }
})

const currentRuleProtocol = computed({
  get: () => editingRule.value ? editingRule.value.rule.protocol : ruleForm.value.protocol,
  set: (val) => {
    if (editingRule.value) {
      editingRule.value.rule.protocol = val
    } else {
      ruleForm.value.protocol = val
    }
  }
})

const currentRuleNetwork = computed({
  get: () => editingRule.value ? editingRule.value.rule.network : ruleForm.value.network,
  set: (val) => {
    if (editingRule.value) {
      editingRule.value.rule.network = val
    } else {
      ruleForm.value.network = val
    }
  }
})

const currentRuleDomain = computed({
  get: () => editingRule.value ? editingRule.value.rule.domain : ruleForm.value.domain,
  set: (val) => {
    if (editingRule.value) {
      editingRule.value.rule.domain = val
    } else {
      ruleForm.value.domain = val
    }
  }
})

const currentRuleDomainSuffix = computed({
  get: () => editingRule.value ? editingRule.value.rule.domain_suffix : ruleForm.value.domain_suffix,
  set: (val) => {
    if (editingRule.value) {
      editingRule.value.rule.domain_suffix = val
    } else {
      ruleForm.value.domain_suffix = val
    }
  }
})

const currentRuleGeosite = computed({
  get: () => editingRule.value ? editingRule.value.rule.geosite : ruleForm.value.geosite,
  set: (val) => {
    if (editingRule.value) {
      editingRule.value.rule.geosite = val
    } else {
      ruleForm.value.geosite = val
    }
  }
})

const currentRuleGeoip = computed({
  get: () => editingRule.value ? editingRule.value.rule.geoip : ruleForm.value.geoip,
  set: (val) => {
    if (editingRule.value) {
      editingRule.value.rule.geoip = val
    } else {
      ruleForm.value.geoip = val
    }
  }
})

const currentRulePort = computed({
  get: () => {
    const ports = editingRule.value ? editingRule.value.rule.port : ruleForm.value.port
    // Convert number[] to string[] for Chips component
    return ports?.map(p => String(p))
  },
  set: (val) => {
    // Convert string[] back to number[] if needed, or keep as is for port ranges
    const ports = val?.map(p => {
      // If it's a range like "8080-8090", keep as string
      // Otherwise convert to number
      return /^\d+$/.test(p) ? Number(p) : p
    }) as any
    if (editingRule.value) {
      editingRule.value.rule.port = ports
    } else {
      ruleForm.value.port = ports
    }
  }
})

const currentRuleAction = computed({
  get: () => editingRule.value ? editingRule.value.rule.action : ruleForm.value.action,
  set: (val) => {
    if (editingRule.value) {
      editingRule.value.rule.action = val
    } else {
      ruleForm.value.action = val
    }
  }
})

const currentAction = computed(() => editingRule.value ? editingRule.value.rule.action : ruleForm.value.action)

// Action-specific computed properties

// Reject action
const currentRuleMethod = computed({
  get: () => editingRule.value ? editingRule.value.rule.method : ruleForm.value.method,
  set: (val) => {
    if (editingRule.value) {
      editingRule.value.rule.method = val
    } else {
      ruleForm.value.method = val
    }
  }
})

// Route-options action
const currentRuleOverrideAddress = computed({
  get: () => editingRule.value ? editingRule.value.rule.override_address : ruleForm.value.override_address,
  set: (val) => {
    if (editingRule.value) {
      editingRule.value.rule.override_address = val
    } else {
      ruleForm.value.override_address = val
    }
  }
})

const currentRuleOverridePort = computed({
  get: () => editingRule.value ? editingRule.value.rule.override_port : ruleForm.value.override_port,
  set: (val) => {
    if (editingRule.value) {
      editingRule.value.rule.override_port = val
    } else {
      ruleForm.value.override_port = val
    }
  }
})

const currentRuleNetworkStrategy = computed({
  get: () => editingRule.value ? editingRule.value.rule.network_strategy : ruleForm.value.network_strategy,
  set: (val) => {
    if (editingRule.value) {
      editingRule.value.rule.network_strategy = val
    } else {
      ruleForm.value.network_strategy = val
    }
  }
})

// Sniff action
const currentRuleSniffer = computed({
  get: () => {
    const sniffer = editingRule.value ? editingRule.value.rule.sniffer : ruleForm.value.sniffer
    return Array.isArray(sniffer) ? sniffer : (sniffer ? [sniffer] : [])
  },
  set: (val) => {
    if (editingRule.value) {
      editingRule.value.rule.sniffer = val
    } else {
      ruleForm.value.sniffer = val
    }
  }
})

const currentRuleTimeout = computed({
  get: () => editingRule.value ? editingRule.value.rule.timeout : ruleForm.value.timeout,
  set: (val) => {
    if (editingRule.value) {
      editingRule.value.rule.timeout = val
    } else {
      ruleForm.value.timeout = val
    }
  }
})

// Resolve action
const currentRuleServer = computed({
  get: () => editingRule.value ? editingRule.value.rule.server : ruleForm.value.server,
  set: (val) => {
    if (editingRule.value) {
      editingRule.value.rule.server = val
    } else {
      ruleForm.value.server = val
    }
  }
})

const currentRuleStrategy = computed({
  get: () => editingRule.value ? editingRule.value.rule.strategy : ruleForm.value.strategy,
  set: (val) => {
    if (editingRule.value) {
      editingRule.value.rule.strategy = val
    } else {
      ruleForm.value.strategy = val
    }
  }
})

// Dialog visibility
const dialogVisible = computed({
  get: () => showAddRuleDialog.value || !!editingRule.value,
  set: (val) => {
    if (!val) {
      showAddRuleDialog.value = false
      editingRule.value = null
      ruleForm.value = { action: 'route', outbound: '' }
    }
  }
})

// Functions
function startEditRule(index: number, rule: RouteRule) {
  // Ensure add dialog is closed
  showAddRuleDialog.value = false

  // Deep copy the rule to avoid mutations
  const ruleCopy: RouteRule = {
    ...rule,
    // Common matching criteria
    inbound: rule.inbound ? [...rule.inbound] : undefined,
    protocol: rule.protocol ? [...rule.protocol] : undefined,
    network: rule.network ? [...rule.network] : undefined,
    domain: rule.domain ? [...rule.domain] : undefined,
    domain_suffix: rule.domain_suffix ? [...rule.domain_suffix] : undefined,
    domain_keyword: rule.domain_keyword ? [...rule.domain_keyword] : undefined,
    domain_regex: rule.domain_regex ? [...rule.domain_regex] : undefined,
    geosite: rule.geosite ? (Array.isArray(rule.geosite) ? [...rule.geosite] : rule.geosite) : undefined,
    source_geoip: rule.source_geoip ? [...rule.source_geoip] : undefined,
    geoip: rule.geoip ? (Array.isArray(rule.geoip) ? [...rule.geoip] : rule.geoip) : undefined,
    ip_cidr: rule.ip_cidr ? [...rule.ip_cidr] : undefined,
    source_ip_cidr: rule.source_ip_cidr ? [...rule.source_ip_cidr] : undefined,
    source_port: rule.source_port ? [...rule.source_port] : undefined,
    port: rule.port ? [...rule.port] : undefined,
    rule_set: rule.rule_set ? (Array.isArray(rule.rule_set) ? [...rule.rule_set] : rule.rule_set) : undefined,
    // Sniffer field can be string[] or string
    sniffer: rule.sniffer ? (Array.isArray(rule.sniffer) ? [...rule.sniffer] : rule.sniffer) : undefined,
  }

  editingRule.value = { index, rule: ruleCopy }
}

function submitAddRule() {
  handleAddRule(ruleForm.value)
}

function submitUpdateRule() {
  if (editingRule.value) {
    handleEditRule(editingRule.value.index, editingRule.value.rule)
  }
}

function submitDeleteRule(index: number) {
  handleDeleteRule(index)
}

// Load data on mount
onMounted(() => {
  fetchRouteRules()
  fetchOutbounds()
})
</script>

<template>
  <div class="space-y-6">
    <Card>
      <div class="flex justify-between items-center mb-4">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">
          Routing Rules
        </h3>
        <button
          @click="showAddRuleDialog = true"
          class="px-4 py-2 bg-violet-600 text-white rounded-md hover:bg-violet-700 transition-colors"
        >
          Add Rule
        </button>
      </div>

      <div v-if="loading" class="text-center py-8">
        <div class="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-violet-600"></div>
      </div>

      <div v-else-if="rules.length === 0" class="text-center py-8 text-gray-500 dark:text-gray-400">
        No routing rules configured
      </div>

      <div v-else class="space-y-4">
        <RoutingRuleItem
          v-for="(rule, index) in rules"
          :key="index"
          :rule="rule"
          :index="index"
          @edit="startEditRule"
          @delete="submitDeleteRule"
        />
      </div>
    </Card>

    <!-- Add/Edit Rule Dialog -->
    <Dialog
      v-model:visible="dialogVisible"
      modal
      :header="editingRule ? 'Edit Rule' : 'Add Routing Rule'"
      class="w-full max-w-2xl"
    >
      <div :key="editingRule ? `edit-${editingRule.index}` : 'add'" class="space-y-4">
        <!-- Action Selection -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Action *</label>
          <Select
            v-model="currentRuleAction"
            :options="actionOptions"
            optionLabel="label"
            optionValue="value"
            placeholder="Select action"
            class="w-full"
          />
        </div>

        <!-- Action-specific fields -->

        <!-- Route Action -->
        <div v-if="currentAction === 'route'">
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Outbound *</label>
          <Select
            editable
            v-model="currentRuleOutbound"
            :options="outboundOptions"
            optionLabel="label"
            optionValue="value"
            placeholder="Select outbound"
            class="w-full"
          />
        </div>

        <!-- Reject Action -->
        <template v-if="currentAction === 'reject'">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Method</label>
            <Select
              v-model="currentRuleMethod"
              :options="rejectMethodOptions"
              optionLabel="label"
              optionValue="value"
              placeholder="Select method"
              class="w-full"
            />
          </div>
        </template>

        <!-- Route Options Action -->
        <template v-if="currentAction === 'route-options'">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Override Address</label>
            <input
              v-model="currentRuleOverrideAddress"
              type="text"
              placeholder="Override address"
              class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
            />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Override Port</label>
            <input
              v-model.number="currentRuleOverridePort"
              type="number"
              placeholder="Override port"
              class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
            />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Network Strategy</label>
            <Select
              v-model="currentRuleNetworkStrategy"
              :options="networkStrategyOptions"
              optionLabel="label"
              optionValue="value"
              placeholder="Select network strategy"
              class="w-full"
            />
          </div>
        </template>

        <!-- Sniff Action -->
        <template v-if="currentAction === 'sniff'">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Sniffer</label>
            <Chips
              v-model="currentRuleSniffer"
              placeholder="Add sniffer protocols (e.g., tls, http)"
              class="w-full"
            />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Timeout</label>
            <input
              v-model="currentRuleTimeout"
              type="text"
              placeholder="e.g., 300ms"
              class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
            />
          </div>
        </template>

        <!-- Resolve Action -->
        <template v-if="currentAction === 'resolve'">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">DNS Server</label>
            <input
              v-model="currentRuleServer"
              type="text"
              placeholder="DNS server tag"
              class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
            />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Strategy</label>
            <Select
              v-model="currentRuleStrategy"
              :options="dnsStrategyOptions"
              optionLabel="label"
              optionValue="value"
              placeholder="Select DNS strategy"
              class="w-full"
            />
          </div>
        </template>

        <!-- Common matching criteria fields -->

        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Inbound</label>
          <Chips
            v-model="currentRuleInbound"
            placeholder="Add inbound tags"
            class="w-full"
          />
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Protocol</label>
          <MultiSelect
            v-model="currentRuleProtocol"
            :options="protocolOptions"
            optionLabel="label"
            optionValue="value"
            placeholder="Select protocols"
            display="chip"
            class="w-full"
          />
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Network</label>
          <MultiSelect
            v-model="currentRuleNetwork"
            :options="networkOptions"
            optionLabel="label"
            optionValue="value"
            placeholder="Select network types"
            display="chip"
            class="w-full"
          />
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Domain</label>
          <Chips
            v-model="currentRuleDomain"
            placeholder="Add domains (e.g., google.com, youtube.com)"
            class="w-full"
          />
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Domain Suffix</label>
          <Chips
            v-model="currentRuleDomainSuffix"
            placeholder="Add domain suffixes (e.g., .com, .org)"
            class="w-full"
          />
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">GeoSite</label>
          <MultiSelect
            v-model="currentRuleGeosite"
            :options="geositeOptions"
            optionLabel="label"
            optionValue="value"
            placeholder="Select geosite categories"
            display="chip"
            filter
            class="w-full"
          />
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">GeoIP</label>
          <MultiSelect
            v-model="currentRuleGeoip"
            :options="geoipOptions"
            optionLabel="label"
            optionValue="value"
            placeholder="Select geoip categories"
            display="chip"
            filter
            class="w-full"
          />
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Port</label>
          <Chips
            v-model="currentRulePort"
            placeholder="Add ports (e.g., 80, 443, 8080-8090)"
            class="w-full"
          />
        </div>
      </div>

      <template #footer>
        <Button
          label="Cancel"
          severity="secondary"
          @click="dialogVisible = false"
        />
        <Button
          :label="editingRule ? 'Update' : 'Add'"
          @click="editingRule ? submitUpdateRule() : submitAddRule()"
        />
      </template>
    </Dialog>
  </div>
</template>