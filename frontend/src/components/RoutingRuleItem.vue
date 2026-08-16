<script setup lang="ts">
import { computed } from 'vue'
import type { RouteRule } from '../types/api'
import PopConfirm from './PopConfirm.vue'
import ListRow from './ListRow.vue'
import ListField from './ListField.vue'

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
  <ListRow>
    <!--
      Thirteen fields, each hidden by <ListField> when the rule does not set it.
      sing-box rules are sparse — a typical one populates two or three — and the
      old markup carried a `v-if="hasValue(…)"` plus a `formatList(…)` call on
      every single line. <ListField> hides empties and joins arrays itself.
    -->
    <ListField :label="$t('route.ruleItem.action')" :value="rule.action" />
    <ListField :label="$t('route.ruleItem.outbound')" :value="rule.outbound" />
    <ListField :label="$t('route.ruleItem.inbound')" :value="rule.inbound" />
    <ListField :label="$t('route.ruleItem.protocol')" :value="rule.protocol" />
    <ListField :label="$t('route.ruleItem.network')" :value="rule.network" />
    <ListField :label="$t('route.ruleItem.domain')" :value="rule.domain" />
    <ListField :label="$t('route.ruleItem.domainSuffix')" :value="rule.domain_suffix" />
    <ListField :label="$t('route.ruleItem.geosite')" :value="rule.geosite" />
    <ListField :label="$t('route.ruleItem.geoip')" :value="rule.geoip" />
    <ListField :label="$t('route.ruleItem.ruleSet')" :value="rule.rule_set" />
    <ListField :label="$t('route.ruleItem.port')" :value="rule.port" />
    <ListField :label="$t('route.ruleItem.sniffer')" :value="rule.sniffer" />
    <ListField :label="$t('route.ruleItem.timeout')" :value="rule.timeout" />

    <template #actions>
      <button
        @click="handleEdit"
        class="list-action-btn text-primary-600 hover:text-primary-800 dark:text-primary-400 dark:hover:text-primary-300"
      >
        {{ $t('common.edit') }}
      </button>
      <PopConfirm
        :message="$t('route.rules.confirm.delete')"
        :target="ruleSummary"
        :confirm-label="$t('common.delete')"
        tone="danger"
        align="right"
        trigger-class="list-action-btn text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300 focus:outline-none focus-visible:ring-2 focus-visible:ring-danger"
        @confirm="handleDelete"
      >
        {{ $t('common.delete') }}
      </PopConfirm>
    </template>
  </ListRow>
</template>
