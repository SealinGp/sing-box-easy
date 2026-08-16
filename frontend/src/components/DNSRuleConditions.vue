<script setup lang="ts">
/**
 * The "conditions" half of the DNS rule editor.
 *
 * The two behaviours here — the rule-set/domain-matcher either-or, and the
 * show-only-what-is-filled field list — are shared with the route rule form and
 * live in `composables/useMatcherFields.ts`, which carries the reasoning. This
 * component supplies the DNS-specific field set and labels.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { MultiSelect } from '../volt'
import ChipsField from './ChipsField.vue'
import Alert from './Alert.vue'
import { PlusCircleIcon } from '@heroicons/vue/24/outline'
import { useExclusiveMatcherGroups, useOptionalFields } from '../composables/useMatcherFields'

/**
 * `rule_set` is a list on the wire — sing-box types it as
 * `badoption.Listable[string]`, and `types/dns.ts` already declares
 * `rule_set?: string[]`. It was edited here as a single Select, and DNSRules
 * bridged the gap by taking the first element on load and re-wrapping it on
 * save. That silently discarded data: a rule matching two rule sets opened
 * showing one, and saving wrote back only that one.
 *
 * Now a MultiSelect, matching RouteRuleMatchers.
 */
const ruleSet = defineModel<string[]>('ruleSet', { required: true })
const domain = defineModel<string[]>('domain', { required: true })
const domainSuffix = defineModel<string[]>('domainSuffix', { required: true })
const domainKeyword = defineModel<string[]>('domainKeyword', { required: true })
const geosite = defineModel<string[]>('geosite', { required: true })

const props = defineProps<{
  ruleSetOptions: { value: string; label: string }[]
}>()

const { t } = useI18n()

/**
 * A rule can name a rule set that was since renamed or deleted. MultiSelect
 * renders nothing for a value it has no option for, so that tag would vanish
 * from the form while still living in the rule — and the next save would look
 * like the operator had removed it. Surface it as a flagged option instead, so
 * it stays visible, stays selected, and can be removed deliberately.
 */
const selectableRuleSets = computed(() => {
  const options = [...props.ruleSetOptions]
  const known = new Set(options.map((option) => option.value))

  for (const tag of ruleSet.value ?? []) {
    if (known.has(tag)) continue
    known.add(tag)
    options.push({ value: tag, label: t('dns.rules.form.ruleSetMissing', { tag }) })
  }

  return options
})

type MatcherKey = 'domain' | 'domainSuffix' | 'domainKeyword' | 'geosite'

const matcherModels: Record<MatcherKey, { value: string[] }> = {
  domain,
  domainSuffix,
  domainKeyword,
  geosite,
}

// Labels live here so the "add" buttons and the field headers can never drift.
const matcherLabelKeys: Record<MatcherKey, string> = {
  domain: 'dns.rules.form.domain',
  domainSuffix: 'dns.rules.form.domainSuffix',
  domainKeyword: 'dns.rules.form.domainKeyword',
  geosite: 'dns.rules.form.geosite',
}

const matcherKeys = Object.keys(matcherLabelKeys) as MatcherKey[]

const {
  isShown: isFieldShown,
  isRemovable: isFieldRemovable,
  hidden: hiddenFields,
  add: addField,
  remove: removeField,
} = useOptionalFields(matcherKeys, (key) => matcherModels[key].value.length > 0)

