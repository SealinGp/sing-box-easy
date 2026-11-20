<script setup lang="ts">
import IconButton from './IconButton.vue'

import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowRightIcon, ChevronDownIcon, Bars3Icon } from '@heroicons/vue/24/outline'
import type { RouteRule } from '../types/api'
import PopConfirm from './PopConfirm.vue'
import ListRow from './ListRow.vue'
import ListField from './ListField.vue'
import RoutingRuleConditions from './RoutingRuleConditions.vue'
import { RouterLink } from 'vue-router'
import { FOCUS_PAGES, focusLink } from '../utils/focusTarget'
import { routeRuleOutcome } from '../utils/routeRulePresentation'
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
const expanded = ref(false)

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

const outcome = computed(() => routeRuleOutcome(record.value, t))
const action = computed(() => actionOf(record.value))
const compactOutcome = computed(() => action.value === 'hijack-dns'
  ? t('route.rules.actions.hijackDns') : outcome.value)
const continues = computed(() => ['sniff', 'resolve', 'route-options'].includes(action.value))
const options = computed(() => outcomeFields.value.filter(field => field.key !== 'outbound'))

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
  'source_mac_address',
  'source_hostname',
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
    <template #leading>
      <span class="inline-flex min-w-8 justify-center text-sm font-semibold tabular-nums text-gray-500 dark:text-gray-400" :aria-label="$t('route.rules.priority.position', { position: index + 1 })" :title="$t('route.rules.priority.position', { position: index + 1 })">{{ index + 1 }}</span>
      <button
        v-if="reorderable"
        v-bind="handleAttrs"
        class="list-drag-handle"
        :aria-label="$t('route.rules.reorder.handle', { position: index + 1 })"
      >
        <Bars3Icon class="w-4 h-4" />
      </button>
    </template>

    <div class="sentence-shell col-span-2 min-w-0">
      <details class="rule-details" :open="expanded" @toggle="expanded = ($event.target as HTMLDetailsElement).open">
        <summary class="rule-summary rounded-control focus-visible:outline-2 focus-visible:outline-primary-500">
          <span class="condition-summary min-w-0 truncate text-sm" :aria-label="$t('route.rules.sentence.conditions')">
            <RoutingRuleConditions :rule="record" compact />
          </span>
          <span class="outcome-summary flex min-w-0 items-center gap-2 text-sm font-medium text-text-primary" :aria-label="$t('route.rules.sentence.action')">
            <ArrowRightIcon class="h-3.5 w-3.5 shrink-0 text-text-secondary" aria-hidden="true" />
            <template v-if="action === 'route' && rule.outbound">
              <RouterLink :to="focusLink(FOCUS_PAGES.proxies, rule.outbound)"
                :aria-label="$t('route.rules.sentence.openProxy', { outbound: rule.outbound })"
                class="min-w-0 truncate rounded-control text-primary-600 underline decoration-primary-300 underline-offset-2 focus-visible:outline-2 focus-visible:outline-primary-500 dark:text-primary-400"
                @click.stop>
                {{ $t('route.rules.sentence.via') }} {{ rule.outbound }}
              </RouterLink>
            </template>
            <span v-else class="truncate">{{ compactOutcome }}</span>
          </span>
          <span class="expand-control flex items-center justify-end gap-1 text-[11px] text-text-secondary">
            {{ $t(expanded ? 'route.rules.sentence.collapse' : 'route.rules.sentence.expand') }}
            <ChevronDownIcon class="h-3.5 w-3.5 shrink-0" :class="{ 'rotate-180': expanded }" aria-hidden="true" />
          </span>
        </summary>
        <div class="mt-2 space-y-3 border-t border-border py-3">
          <RoutingRuleConditions :rule="record" />
          <div class="flex min-w-0 flex-wrap items-center gap-2 text-sm font-medium text-text-primary">
            <ArrowRightIcon class="h-3.5 w-3.5 shrink-0 text-text-secondary" aria-hidden="true" />
            <RouterLink v-if="action === 'route' && rule.outbound" :to="focusLink(FOCUS_PAGES.proxies, rule.outbound)"
              class="break-all rounded-control text-primary-600 underline focus-visible:outline-2 focus-visible:outline-primary-500 dark:text-primary-400">
              {{ $t('route.rules.sentence.via') }} {{ rule.outbound }}
            </RouterLink>
            <span v-else class="break-all">{{ outcome }}</span>
          </div>
          <div v-if="options.length" class="space-y-1 text-xs">
            <p class="font-medium text-text-secondary">{{ $t('route.rules.sentence.options', { count: options.length }) }}</p>
            <div class="space-y-1 break-all">
              <ListField v-for="field in options" :key="field.key" :label="field.label" :value="field.value" />
            </div>
          </div>
          <p class="text-[11px] text-text-secondary">{{ $t(continues ? 'route.rules.sentence.continue' : 'route.rules.sentence.stop') }}</p>
        </div>
      </details>
    </div>

    <!-- Reorder mode hides edit/delete on purpose: both are index-addressed,
         and the indices are exactly what is in motion. -->
    <template v-if="!reorderable" #actions>
      <IconButton
        action="edit"
        @click="handleEdit"
      />
      <PopConfirm
        :message="$t('route.rules.confirm.delete')"
        :target="ruleSummary"
        :confirm-label="$t('common.delete')"
        tone="danger"
        align="right"
        @confirm="handleDelete"
      />
    </template>
  </ListRow>
</template>

<style scoped>
.sentence-shell { container-type: inline-size; }
.rule-summary {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 12rem 3.5rem;
  align-items: center;
  gap: 1rem;
  min-height: 1.75rem;
  cursor: pointer;
  list-style: none;
}
.rule-summary::-webkit-details-marker { display: none; }
@container (max-width: 600px) {
  .rule-summary { grid-template-columns: minmax(0, 1fr) 3.5rem; gap: .375rem .75rem; }
  .condition-summary { grid-column: 1; }
  .outcome-summary { grid-column: 1; grid-row: 2; }
  .expand-control { grid-column: 2; grid-row: 1 / 3; }
}
</style>
