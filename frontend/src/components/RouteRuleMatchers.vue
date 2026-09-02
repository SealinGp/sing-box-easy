<script setup lang="ts">
/**
 * Reusable editor for a route rule's *matching criteria*. Self-contained: pass
 * any RouteRule via v-model and render it anywhere (route rules dialog, init
 * wizard, …).
 *
 * Schema-driven since the route rule work: the field list comes from
 * `schemas/routeRuleMatcherFields.ts`, which curates the generated inventory of
 * `option.RawDefaultRule`. The hand-written version rendered 9 of the 37
 * matchers sing-box accepts, and two of those nine were dead (see below).
 *
 * WHAT DID NOT CHANGE
 * ───────────────────
 * The grouping model, which predates the schema and is the load-bearing
 * decision here. It now lives in the curation file as a `group` per field, but
 * the reasoning is unchanged:
 *
 *   CONTENT matchers describe *what* the traffic is. A rule set already
 *   expresses exactly these, so combining the two in one rule is the AND trap —
 *   "in the rule set AND also this domain" — and they are therefore presented
 *   as alternatives to `rule_set`.
 *
 *   CONTEXT matchers describe *where the traffic came from*. A rule set cannot
 *   express them, so narrowing a rule set by `network: udp` or `port: 443` is a
 *   normal, correct thing to do. These are never folded away by the rule-set
 *   choice — only by their own emptiness.
 *
 * Collapsing every matcher against `rule_set` would have been the simpler code
 * and the wrong model: it would flag the most common correct route rule in
 * sing-box (a rule set narrowed by port or network) as a mistake.
 *
 * WHY THREE EDITORS AND NOT ONE
 * ─────────────────────────────
 * `SchemaFieldsEditor` takes an already-resolved field list, so grouping is just
 * filtering — three instances bound to the same rule, each handed its own
 * subset. That keeps the exclusivity logic *here*, next to the comment
 * explaining it, instead of generalising it into machinery with one user. The
 * editor itself needed no change at all.
 *
 * THE CHOICE IS A RADIO, AND IT COVERS TWO GROUPS, NOT THREE
 * ──────────────────────────────────────────────────────────
 * Rule set vs content is now picked explicitly (`useMatchStyle`) and only the
 * chosen editor renders. It used to be inferred from the rule's contents, with
 * the other style folded behind a "show anyway" button — which read as the form
 * deciding, and hid the existence of the alternative behind two clicks.
 *
 * Context is deliberately NOT a third radio option. sing-box ANDs every matcher
 * in a rule and a rule set cannot express where traffic came from, so
 * `rule_set: geosite-cn` narrowed by `network: udp` is an ordinary correct rule;
 * a three-way radio would make it unwritable in the form. The two things that
 * are alternatives get the radio, the thing that composes stays outside it.
 *
 * GEOSITE AND GEOIP ARE GONE
 * ──────────────────────────
 * They had curated dropdowns with 12 and 7 options and were the primary content
 * matchers. sing-box REMOVED them in 1.12.0 and answers with a hard startup
 * error, so all 19 values produced a rule that could not run. The version gate
 * now withholds them — while still rendering one that a loaded config already
 * uses, so it can be seen and cleared. The replacement is a rule set, which
 * this repo's templates and init wizard already use.
 */
import { computed, useId } from 'vue'
import { useI18n } from 'vue-i18n'
import SchemaFieldsEditor from './SchemaFieldsEditor.vue'
import Alert from './Alert.vue'
import { useMatchStyle, type MatchStyle } from '../composables/useMatcherFields'
import { isFieldFilled } from '../schemas/optionSchema'
import {
  CONTENT_MATCHER_KEYS,
  resolveMatcherFields,
} from '../schemas/routeRuleMatcherFields'
import type { RouteRule } from '../types/api'

const model = defineModel<RouteRule>({ required: true })
const { t } = useI18n()

// Resolved once: the field list is a pure function of the curation, and this
// domain has a single type, so it cannot go stale.
const ruleSetFields = resolveMatcherFields('ruleSet')
const contentFields = resolveMatcherFields('content')
const contextFields = resolveMatcherFields('context')

/**
 * The rule as a plain record, for the editors.
 *
 * Each editor replaces the whole object with an immutable spread touching only
 * the keys it renders, so all three plus the parent's action editor compose
 * without stepping on each other.
 */
const record = computed<Record<string, unknown>>({
  get: () => model.value as Record<string, unknown>,
  set: (v) => {
    model.value = v as RouteRule
  },
})

const isFilled = (key: string) => isFieldFilled((model.value as Record<string, unknown>)[key])

const hasRuleSet = computed(() => isFilled('rule_set'))

/**
 * Whether each section currently narrows the match.
 *
 * Drives the AND connectors: a connector between two sections is only honest
 * when both sides actually carry a condition. Showing "AND" above an empty
 * context section would claim a constraint that is not there.
 */
const contextKeys = contextFields.map((f) => f.key)
const hasContext = computed(() => contextKeys.some(isFilled))
// Derived from the curation rather than a second hand-written list, so the
// warning cannot drift from what the content section actually renders.
const hasMatchers = computed(() => CONTENT_MATCHER_KEYS.some(isFilled))

const { style, select, strandedRuleSet, strandedContent } = useMatchStyle({
  hasRuleSet,
  hasMatchers,
})

