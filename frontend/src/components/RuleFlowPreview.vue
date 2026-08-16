<script setup lang="ts">
/**
 * Reads a route rule back as the sentence sing-box will execute.
 *
 * The form shows which fields exist; it never said what they DO once a
 * connection arrives. This is that missing half — a live restatement of the rule
 * as `WHEN <conditions> THEN <action>`, updating as the operator edits.
 *
 * IT SAYS "AND", AND THAT IS THE POINT
 * ────────────────────────────────────
 * The intuition most people bring is "match the rule set OR the domain". The
 * runtime disagrees (route/rule/rule_abstract.go:109-115):
 *
 *   for _, item := range r.items {
 *       if !item.Match(metadata) { return r.invert }
 *   }
 *
 * Every matcher must match. The OR lives one level down, INSIDE a single field:
 * `domain: [a, b]` matches a or b, because that one item iterates its own values
 * and breaks on the first hit. So the preview renders AND between fields and
 * "any of" within one — which is also exactly what the existing mix warning in
 * RouteRuleMatchers.vue has been trying to say in prose.
 *
 * To get a real OR across different conditions you need two rules, or a logical
 * rule with `mode: or`.
 *
 * THE EMPTY CASE IS NOT "MATCHES NOTHING"
 * ───────────────────────────────────────
 * `rule_abstract.go:55` — a rule with no items returns true for everything. For
 * a terminal action that swallows all remaining traffic and kills every rule
 * below it, so it is called out rather than rendered as a blank. For a
 * non-terminal action it is ordinary: `{"action": "sniff"}` with no conditions
 * means "sniff everything", and this repo's own config has exactly that.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowRightIcon, ExclamationTriangleIcon } from '@heroicons/vue/24/outline'
import { humanizeFieldName } from '../utils/fieldLabels'
import { actionOf, isTerminalAction } from '../schemas/routeRuleActionFields'
import { ALL_MATCHER_KEYS } from '../schemas/routeRuleMatcherFields'
import { isFieldFilled } from '../schemas/optionSchema'
import type { RouteRule } from '../types/api'

const props = defineProps<{ rule: RouteRule }>()

const { t, te } = useI18n()

const record = computed(() => props.rule as Record<string, unknown>)
const action = computed(() => actionOf(record.value))

/** How many values of one matcher to spell out before collapsing to a count. */
const MAX_VALUES = 3

function label(key: string): string {
  const labelKey = `route.rules.fields.${key}`
  return te(labelKey) ? t(labelKey) : humanizeFieldName(key)
}

interface Condition {
  key: string
  label: string
  /** Spelled-out values; empty for a boolean matcher, which the label carries. */
  values: string[]
  /** Values beyond MAX_VALUES, collapsed to a count. */
  overflow: number
}

/**
 * One entry per FILLED matcher, in the rule's own key order.
 *
 * `invert` is excluded on purpose — it does not narrow the match, it flips the
 * whole result, so it belongs with the verb rather than in the AND chain.
 */
const conditions = computed<Condition[]>(() => {
  const out: Condition[] = []

  for (const key of ALL_MATCHER_KEYS) {
    if (key === 'invert') continue
    const value = record.value[key]
    if (!isFieldFilled(value)) continue

    // A boolean matcher is its own statement ("Destination Is Private"); there
    // is no value worth printing next to it.
    if (typeof value === 'boolean') {
      out.push({ key, label: label(key), values: [], overflow: 0 })
      continue
    }

    const all = (Array.isArray(value) ? value : [value]).map(String)
    out.push({
      key,
      label: label(key),
      values: all.slice(0, MAX_VALUES),
      overflow: Math.max(0, all.length - MAX_VALUES),
    })
  }

  return out
})

const isInverted = computed(() => isFieldFilled(record.value.invert))

/** No conditions at all — see the header note. */
const matchesEverything = computed(() => conditions.value.length === 0)

/** A condition-less TERMINAL rule swallows everything below it. */
const isCatchAll = computed(() => matchesEverything.value && isTerminalAction(action.value))

const actionLabelKey = computed(() => `route.rules.flow.then.${camel(action.value)}`)

function camel(value: string) {
  return value.replace(/-([a-z])/g, (_, c: string) => c.toUpperCase())
}

/**
 * The outcome, as a phrase. Each action names the field that carries its
 * result, so "route" reads as the outbound rather than the word "route".
 */