const hasRuleSet = computed(() => (ruleSet.value?.length ?? 0) > 0)
const hasMatchers = computed(() =>
  matcherKeys.some((key) => matcherModels[key].value.length > 0),
)

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
  <div class="space-y-3">
    <Alert v-if="showMixWarning" type="warning" :title="$t('dns.rules.form.mixing.title')">
      {{ $t('dns.rules.form.mixing.warning') }}
    </Alert>

    <!-- Rule set -->
    <div v-if="showRuleSet">
      <div class="flex items-center justify-between mb-1">
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ $t('dns.rules.form.ruleSet') }}
        </label>
        <button
          v-if="canHideRuleSet"
          type="button"
          class="text-xs text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 transition-colors"
          @click="collapseRuleSet"
        >
          {{ $t('dns.rules.form.mixing.hide') }}
        </button>
      </div>
      <MultiSelect
        class="w-full"
        optionLabel="label"
        optionValue="value"
        v-model="ruleSet"
        :options="selectableRuleSets"
        display="chip"
        filter
        :placeholder="$t('dns.rules.form.ruleSetSelect')"
        :filterPlaceholder="$t('dns.rules.form.ruleSetSearch')"
        :emptyFilterMessage="$t('dns.rules.form.ruleSetNoOptions')"
        :emptyMessage="$t('dns.rules.form.ruleSetEmpty')"
      />
      <p class="mt-1 text-xs text-gray-500">{{ $t('dns.rules.form.ruleSetHelp') }}</p>
    </div>

    <button
      v-else
      type="button"
      class="w-full flex items-center gap-2 px-3 py-2 rounded-control border border-dashed border-gray-300 dark:border-gray-600 text-left text-xs text-gray-500 dark:text-gray-400 hover:border-primary-400 hover:text-primary-600 dark:hover:text-primary-300 transition-colors"
      @click="expandRuleSet"
    >
      <PlusCircleIcon class="h-4 w-4 shrink-0" />
      <span class="flex-1">{{ $t('dns.rules.form.mixing.ruleSetCollapsed') }}</span>
      <span class="font-medium">{{ $t('dns.rules.form.mixing.show') }}</span>
    </button>

    <!-- Domain conditions -->
    <template v-if="showMatchers">
      <div v-if="canHideMatchers" class="flex items-center justify-between">
        <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ $t('dns.rules.form.mixing.matchersGroup') }}
        </span>
        <button
          type="button"
          class="text-xs text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 transition-colors"
          @click="collapseMatchers"
        >
          {{ $t('dns.rules.form.mixing.hide') }}
        </button>
      </div>

      <ChipsField
        v-if="isFieldShown('domain')"
        v-model="domain"
        :label="$t('dns.rules.form.domain')"
        :placeholder="$t('dns.rules.form.domainPlaceholder')"
        :hint="$t('dns.rules.form.domainHelp')"
        :removable="isFieldRemovable('domain')"
        @remove="removeField('domain')"
      />

      <ChipsField
        v-if="isFieldShown('domainSuffix')"
        v-model="domainSuffix"
        :label="$t('dns.rules.form.domainSuffix')"
        :placeholder="$t('dns.rules.form.domainSuffixPlaceholder')"
        :hint="$t('dns.rules.form.domainSuffixHelp')"
        :removable="isFieldRemovable('domainSuffix')"
        @remove="removeField('domainSuffix')"
      />

      <ChipsField
        v-if="isFieldShown('domainKeyword')"
        v-model="domainKeyword"
        :label="$t('dns.rules.form.domainKeyword')"
        :placeholder="$t('dns.rules.form.domainKeywordPlaceholder')"
        :hint="$t('dns.rules.form.domainKeywordHelp')"
        :removable="isFieldRemovable('domainKeyword')"
        @remove="removeField('domainKeyword')"
      />

      <ChipsField
        v-if="isFieldShown('geosite')"
        v-model="geosite"
        :label="$t('dns.rules.form.geosite')"
        :placeholder="$t('dns.rules.form.geositePlaceholder')"
        :hint="$t('dns.rules.form.geositeHelp')"
        :removable="isFieldRemovable('geosite')"
        @remove="removeField('geosite')"
      />

      <!-- The matchers this rule does not use yet: one click each, and the list
           doubles as documentation of what can be matched on. -->
      <div v-if="hiddenFields.length" class="flex flex-wrap items-center gap-2">
        <span class="text-xs text-gray-500 dark:text-gray-400">
          {{ $t('dns.rules.form.matchers.add') }}
        </span>
        <button
          v-for="key in hiddenFields"
          :key="key"
          type="button"
          class="inline-flex items-center gap-1 px-2.5 py-1 rounded-pill border border-dashed border-gray-300 dark:border-gray-600 text-xs text-gray-600 dark:text-gray-300 hover:border-primary-400 hover:text-primary-600 dark:hover:text-primary-300 transition-colors"
          @click="addField(key)"
        >
          <PlusCircleIcon class="h-3.5 w-3.5" />
          {{ $t(matcherLabelKeys[key]) }}
        </button>
      </div>
    </template>

    <button
      v-else
      type="button"
      class="w-full flex items-center gap-2 px-3 py-2 rounded-control border border-dashed border-gray-300 dark:border-gray-600 text-left text-xs text-gray-500 dark:text-gray-400 hover:border-primary-400 hover:text-primary-600 dark:hover:text-primary-300 transition-colors"
      @click="expandMatchers"
    >
      <PlusCircleIcon class="h-4 w-4 shrink-0" />
      <span class="flex-1">{{ $t('dns.rules.form.mixing.matchersCollapsed') }}</span>
      <span class="font-medium">{{ $t('dns.rules.form.mixing.show') }}</span>
    </button>
  </div>
</template>
