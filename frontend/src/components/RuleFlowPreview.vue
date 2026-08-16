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
import { isFieldFilled } from '../schemas/optionSchema'

const props = defineProps<{
  /** The rule being edited, whole. */
  rule: Record<string, unknown>
  /**
   * Which keys count as conditions, in render order. Supplied by the caller
   * because the two rule families disagree: a route rule has a generated
   * matcher inventory to order by, a DNS rule's conditions are whatever is left
   * once the action's own fields are removed.
   */
  conditionKeys: readonly string[]
  /** i18n stem for condition labels, e.g. `route.rules.fields`. */
  labelPrefix: string
  /** The outcome as a resolved phrase — each domain names its own actions. */
  outcome: string
  /** Non-terminal actions annotate and let the next rule be tried. */
  continuesMatching?: boolean
  /** A condition-less TERMINAL rule swallows everything below it. */
  catchAll?: boolean
}>()

const { t, te } = useI18n()

const record = computed(() => props.rule)

/** How many values of one matcher to spell out before collapsing to a count. */
const MAX_VALUES = 3

function label(key: string): string {
  const labelKey = `${props.labelPrefix}.${key}`
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

  for (const key of props.conditionKeys) {
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
        {{ t('rule.flow.when') }}
      </span>

      <span v-if="matchesEverything" class="text-sm text-gray-700 dark:text-gray-300">
        {{ t('rule.flow.everything') }}
      </span>

      <template v-else>
        <span v-if="isInverted" class="text-sm font-medium text-amber-600 dark:text-amber-400">
          {{ t('rule.flow.invertPrefix') }}
        </span>
        <template v-for="(condition, i) in conditions" :key="condition.key">
          <!-- The AND is the correction this whole panel exists to make. -->
          <span
            v-if="i > 0"
            class="text-[11px] font-semibold uppercase tracking-wide text-gray-400 dark:text-gray-500"
          >
            {{ t('rule.flow.and') }}
          </span>
          <span class="text-sm text-gray-700 dark:text-gray-300">
            <span class="text-gray-500 dark:text-gray-400">{{ condition.label }}</span>
            <template v-if="condition.values.length">
              <span class="text-gray-400 dark:text-gray-500">
                {{ condition.values.length > 1 ? ` ${t('rule.flow.anyOf')} ` : ' ' }}
              </span>
              <code
                v-for="(value, vi) in condition.values"
                :key="value"
                class="font-mono text-xs text-gray-800 dark:text-gray-200"
                >{{ value }}<span v-if="vi < condition.values.length - 1">, </span></code
              >
              <span v-if="condition.overflow" class="text-gray-400 dark:text-gray-500">
                {{ t('rule.flow.more', { count: condition.overflow }) }}
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
        {{ t('rule.flow.then.label') }}
      </span>
      <span class="inline-flex items-center gap-1.5 text-sm text-gray-800 dark:text-gray-200">
        <ArrowRightIcon class="h-3.5 w-3.5 shrink-0 text-primary-500" />
        {{ props.outcome }}
      </span>
      <!--
        Stated because it is the least obvious thing about a route rule: only
        route/reject/hijack-dns stop here. The others annotate the connection and
        the next rule is still tried.
      -->
      <span v-if="props.continuesMatching" class="text-xs text-gray-400 dark:text-gray-500">
        {{ t('rule.flow.continues') }}
      </span>
    </div>

    <!--
      A condition-less terminal rule matches every connection and makes every
      rule below it unreachable. Not shown for sniff/resolve/route-options, where
      "no conditions" is the normal way to say "all traffic".
    -->
    <p
      v-if="catchAll && matchesEverything"
      class="flex items-start gap-1.5 text-xs text-amber-600 dark:text-amber-400 pt-1 border-t border-gray-200 dark:border-gray-700"
    >
      <ExclamationTriangleIcon class="h-3.5 w-3.5 shrink-0 mt-0.5" />
      <span>{{ t('rule.flow.catchAll') }}</span>
    </p>
  </div>
</template>