const outcome = computed(() => {
  const r = record.value
  switch (action.value) {
    case 'route':
      return isFieldFilled(r.outbound)
        ? t('route.rules.flow.then.route', { outbound: String(r.outbound) })
        : t('route.rules.flow.then.routeIncomplete')
    case 'reject':
      return r.method === 'drop'
        ? t('route.rules.flow.then.rejectDrop')
        : t('route.rules.flow.then.reject')
    case 'resolve':
      return isFieldFilled(r.server)
        ? t('route.rules.flow.then.resolveWith', { server: String(r.server) })
        : t('route.rules.flow.then.resolve')
    default:
      return te(actionLabelKey.value) ? t(actionLabelKey.value) : action.value
  }
})

/** Non-terminal actions keep matching, which is the thing people miss. */
const continuesMatching = computed(() => !isTerminalAction(action.value))
</script>

<template>
  <div
    class="rounded-surface border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50 p-3 space-y-2"
  >
    <!-- WHEN -->
    <div class="flex gap-2 items-baseline flex-wrap">
      <span
        class="shrink-0 text-[11px] font-semibold uppercase tracking-wide text-gray-400 dark:text-gray-500 w-10"
      >
        {{ t('route.rules.flow.when') }}
      </span>

      <span v-if="matchesEverything" class="text-sm text-gray-700 dark:text-gray-300">
        {{ t('route.rules.flow.everything') }}
      </span>

      <template v-else>
        <span v-if="isInverted" class="text-sm font-medium text-amber-600 dark:text-amber-400">
          {{ t('route.rules.flow.invertPrefix') }}
        </span>
        <template v-for="(condition, i) in conditions" :key="condition.key">
          <!-- The AND is the correction this whole panel exists to make. -->
          <span
            v-if="i > 0"
            class="text-[11px] font-semibold uppercase tracking-wide text-gray-400 dark:text-gray-500"
          >
            {{ t('route.rules.flow.and') }}
          </span>
          <span class="text-sm text-gray-700 dark:text-gray-300">
            <span class="text-gray-500 dark:text-gray-400">{{ condition.label }}</span>
            <template v-if="condition.values.length">
              <span class="text-gray-400 dark:text-gray-500">
                {{ condition.values.length > 1 ? ` ${t('route.rules.flow.anyOf')} ` : ' ' }}
              </span>
              <code
                v-for="(value, vi) in condition.values"
                :key="value"
                class="font-mono text-xs text-gray-800 dark:text-gray-200"
                >{{ value }}<span v-if="vi < condition.values.length - 1">, </span></code
              >
              <span v-if="condition.overflow" class="text-gray-400 dark:text-gray-500">
                {{ t('route.rules.flow.more', { count: condition.overflow }) }}
              </span>
            </template>
          </span>
        </template>
      </template>
    </div>

    <!-- THEN -->
    <div class="flex gap-2 items-baseline flex-wrap">
      <span
        class="shrink-0 text-[11px] font-semibold uppercase tracking-wide text-gray-400 dark:text-gray-500 w-10"
      >
        {{ t('route.rules.flow.then.label') }}
      </span>
      <span class="inline-flex items-center gap-1.5 text-sm text-gray-800 dark:text-gray-200">
        <ArrowRightIcon class="h-3.5 w-3.5 shrink-0 text-primary-500" />
        {{ outcome }}
      </span>
      <!--
        Stated because it is the least obvious thing about a route rule: only
        route/reject/hijack-dns stop here. The others annotate the connection and
        the next rule is still tried.
      -->
      <span v-if="continuesMatching" class="text-xs text-gray-400 dark:text-gray-500">
        {{ t('route.rules.flow.continues') }}
      </span>
    </div>

    <!--
      A condition-less terminal rule matches every connection and makes every
      rule below it unreachable. Not shown for sniff/resolve/route-options, where
      "no conditions" is the normal way to say "all traffic".
    -->
    <p
      v-if="isCatchAll"
      class="flex items-start gap-1.5 text-xs text-amber-600 dark:text-amber-400 pt-1 border-t border-gray-200 dark:border-gray-700"
    >
      <ExclamationTriangleIcon class="h-3.5 w-3.5 shrink-0 mt-0.5" />
      <span>{{ t('route.rules.flow.catchAll') }}</span>
    </p>
  </div>
</template>
