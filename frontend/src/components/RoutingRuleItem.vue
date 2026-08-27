<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Bars3Icon } from '@heroicons/vue/24/outline'
import type { RouteRule } from '../types/api'
import PopConfirm from './PopConfirm.vue'
import ListRow from './ListRow.vue'
import ListField from './ListField.vue'
import { ALL_MATCHER_KEYS } from '../schemas/routeRuleMatcherFields'
import { actionOf, resolveRouteRuleActionFields } from '../schemas/routeRuleActionFields'
import { isFieldFilled } from '../schemas/optionSchema'
import { humanizeFieldName } from '../utils/fieldLabels'

interface Props {
  rule: RouteRule
  index: number
  /** Reorder mode: show the drag handle, hide edit/delete. */
  reorderable?: boolean
  /**
   * Drag handlers for the row and for its handle, from `useDragReorder`. They
   * arrive as attribute bags rather than as emits so the mechanics live in one
   * place — the DNS rules table binds the same two objects to a `<tr>`.
   */
  rowAttrs?: Record<string, unknown>
  handleAttrs?: Record<string, unknown>
}

interface Emits {
  (e: 'edit', index: number, rule: RouteRule): void
  (e: 'delete', index: number): void
}

const props = withDefaults(defineProps<Props>(), {
  reorderable: false,
  rowAttrs: () => ({}),
  handleAttrs: () => ({}),
})
const emit = defineEmits<Emits>()

const { t, te } = useI18n()

function handleEdit() {
  emit('edit', props.index, props.rule)
}

function handleDelete() {
  emit('delete', props.index)
}

// Defensive renderers for the delete-confirmation summary below. sing-box can
// return a scalar ("inbound": "dns-in") or an array (["dns-in"]);
// RoutingRules.vue normalizes at the boundary, but these keep the item safe if
// any caller forgets.
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

/* ── Fields ─────────────────────────────────────────────────────────────────
 *
 * The row reads in the SAME ORDER as the flow preview inside the edit dialog:
 * every filled condition first, then what the rule does about them. The old
 * markup led with `action` + `outbound`, which is the order sing-box evaluates
 * a rule in reversed — matchers are tested first and the action only applies
 * once they all pass, and a list that opens with the outcome makes the two
 * halves of a rule read backwards between the list and the dialog.
 *
 * It is also derived from the same inventory the preview walks, rather than the
 * thirteen hand-listed fields it used to carry. Those thirteen covered 11 of
 * the 37 matchers and 2 of the ~45 action fields, so a rule matching on
 * `ip_cidr` (which this repo's own init wizard emits) or `source_ip_cidr`
 * displayed as if it had no conditions at all.
 */

/** Boolean matchers carry their statement in the label — see RuleFlowPreview. */
const BOOLEAN_MARK = '✓'

interface DisplayField {
  key: string
  label: string
  value: string
}

/**
 * `route.rules.fields.*`, humanized from the JSON key when untranslated — the
 * same resolution the schema-driven form and the flow preview use.
 *
 * The trailing colon is normalized on rather than baked into the locale,
 * because most of these labels are shared with the form, where a colon would be
 * wrong.
 */
function fieldLabel(key: string, labelKey = `route.rules.fields.${key}`): string {
  const raw = te(labelKey) ? t(labelKey) : humanizeFieldName(key)
  return `${raw.replace(/[:：]\s*$/, '')}:`
}

function displayValue(value: unknown): string {
  return typeof value === 'boolean' ? BOOLEAN_MARK : formatList(value)
}

const record = computed(() => props.rule as Record<string, unknown>)

/**
 * WHEN — every filled matcher, in inventory order.
 *
 * `invert` is pulled out of the chain for the same reason the preview does it:
 * it does not narrow the match, it flips the whole result.
 */
const conditions = computed<DisplayField[]>(() =>
  ALL_MATCHER_KEYS.filter((key) => key !== 'invert' && isFieldFilled(record.value[key])).map(
    (key) => ({
      key,
      label: fieldLabel(key),
      value: displayValue(record.value[key]),
    }),
  ),
)

const inverted = computed(() => isFieldFilled(record.value.invert))

/**
 * THEN — the fields the effective action owns, filled ones only.
 *
 * `actionOf` supplies the "route" default for a rule that omitted `action`, so
 * its `outbound` is still listed.
 */
const outcomeFields = computed<DisplayField[]>(() =>
  resolveRouteRuleActionFields(actionOf(record.value))
    .filter((field) => isFieldFilled(record.value[field.key]))
    .map((field) => ({
      key: field.key,
      label: fieldLabel(field.key, field.labelKey),
      value: displayValue(record.value[field.key]),
    })),
)

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
  const rule = record.value
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
  <ListRow v-bind="rowAttrs">
    <template v-if="reorderable" #leading>
      <button
        v-bind="handleAttrs"
        class="list-drag-handle"
        :aria-label="$t('route.rules.reorder.handle', { position: index + 1 })"
      >
        <Bars3Icon class="w-4 h-4" />
      </button>
    </template>

    <!--
      WHEN … THEN, the same order the dialog's RuleFlowPreview reads in.
      <ListField> hides anything empty and joins arrays itself; sing-box rules
      are sparse, so a typical row still renders two or three of these.
    -->
    <ListField
      v-if="inverted"
      :label="fieldLabel('invert')"
      :value="BOOLEAN_MARK"
    />
    <ListField
      v-for="condition in conditions"
      :key="`when-${condition.key}`"
      :label="condition.label"
      :value="condition.value"
    />

    <ListField :label="$t('route.ruleItem.action')" :value="rule.action" />
    <ListField
      v-for="field in outcomeFields"
      :key="`then-${field.key}`"
      :label="field.label"
      :value="field.value"
    />

    <!-- Reorder mode hides edit/delete on purpose: both are index-addressed,
         and the indices are exactly what is in motion. -->
    <template v-if="!reorderable" #actions>
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
