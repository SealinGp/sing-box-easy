<script setup lang="ts">
/**
 * The "conditions" half of the DNS rule editor.
 *
 * sing-box combines the matchers WITHIN one rule with AND, not OR. So
 *
 *   { "rule_set": ["blocklist"], "domain": ["example.com"], "server": "block" }
 *
 * does NOT mean "the blocklist plus example.com" — it means "example.com, but
 * only if it is also in the blocklist", which is almost never what someone
 * building a rule intends. The two ways of matching are therefore presented as
 * alternatives: whichever one the rule already uses is shown, the other is
 * collapsed out of the way. Nothing is ever hidden while it holds a value, and
 * the collapsed side stays one click away — mixing is legal, so we explain the
 * consequence instead of forbidding it.
 */
import { computed, ref, watch } from 'vue'
import { Select } from '../volt'
import ChipsField from './ChipsField.vue'
import Alert from './Alert.vue'
import { PlusCircleIcon } from '@heroicons/vue/24/outline'

const ruleSet = defineModel<string>('ruleSet', { required: true })
const domain = defineModel<string[]>('domain', { required: true })
const domainSuffix = defineModel<string[]>('domainSuffix', { required: true })
const domainKeyword = defineModel<string[]>('domainKeyword', { required: true })
const geosite = defineModel<string[]>('geosite', { required: true })

defineProps<{
  ruleSetOptions: { value: string; label: string }[]
}>()

const hasRuleSet = computed(() => !!ruleSet.value)
const matcherCount = computed(
  () =>
    domain.value.length +
    domainSuffix.value.length +
    domainKeyword.value.length +
    geosite.value.length,
)
const hasMatchers = computed(() => matcherCount.value > 0)

const showRuleSet = ref(true)
const showMatchers = ref(true)
// Per-side, set once the operator deliberately opens the OTHER matching style,
// so the AND warning appears before they type rather than after the damage is
// done. Cleared again when they fold that side back away.
const ruleSetOpenedByUser = ref(false)
const matchersOpenedByUser = ref(false)

// Follows the rule as it is built, not just as it was loaded: picking a rule set
// folds the domain conditions away, and adding the first domain/suffix/keyword/
// geosite folds the rule set away. Runs immediately, so opening an existing rule
// lands on the same layout its contents imply.
//
// Three guards keep this from fighting the operator:
//   - only the EMPTY side is ever folded, so no value is hidden;
//   - a side the operator opened on purpose (`*OpenedByUser`) is left alone —
//     that is how deliberate mixing stays possible;
//   - clearing the rule out entirely restores both, since neither style is in
//     use any more and the choice is open again.
watch(
  [hasRuleSet, hasMatchers],
  ([usesRuleSet, usesMatchers]) => {
    if (!usesRuleSet && !usesMatchers) {
      showRuleSet.value = true
      showMatchers.value = true
      ruleSetOpenedByUser.value = false
      matchersOpenedByUser.value = false
      return
    }
    if (usesRuleSet && !usesMatchers && !matchersOpenedByUser.value) {
      showMatchers.value = false
      return
    }
    if (usesMatchers && !usesRuleSet && !ruleSetOpenedByUser.value) {
      showRuleSet.value = false
    }
  },
  { immediate: true },
)

const expandRuleSet = () => {
  showRuleSet.value = true
  ruleSetOpenedByUser.value = true
}

const expandMatchers = () => {
  showMatchers.value = true
  matchersOpenedByUser.value = true
}

// Which of the four domain matchers to render. Only the ones carrying a value
// show by default: a rule typically uses one or two of them, and rendering all
// four turns the modal into a wall of empty boxes. The rest are offered as "add"
// buttons, which doubles as the discovery mechanism for what CAN be matched.
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

// Fields the operator explicitly added this session. Replaced, never mutated,
// so the computed views below re-run.
const addedFields = ref<MatcherKey[]>([])

const isFieldShown = (key: MatcherKey) =>
  matcherModels[key].value.length > 0 || addedFields.value.includes(key)

// A field is removable only while empty — the same "never hide a value" rule the
// two condition styles follow.
const isFieldRemovable = (key: MatcherKey) => matcherModels[key].value.length === 0

const hiddenFields = computed(() => matcherKeys.filter((key) => !isFieldShown(key)))

const addField = (key: MatcherKey) => {
  if (addedFields.value.includes(key)) return
  addedFields.value = [...addedFields.value, key]
}

const removeField = (key: MatcherKey) => {
  addedFields.value = addedFields.value.filter((k) => k !== key)
}

// Hiding is offered only for a side that is EMPTY while the other one carries
// the rule — a value is never folded out of sight, and the last remaining
// matching style is never hideable.
const canHideRuleSet = computed(() => !hasRuleSet.value && hasMatchers.value)
const canHideMatchers = computed(() => !hasMatchers.value && hasRuleSet.value)

const collapseRuleSet = () => {
  showRuleSet.value = false
  ruleSetOpenedByUser.value = false
}

const collapseMatchers = () => {
  showMatchers.value = false
  matchersOpenedByUser.value = false
}

// Warn on a real intersection, or as soon as the operator opts into building
// one. A fresh rule with neither side filled stays quiet.
const showMixWarning = computed(
  () =>
    (hasRuleSet.value && hasMatchers.value) ||
    ruleSetOpenedByUser.value ||
    matchersOpenedByUser.value,
)
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
      <Select
        class="w-full"
        optionLabel="label"
        optionValue="value"
        v-model="ruleSet"
        :options="ruleSetOptions"
        :filter="true"
        :showClear="true"
        :placeholder="$t('dns.rules.form.ruleSetSelect')"
        :filterPlaceholder="$t('dns.rules.form.ruleSetSearch')"
        :emptyFilterMessage="$t('dns.rules.form.ruleSetNoOptions')"
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
