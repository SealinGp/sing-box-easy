<script setup lang="ts">
import { computed } from 'vue'
import type { RouteRule } from '../types/api'
import PopConfirm from './PopConfirm.vue'

interface Props {
  rule: RouteRule
  index: number
}

interface Emits {
  (e: 'edit', index: number, rule: RouteRule): void
  (e: 'delete', index: number): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

// Defensive renderer for list-like matchers. sing-box can return a scalar
// ("inbound": "dns-in") or an array (["dns-in"]); RoutingRules.vue normalizes
// at the boundary, but this guard keeps the item safe if any caller forgets.
// Returns '' for nullish so the v-if (which uses `||`) hides the row.
function formatList(v: unknown): string {
  if (v === undefined || v === null) return ''
  if (Array.isArray(v)) return v.join(', ')
  return String(v)
}

function hasValue(v: unknown): boolean {
  if (v === undefined || v === null) return false
  if (Array.isArray(v)) return v.length > 0
  if (typeof v === 'string') return v.length > 0
  return true
}

function handleEdit() {
  emit('edit', props.index, props.rule)
}

function handleDelete() {
  emit('delete', props.index)
}

/*
 * A routing rule has no user-assigned name, so the delete confirmation would
 * otherwise say only "delete this rule?" with nothing to check it against.
 * Build a short identity from the rule's most distinctive matcher plus its
 * destination — e.g. `#2 · rule_set: geosite-cn → direct`.
 *
 * Order matters: the earlier keys are the ones a human actually recognises a
 * rule by, so the first one present wins.
 */
const IDENTIFYING_MATCHERS = [
  'rule_set',
  'geosite',
  'geoip',
  'domain',
  'domain_suffix',
  'domain_keyword',
  'domain_regex',
  'ip_cidr',
  'source_ip_cidr',
  'port',
  'source_port',
  'protocol',
  'network',
  'inbound',
] as const

// Keeps the summary to roughly two lines in the popover. The destination is
// appended after this, and it must survive the clip — it is the half of the
// summary that says what the rule actually does.
const MAX_MATCHER_CHARS = 24

const ruleSummary = computed(() => {
  const rule = props.rule as Record<string, unknown>
  // `action` defaults to "route" when omitted — mirror that here so a rule
  // without an explicit action still reads sensibly.
  const destination = props.rule.outbound || props.rule.action || 'route'
  const position = `#${props.index + 1}`

  for (const key of IDENTIFYING_MATCHERS) {
    if (!hasValue(rule[key])) continue
    const rendered = formatList(rule[key])
    const clipped =
      rendered.length > MAX_MATCHER_CHARS
        ? `${rendered.slice(0, MAX_MATCHER_CHARS)}…`
        : rendered
    return `${position} · ${key}: ${clipped} → ${destination}`
  }

  // A rule with no matchers at all still needs to be distinguishable.
  return `${position} · → ${destination}`
})
</script>

<template>
  <div class="border border-gray-200 dark:border-gray-700 rounded-surface p-4">
    <div class="flex justify-between items-start">
      <div class="flex-1">
        <div class="grid grid-cols-2 gap-4 text-sm">
          <div v-if="rule.action">
            <span class="font-medium text-gray-700 dark:text-gray-300">{{ $t('route.ruleItem.action') }}</span>
            <span class="ml-2 text-gray-900 dark:text-gray-100">{{ rule.action }}</span>
          </div>
          <div v-if="rule.outbound">
            <span class="font-medium text-gray-700 dark:text-gray-300">{{ $t('route.ruleItem.outbound') }}</span>
            <span class="ml-2 text-gray-900 dark:text-gray-100">{{ rule.outbound }}</span>
          </div>
          <div v-if="hasValue(rule.inbound)">
            <span class="font-medium text-gray-700 dark:text-gray-300">{{ $t('route.ruleItem.inbound') }}</span>
            <span class="ml-2 text-gray-900 dark:text-gray-100">{{ formatList(rule.inbound) }}</span>
          </div>
          <div v-if="hasValue(rule.protocol)">
            <span class="font-medium text-gray-700 dark:text-gray-300">{{ $t('route.ruleItem.protocol') }}</span>
            <span class="ml-2 text-gray-900 dark:text-gray-100">{{ formatList(rule.protocol) }}</span>
          </div>
          <div v-if="hasValue(rule.network)">
            <span class="font-medium text-gray-700 dark:text-gray-300">{{ $t('route.ruleItem.network') }}</span>
            <span class="ml-2 text-gray-900 dark:text-gray-100">{{ formatList(rule.network) }}</span>
          </div>
          <div v-if="hasValue(rule.domain)">
            <span class="font-medium text-gray-700 dark:text-gray-300">{{ $t('route.ruleItem.domain') }}</span>
            <span class="ml-2 text-gray-900 dark:text-gray-100">{{ formatList(rule.domain) }}</span>
          </div>
          <div v-if="hasValue(rule.domain_suffix)">
            <span class="font-medium text-gray-700 dark:text-gray-300">{{ $t('route.ruleItem.domainSuffix') }}</span>
            <span class="ml-2 text-gray-900 dark:text-gray-100">{{ formatList(rule.domain_suffix) }}</span>
          </div>
          <div v-if="rule.geosite && (Array.isArray(rule.geosite) ? rule.geosite.length : rule.geosite)">
            <span class="font-medium text-gray-700 dark:text-gray-300">{{ $t('route.ruleItem.geosite') }}</span>
            <span class="ml-2 text-gray-900 dark:text-gray-100">
              {{ Array.isArray(rule.geosite) ? rule.geosite.join(', ') : rule.geosite }}
            </span>
          </div>
          <div v-if="rule.geoip && (Array.isArray(rule.geoip) ? rule.geoip.length : rule.geoip)">
            <span class="font-medium text-gray-700 dark:text-gray-300">{{ $t('route.ruleItem.geoip') }}</span>
            <span class="ml-2 text-gray-900 dark:text-gray-100">
              {{ Array.isArray(rule.geoip) ? rule.geoip.join(', ') : rule.geoip }}
            </span>
          </div>
          <div v-if="rule.rule_set && (Array.isArray(rule.rule_set) ? rule.rule_set.length : rule.rule_set)">
            <span class="font-medium text-gray-700 dark:text-gray-300">{{ $t('route.ruleItem.ruleSet') }}</span>
            <span class="ml-2 text-gray-900 dark:text-gray-100">
              {{ Array.isArray(rule.rule_set) ? rule.rule_set.join(', ') : rule.rule_set }}
            </span>
          </div>
          <div v-if="hasValue(rule.port)">
            <span class="font-medium text-gray-700 dark:text-gray-300">{{ $t('route.ruleItem.port') }}</span>
            <span class="ml-2 text-gray-900 dark:text-gray-100">{{ formatList(rule.port) }}</span>
          </div>
          <div v-if="rule.sniffer">
            <span class="font-medium text-gray-700 dark:text-gray-300">{{ $t('route.ruleItem.sniffer') }}</span>
            <span class="ml-2 text-gray-900 dark:text-gray-100">
              {{ Array.isArray(rule.sniffer) ? rule.sniffer.join(', ') : rule.sniffer }}
            </span>
          </div>
          <div v-if="rule.timeout">
            <span class="font-medium text-gray-700 dark:text-gray-300">{{ $t('route.ruleItem.timeout') }}</span>
            <span class="ml-2 text-gray-900 dark:text-gray-100">{{ rule.timeout }}</span>
          </div>
        </div>
      </div>
      <div class="flex space-x-2 ml-4">
        <button
          @click="handleEdit"
          class="text-primary-600 hover:text-primary-800 dark:text-primary-400 dark:hover:text-primary-300"
        >
          {{ $t('common.edit') }}
        </button>
        <PopConfirm
          :message="$t('route.rules.confirm.delete')"
          :target="ruleSummary"
          :confirm-label="$t('common.delete')"
          tone="danger"
          align="right"
          trigger-class="text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300 focus:outline-none focus-visible:ring-2 focus-visible:ring-danger rounded-control"
          @confirm="handleDelete"
        >
          {{ $t('common.delete') }}
        </PopConfirm>
      </div>
    </div>
  </div>
</template>