const styleOptions: { value: MatchStyle; label: string; hint: string }[] = [
  {
    value: 'ruleSet',
    label: 'route.rules.mixing.ruleSetGroup',
    hint: 'route.rules.mixing.ruleSetStyleHint',
  },
  {
    value: 'content',
    label: 'route.rules.mixing.matchersGroup',
    hint: 'route.rules.mixing.matchersStyleHint',
  },
]

/**
 * Radio groups are keyed by `name`, so two of these on one page (an add dialog
 * left in the DOM behind an edit dialog) would share a selection. Unique per
 * instance.
 */
const groupName = `match-style-${useId()}`

/**
 * Clearing a stranded group.
 *
 * The only destructive control here, and it is explicit: the alert says which
 * values are still in force, and this removes exactly those keys. Switching the
 * radio never clears anything on its own — a mistyped click would otherwise
 * silently drop a list of rule sets.
 */
function clearKeys(keys: readonly string[]) {
  const next = { ...(model.value as Record<string, unknown>) }
  for (const key of keys) delete next[key]
  model.value = next as RouteRule
}

const clearRuleSet = () => clearKeys(ruleSetFields.map((f) => f.key))
const clearContent = () => clearKeys(CONTENT_MATCHER_KEYS)
</script>

<template>
  <div class="space-y-4">
    <!-- ── The either-or, stated as a choice ──────────────────────────────
         A rule set already expresses what the content matchers express, and
         sing-box ANDs them, so "rule set AND domain" means "this domain, but
         only if it is also in the set" — almost never the intent. Presenting
         them as two radio options makes the exclusivity the default reading
         instead of something the form has to warn about afterwards. -->
    <fieldset>
      <legend class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
        {{ t('route.rules.mixing.styleLabel') }}
      </legend>
      <div class="grid gap-2 sm:grid-cols-2">
        <label
          v-for="option in styleOptions"
          :key="option.value"
          class="flex items-start gap-2.5 p-3 rounded-control border cursor-pointer transition-colors"
          :class="
            style === option.value
              ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20 dark:border-primary-500'
              : 'border-gray-200 dark:border-gray-700 hover:border-primary-300 dark:hover:border-primary-600'
          "
        >
          <input
            type="radio"
            :name="groupName"
            :value="option.value"
            :checked="style === option.value"
            class="mt-0.5 w-4 h-4 shrink-0 text-primary-600 border-gray-300 focus:ring-primary-500"
            @change="select(option.value)"
          />
          <span class="min-w-0">
            <span class="block text-sm font-medium text-gray-900 dark:text-gray-100">
              {{ t(option.label) }}
            </span>
            <span class="block text-xs text-gray-500 dark:text-gray-400 mt-0.5">
              {{ t(option.hint) }}
            </span>
          </span>
        </label>
      </div>
    </fieldset>

    <!-- ── Stranded values ──────────────────────────────────────────────────
         The unselected style can still hold values: a config loaded from disk
         may legitimately mix, and switching the radio mid-edit strands whatever
         was already typed. Those values still save, so hiding them would make
         the form lie about the rule. Surfaced with the AND consequence spelled
         out and a way to act on it — never cleared behind the operator's back. -->
    <Alert v-if="strandedRuleSet" type="warning" :title="t('route.rules.mixing.title')">
      <p>{{ t('route.rules.mixing.strandedRuleSet') }}</p>
      <div class="mt-2 flex gap-3">
        <button type="button" class="text-xs font-medium underline" @click="select('ruleSet')">
          {{ t('route.rules.mixing.showStranded') }}
        </button>
        <button type="button" class="text-xs font-medium underline" @click="clearRuleSet">
          {{ t('route.rules.mixing.clearStranded') }}
        </button>
      </div>
    </Alert>

    <Alert v-if="strandedContent" type="warning" :title="t('route.rules.mixing.title')">
      <p>{{ t('route.rules.mixing.strandedContent') }}</p>
      <div class="mt-2 flex gap-3">
        <button type="button" class="text-xs font-medium underline" @click="select('content')">
          {{ t('route.rules.mixing.showStranded') }}
        </button>
        <button type="button" class="text-xs font-medium underline" @click="clearContent">
          {{ t('route.rules.mixing.clearStranded') }}
        </button>
      </div>
    </Alert>

    <!-- Only the chosen style renders. -->
    <SchemaFieldsEditor v-if="style === 'ruleSet'" v-model="record" :fields="ruleSetFields" />
    <SchemaFieldsEditor v-else v-model="record" :fields="contentFields" />

    <!-- ── Context conditions ───────────────────────────────────────────────
         Never folded by the rule-set choice: a rule set cannot express where
         traffic came from, so narrowing one by network/port/inbound is correct,
         not an accidental intersection. -->
    <div class="border-t border-gray-200 dark:border-gray-700 pt-4">
      <div class="flex items-baseline gap-2 mb-1">
        <span
          v-if="hasContext && (hasRuleSet || hasMatchers)"
          class="text-[11px] font-semibold uppercase tracking-wide text-gray-400 dark:text-gray-500"
        >
          {{ t('route.rules.flow.and') }}
        </span>
        <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('route.rules.mixing.contextGroup') }}
        </span>
      </div>
      <p class="text-xs text-gray-500 dark:text-gray-400 mb-3">
        {{ t('route.rules.flow.contextHint') }}
      </p>
      <SchemaFieldsEditor v-model="record" :fields="contextFields" />
    </div>
  </div>
</template>
