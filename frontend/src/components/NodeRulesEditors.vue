<script setup lang="ts">
import FieldHelp from './FieldHelp.vue'
import IconButton from '../components/IconButton.vue'
import Button from '../components/Button.vue'
import Input from '../components/Input.vue'
import SearchPicker from '../components/SearchPicker.vue'
import { ContextPanel } from '../volt'
import { Select } from '../volt'
import { XMarkIcon, AdjustmentsHorizontalIcon, SparklesIcon } from '@heroicons/vue/24/outline'
import { inject } from 'vue'
import { nodeRulesPageKey } from '../composables/nodeRulesPageContext'
const state = inject(nodeRulesPageKey)!
const { t, filters, keywords, outboundTypeOptions, matcherTypeOptions, editingFilterId, filterForm, endpointTags, nodeOptions, countryOptions, cancelFilterEdit, addMatcher, removeMatcher, addMatcherNode, addCodeMatcher, addExclude, removeExclude, addExcludeNode, saveFilter, editingGroupId, groupForm, addGroupTag, removeGroupTag, cancelGroupEdit, toggleGroupFilter, saveGroup } = state
</script>
<template>
    <ContextPanel
      :visible="editingFilterId !== null"
      @update:visible="(v: boolean) => { if (!v) cancelFilterEdit() }"
      modal
      class="w-full max-w-2xl"
    >
      <!-- The header slot rather than `:header`, so the section icon survives
        the move onto the shared dialog. -->
      <template #header>
        <h3 class="text-sm font-bold text-text-primary flex items-center gap-2">
          <AdjustmentsHorizontalIcon class="h-5 w-5 text-primary-500" />
          {{ editingFilterId === '' ? $t('nodeRules.modal.addFilter') : $t('nodeRules.modal.editFilter') }}
        </h3>
      </template>

      <div class="filter-form space-y-3">
        <section class="space-y-2.5">
          <h4 class="text-xs font-semibold text-text-secondary">{{ t('nodeRules.form.basics') }}</h4>
          <div class="filter-basics">
            <Input class="filter-name" v-model="filterForm.name" :label="t('nodeRules.form.name')" :placeholder="t('nodeRules.filterName')" />
            <div class="min-w-0">
              <label for="filter-outbound-type" class="mb-1 block text-sm font-medium text-text-secondary">{{ t('nodeRules.form.outboundType') }}</label>
              <Select input-id="filter-outbound-type" class="w-full min-w-0" v-model="filterForm.outbound_type" :options="outboundTypeOptions" optionLabel="label" optionValue="value" />
            </div>
            <Input v-model.number="filterForm.priority" :label="t('nodeRules.form.priority')" :hint="t('nodeRules.form.priorityHint')" type="number" />
          </div>

        </section>

        <!-- Rule Matchers Section -->
        <div class="space-y-2.5 border-t border-gray-100 dark:border-gray-800 pt-3">
          <h4 class="text-xs font-bold text-gray-400 dark:text-gray-500 uppercase tracking-wider flex flex-wrap items-center gap-1.5">
            Rule Matchers
            <span class="text-[11px] font-normal text-gray-400 normal-case">(Matches node names by keywords, country codes, or emoji)</span>
          </h4>

          <!-- Existing Matchers list -->
          <div class="space-y-2 max-h-40 overflow-y-auto">
            <div v-for="(m, idx) in filterForm.matchers" :key="idx" class="flex gap-2 items-center bg-gray-50 dark:bg-slate-800/40 p-1.5 rounded-control border border-gray-100 dark:border-gray-800/50">
              <Select
                class="matcher-type min-w-0 shrink-0"
                v-model="m.type"
                :options="matcherTypeOptions"
                optionLabel="label"
                optionValue="value"
              />
              <input
                v-model="m.value"
                class="bg-transparent min-w-0 w-0 flex-1 text-sm text-text-primary focus:outline-none border-none py-1 px-2"
                :placeholder="t('nodeRules.matcherValue')"
              />
              <IconButton action="delete" @click="removeMatcher(idx)" />
            </div>
            <div v-if="!filterForm.matchers.length" class="text-xs text-gray-400 text-center py-2 bg-gray-50/40 dark:bg-slate-800/10 rounded-control border border-dashed border-border">
              No matchers configured. This filter will not match any nodes.
            </div>
          </div>

          <!-- Add Matcher Buttons -->
          <div class="flex flex-wrap gap-2 items-center">
            <button
              type="button"
              @click="addMatcher('keyword')"
              class="border border-primary-200 dark:border-primary-800 hover:bg-primary-50 dark:hover:bg-primary-950/20 text-primary-600 dark:text-primary-400 text-xs font-semibold px-2 py-1 rounded-control transition-all cursor-pointer"
            >
              + Add Keyword
            </button>
            <button
              type="button"
              @click="addMatcher('emoji')"
              class="border border-primary-200 dark:border-primary-800 hover:bg-primary-50 dark:hover:bg-primary-950/20 text-primary-600 dark:text-primary-400 text-xs font-semibold px-2 py-1 rounded-control transition-all cursor-pointer"
            >
              + Add Emoji
            </button>

            <!--
              Node Picker: adds an exact-tag keyword matcher. This is how a
              `direct` outbound joins a filter — the matcher never collects one
              automatically, so without an explicit pick it is unreachable.
            -->
            <SearchPicker
              v-if="endpointTags.length"
              :options="nodeOptions"
              :placeholder="t('nodeRules.includeNode')"
              :empty-message="t('nodeRules.noNodeMatches')"
              input-class="w-full min-w-0"
              class="min-w-0 flex-[1_1_10rem]"
              panel-class="w-64"
              @select="addMatcherNode"
            />

            <!-- Country Picker Searchable Input -->
            <!-- The whole catalog is offered uncapped (`:limit="0"`): it is a
              fixed ~200 countries, not a subscription that grows. -->
            <SearchPicker
              v-if="keywords.length"
              :options="countryOptions"
              :limit="0"
              :placeholder="t('nodeRules.addCountry')"
              :empty-message="t('nodeRules.noCodeMatches')"
              input-class="w-full min-w-0"
              class="min-w-0 flex-[1_1_10rem]"
              panel-class="w-60"
              @select="addCodeMatcher"
            />
          </div>
        </div>

        <!-- Rule Excludes Section -->
        <div class="space-y-2.5 border-t border-gray-100 dark:border-gray-800 pt-3">
          <h4 class="text-xs font-bold text-gray-400 dark:text-gray-500 uppercase tracking-wider flex flex-wrap items-center gap-1.5">
            Exclude Rules (Deny-list)
            <span class="text-[11px] font-normal text-gray-400 normal-case">(Keep nodes OUT even if they match)</span>
          </h4>

          <!-- Existing Excludes list -->
          <div class="space-y-2 max-h-40 overflow-y-auto">
            <div v-for="(m, idx) in filterForm.excludes" :key="idx" class="flex gap-2 items-center bg-red-50/20 dark:bg-red-950/5 p-1.5 rounded-control border border-red-100/50 dark:border-red-950/20">
              <Select
                class="matcher-type min-w-0 shrink-0"
                v-model="m.type"
                :options="matcherTypeOptions"
                optionLabel="label"
                optionValue="value"
              />
              <input
                v-model="m.value"
                class="bg-transparent min-w-0 w-0 flex-1 text-sm text-text-primary focus:outline-none border-none py-1 px-2"
                :placeholder="t('nodeRules.matcherValue')"
              />
              <IconButton action="delete" @click="removeExclude(idx)" />
            </div>
            <div v-if="!filterForm.excludes.length" class="text-xs text-gray-400 text-center py-2 bg-gray-50/40 dark:bg-slate-800/10 rounded-control border border-dashed border-border">
              No exclusion rules configured.
            </div>
          </div>

          <!-- Add Exclude Buttons -->
          <div class="flex flex-wrap gap-2 items-center">
            <button
              type="button"
              @click="addExclude('keyword')"
              class="border border-red-200 dark:border-red-900/50 hover:bg-red-50/10 dark:hover:bg-red-950/20 text-red-600 dark:text-red-400 text-xs font-semibold px-2 py-1 rounded-control transition-all cursor-pointer"
            >
              + Add Exclude Keyword
            </button>

            <!-- Node Picker Searchable Input -->
            <SearchPicker
              v-if="endpointTags.length"
              :options="nodeOptions"
              tone="danger"
              :placeholder="t('nodeRules.excludeNode')"
              :empty-message="t('nodeRules.noNodeMatches')"
              input-class="w-full min-w-0"
              class="min-w-0 flex-[1_1_10rem]"
              panel-class="w-64"
              @select="addExcludeNode"
            />
          </div>
        </div>

        <section v-if="filterForm.outbound_type === 'urltest'" class="space-y-2.5 border-t border-border pt-3">
          <h4 class="text-xs font-semibold text-text-secondary">{{ t('nodeRules.form.healthCheck') }}</h4>
          <div class="filter-health">
            <Input class="filter-url" v-model="filterForm.test_url" :label="t('nodeRules.form.testUrl')" placeholder="https://www.gstatic.com/generate_204" />
            <Input v-model="filterForm.test_interval" :label="t('nodeRules.form.interval')" :hint="t('nodeRules.intervalHint')" placeholder="3m" />
            <Input v-model.number="filterForm.test_tolerance" :label="t('nodeRules.form.tolerance')" type="number" min="0" placeholder="200" />
          </div>

        </section>
      </div>

      <template #footer>
        <Button variant="secondary" size="sm" action @click="cancelFilterEdit">
          {{ t('nodeRules.cancel') }}
        </Button>
        <Button variant="primary" size="sm" action @click="saveFilter">
          {{ t('nodeRules.save') }}
        </Button>
      </template>
    </ContextPanel>

    <!-- Create/Edit Group Modal -->
    <ContextPanel
      :visible="editingGroupId !== null"
      @update:visible="(v: boolean) => { if (!v) cancelGroupEdit() }"
      modal
      class="w-full max-w-md"
    >
      <!-- The header slot rather than `:header`, so the section icon survives
        the move onto the shared dialog. -->
      <template #header>
        <h3 class="text-sm font-bold text-text-primary flex items-center gap-2">
          <SparklesIcon class="h-5 w-5 text-primary-500" />
          {{ editingGroupId === '' ? $t('nodeRules.modal.addGroup') : $t('nodeRules.modal.editGroup') }}
        </h3>
      </template>

      <div class="space-y-3">
        <div>
          <label class="block text-xs font-semibold text-gray-500 dark:text-gray-400 mb-1.5">Group Name</label>
          <Input
            v-model="groupForm.name"
            :placeholder="t('nodeRules.groupName')"
          />
        </div>

        <div>
          <label class="block text-xs font-semibold text-gray-500 dark:text-gray-400 mb-1.5 flex justify-between">
            <span>Priority</span>
            <span class="text-gray-400 font-normal">lower number = matched first</span>
          </label>
          <Input
            v-model.number="groupForm.priority"
            type="number"
          />
        </div>

        <div class="space-y-2">
          <label class="block text-xs font-semibold text-gray-500 dark:text-gray-400">Select Filters to Include</label>
          <div class="flex flex-wrap gap-2 pt-1">
            <button
              v-for="f in filters"
              :key="f.id"
              type="button"
              @click="toggleGroupFilter(f.id)"
              class="px-2 py-1 text-xs font-semibold rounded-control border transition-all cursor-pointer"
              :class="groupForm.filter_ids.includes(f.id)
              ? 'bg-primary-600 border-primary-600 text-white'
              : 'bg-gray-50 dark:bg-slate-800 border-border text-gray-750 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-slate-700'"
            >
              {{ f.name }}
            </button>
          </div>
          <div v-if="!filters.length" class="text-xs text-gray-400 py-1">
            No filters available. Please add a filter first.
          </div>
        </div>

        <!--
          Outbounds named directly. Separate from the filter chips above
          because they are a different kind of member: a `direct` outbound is
          never collected by a filter's matchers, so it can only reach a group
          through here.
        -->
        <div class="space-y-2 border-t border-gray-100 dark:border-gray-800 pt-3">
          <label class="block text-xs font-semibold text-gray-500 dark:text-gray-400">
            {{ t('nodeRules.groupExtraTags') }}
            <FieldHelp :label="t('nodeRules.groupExtraTags')" :text="t('nodeRules.groupExtraTagsHint')" />
          </label>

          <div v-if="groupForm.extra_tags.length" class="flex flex-wrap gap-1">
            <span
              v-for="tag in groupForm.extra_tags"
              :key="tag"
              class="inline-flex items-center gap-1 px-2 py-0.5 rounded-control text-[11px] bg-emerald-50/60 dark:bg-emerald-950/10 border border-emerald-200/50 dark:border-emerald-900/30 text-emerald-700 dark:text-emerald-400 font-medium"
            >
              {{ tag }}
              <button type="button" class="text-emerald-500 hover:text-red-500 cursor-pointer" @click="removeGroupTag(tag)">
                <XMarkIcon class="h-3 w-3" />
              </button>
            </span>
          </div>

          <SearchPicker
            v-if="endpointTags.length"
            :options="nodeOptions"
            :placeholder="t('nodeRules.groupAddTag')"
            :empty-message="t('nodeRules.noNodeMatches')"
            @select="addGroupTag"
          />
          <div v-else class="text-xs text-gray-400 py-1">{{ t('nodeRules.groupTagsEmpty') }}</div>
        </div>
      </div>

      <template #footer>
        <Button variant="secondary" size="sm" action @click="cancelGroupEdit">
          {{ t('nodeRules.cancel') }}
        </Button>
        <Button variant="primary" size="sm" action @click="saveGroup">
          {{ t('nodeRules.save') }}
        </Button>
      </template>
    </ContextPanel>
</template>

<style scoped>
.filter-form { container-type: inline-size; }
.filter-basics { display: grid; grid-template-columns: minmax(0, 1fr) 7rem; gap: .75rem; align-items: start; }
.filter-name { grid-column: 1 / -1; }
.filter-health { display: grid; grid-template-columns: minmax(0, 8rem) minmax(0, 8rem); gap: .75rem; }
.filter-url { grid-column: 1 / -1; }
.filter-basics > *, .filter-health > * { min-width: 0; }
.matcher-type { width: clamp(5rem, 27cqi, 8rem); }
@container (min-width: 32rem) {
  .filter-basics { grid-template-columns: minmax(0, 1fr) 9rem 6rem; }
  .filter-name { grid-column: auto; }
  .filter-health { grid-template-columns: minmax(0, 1fr) 6rem 7rem; }
  .filter-url { grid-column: auto; }
}
</style>
