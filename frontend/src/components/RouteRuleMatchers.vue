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
 * GEOSITE AND GEOIP ARE GONE
 * ──────────────────────────
 * They had curated dropdowns with 12 and 7 options and were the primary content
 * matchers. sing-box REMOVED them in 1.12.0 and answers with a hard startup
 * error, so all 19 values produced a rule that could not run. The version gate
 * now withholds them — while still rendering one that a loaded config already
 * uses, so it can be seen and cleared. The replacement is a rule set, which
 * this repo's templates and init wizard already use.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import SchemaFieldsEditor from './SchemaFieldsEditor.vue'
import Alert from './Alert.vue'
import { PlusCircleIcon } from '@heroicons/vue/24/outline'
import { useExclusiveMatcherGroups } from '../composables/useMatcherFields'
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
// Derived from the curation rather than a second hand-written list, so the
// warning cannot drift from what the content section actually renders.
const hasMatchers = computed(() => CONTENT_MATCHER_KEYS.some(isFilled))

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
          {{ t('route.rules.mixing.ruleSetGroup') }}
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
      <SchemaFieldsEditor v-model="record" :fields="ruleSetFields" />
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
      <div class="flex items-center justify-between">
        <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('route.rules.mixing.matchersGroup') }}
        </span>
        <button
          v-if="canHideMatchers"
          type="button"
          class="text-xs text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 transition-colors"
          @click="collapseMatchers"
        >
          {{ t('route.rules.mixing.hide') }}
        </button>
      </div>

      <SchemaFieldsEditor v-model="record" :fields="contentFields" />
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
    <div class="border-t border-gray-200 dark:border-gray-700 pt-4">
      <div class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
        {{ t('route.rules.mixing.contextGroup') }}
      </div>
      <SchemaFieldsEditor v-model="record" :fields="contextFields" />
    </div>
  </div>
</template>
