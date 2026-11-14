<script setup lang="ts">
import IconButton from '../../components/IconButton.vue'
import PopConfirm from '../../components/PopConfirm.vue'
import Button from '../../components/Button.vue'
import { PlusIcon, AdjustmentsHorizontalIcon, SparklesIcon, InformationCircleIcon } from '@heroicons/vue/24/outline'
import { provide } from 'vue'
import { useNodeRulesPage } from '../../composables/useNodeRulesPage'
import { nodeRulesPageKey } from '../../composables/nodeRulesPageContext'
import NodeRulesEditors from '../../components/NodeRulesEditors.vue'
const state = useNodeRulesPage()
provide(nodeRulesPageKey, state)
const { t, store, filters, groups, templates, preview, loading, error, applying, notice, mode, showPreview, modeOptions, startCreateFilter, startEditFilter, removeFilter, startCreateGroup, startEditGroup, removeGroup, addTemplate, doApply, filterName, isFocused } = state
</script>
<template>

  <div class="space-y-3 animate-fade-in">
    <!-- Header: subtitle + actions -->
    <div class="flex flex-wrap items-center justify-between gap-2">
      <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('nodeRules.subtitle') }}</p>
      <div class="flex items-center gap-2">
        <!--
          Two modes, one control. A segmented switch rather than a pair of
          buttons: the pill shows which view you are in, which a button that
          merely triggers something cannot.
        -->
        <div
          class="inline-flex items-center gap-0.5 p-0.5 rounded-control bg-gray-100 dark:bg-slate-800 border border-border"
          role="tablist"
          :aria-label="t('nodeRules.modes.switchHint')"
        >
          <button
            v-for="option in modeOptions"
            :key="option.value"
            type="button"
            role="tab"
            :aria-selected="mode === option.value"
            class="px-2.5 py-1 text-xs font-semibold rounded-control transition-colors duration-200 cursor-pointer"
            :class="mode === option.value
            ? 'bg-bg-secondary text-primary-700 dark:text-primary-400 shadow-surface'
            : 'text-gray-500 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white'"
            @click="option.value === 'preview' ? showPreview() : (mode = 'edit')"
          >
            {{ option.label }}
          </button>
        </div>
        <Button variant="primary" size="sm" action :disabled="applying" @click="doApply">
          {{ applying ? t('nodeRules.applying') : t('nodeRules.applyNow') }}
        </Button>
      </div>
    </div>

    <!-- Alert Messages -->
    <div v-if="notice" class="px-3 py-2 rounded-control bg-primary-500/10 border border-primary-500/20 text-primary-700 dark:text-primary-400 text-sm animate-fade-in flex items-center gap-2">
      <span class="w-1.5 h-1.5 rounded-pill bg-primary-500 animate-ping"></span>
      <span>{{ notice }}</span>
    </div>
    <div v-if="error" class="px-3 py-2 rounded-control bg-red-500/10 border border-red-500/20 text-red-700 dark:text-red-400 text-sm animate-fade-in flex items-center gap-2">
      <span class="w-1.5 h-1.5 rounded-pill bg-red-500 animate-ping"></span>
      <span>{{ error }}</span>
    </div>

    <!--
      Edit ⇄ Preview. `mode="out-in"` so the two never overlap: they are
      different shapes at different heights, and cross-fading them makes the
      page jump. The cost is a ~150ms gap, which is what the leave/enter
      durations below are tuned around.
    -->
    <Transition name="mode-swap" mode="out-in">
      <div v-if="mode === 'edit'" key="edit" class="space-y-3">
        <!-- Templates -->
        <section v-if="templates.length" class="space-y-2 bg-gray-50/50 dark:bg-slate-900/40 border border-gray-100 dark:border-gray-800/80 px-3 py-2.5 rounded-surface">
          <h2 class="text-xs font-bold text-gray-400 dark:text-gray-500 uppercase tracking-wider">{{ t('nodeRules.templates') }}</h2>
          <div class="flex flex-wrap gap-1.5">
            <button
              v-for="tpl in templates"
              :key="tpl.id"
              class="border border-primary-200 dark:border-primary-800 hover:border-primary-500 hover:bg-primary-50 dark:hover:bg-primary-950/20 text-primary-700 dark:text-primary-400 text-xs font-semibold px-2 py-1 rounded-control transition-all cursor-pointer"
              :title="tpl.description"
              @click="addTemplate(tpl.id)"
            >
              + {{ tpl.name }}
            </button>
          </div>
        </section>

        <!-- Filters & Groups Grid -->
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-3">
          <!-- Filters panel -->
          <section class="space-y-2 flex flex-col">
            <div class="flex items-center justify-between">
              <h2 class="text-sm font-bold text-text-primary flex items-center gap-1.5">
                <AdjustmentsHorizontalIcon class="h-4 w-4 text-primary-500" />
                {{ t('nodeRules.filters') }}
              </h2>
              <button
                class="node-rule-primary-button bg-primary-600 hover:bg-primary-500 text-white text-xs font-bold px-2 py-1 rounded-control transition-colors cursor-pointer flex items-center gap-1"
                @click="startCreateFilter"
              >
                <PlusIcon class="h-3.5 w-3.5" />
                {{ t('nodeRules.addFilter') }}
              </button>
            </div>

            <!-- Filter list -->
            <ul class="space-y-1.5">
              <li
                v-for="f in filters"
                :key="f.id"
                :class="{ 'focus-target': isFocused(f.name) }"
                class="node-rule-card rounded-control border border-border px-2.5 py-2 flex items-start justify-between gap-2 bg-bg-secondary transition-colors duration-200"
              >
                <div class="min-w-0 space-y-1">
                  <div class="flex items-center gap-2 flex-wrap">
                    <span class="font-bold text-sm text-text-primary truncate">{{ f.name }}</span>
                    <span class="px-2 py-0.5 rounded-pill text-[10px] font-bold uppercase tracking-wider bg-primary-100 dark:bg-primary-950/40 text-primary-700 dark:text-primary-400">
                      {{ f.outbound_type }}
                    </span>
                    <span v-if="f.is_fallback" class="px-2 py-0.5 rounded-pill text-[10px] font-bold bg-amber-100 dark:bg-amber-950/40 text-amber-700 dark:text-amber-400">
                      {{ t('nodeRules.fallback') }}
                    </span>
                    <span class="text-xs text-gray-400 font-mono">P{{ f.priority }}</span>
                  </div>
                  <div v-if="!f.is_fallback" class="flex flex-wrap gap-1">
                    <span v-for="(m, i) in f.matchers" :key="i" class="px-2 py-0.5 rounded-surface text-[11px] bg-bg-tertiary border border-border text-text-secondary font-medium">
                      {{ m.value }}
                    </span>
                    <span v-if="!f.matchers.length" class="text-xs text-gray-400 italic">{{ t('nodeRules.noMatchers') }}</span>
                    <span
                      v-for="(m, i) in (f.excludes ?? [])"
                      :key="`x${i}`"
                      class="px-2 py-0.5 rounded-control text-[11px] bg-red-50/50 dark:bg-red-950/10 border border-red-200/40 dark:border-red-950/20 text-red-600 dark:text-red-400 font-medium"
                      :title="t('nodeRules.excludeBadgeTitle')"
                    >
                      − {{ m.value }}
                    </span>
                  </div>
                  <div v-else class="text-xs text-gray-400">{{ t('nodeRules.fallbackHint') }}</div>
                </div>
                <div class="flex gap-0.5 shrink-0">
                  <IconButton
                    action="edit"
                    @click="startEditFilter(f)"
                    :label="t('nodeRules.edit')"
                  />
                  <PopConfirm
                    v-if="!f.is_fallback"
                    :message="t('nodeRules.confirmDeleteFilter', { name: f.name })"
                    :confirm-label="t('nodeRules.delete')"
                    tone="danger"
                    @confirm="removeFilter(f)"
                  />
                </div>
              </li>
            </ul>
          </section>

          <!-- Groups panel -->
          <section class="space-y-2 flex flex-col">
            <div class="flex items-center justify-between">
              <h2 class="text-sm font-bold text-text-primary flex items-center gap-1.5">
                <SparklesIcon class="h-4 w-4 text-primary-500" />
                {{ t('nodeRules.groups') }}
              </h2>
              <button
                class="node-rule-primary-button bg-primary-600 hover:bg-primary-500 text-white text-xs font-bold px-2 py-1 rounded-control transition-colors cursor-pointer flex items-center gap-1"
                @click="startCreateGroup"
              >
                <PlusIcon class="h-3.5 w-3.5" />
                {{ t('nodeRules.addGroup') }}
              </button>
            </div>

            <!-- Group list -->
            <ul class="space-y-1.5">
              <li
                v-for="g in groups"
                :key="g.id"
                :class="{ 'focus-target': isFocused(g.name) }"
                class="node-rule-card rounded-control border border-border px-2.5 py-2 flex items-start justify-between gap-2 bg-bg-secondary transition-colors duration-200"
              >
                <div class="min-w-0 space-y-1">
                  <div class="flex items-center gap-2">
                    <span class="font-bold text-sm text-text-primary truncate">{{ g.name }}</span>
                    <span class="text-xs text-gray-400 font-mono">P{{ g.priority }}</span>
                  </div>
                  <div class="flex flex-wrap gap-1">
                    <span v-for="fid in g.filter_ids" :key="fid" class="px-2 py-0.5 rounded-surface text-[11px] bg-bg-tertiary border border-border text-text-secondary font-medium">
                      {{ filterName(fid) }}
                    </span>
                    <span
                      v-for="tag in (g.extra_tags ?? [])"
                      :key="`t${tag}`"
                      class="px-2 py-0.5 rounded-surface text-[11px] bg-emerald-50/60 dark:bg-emerald-950/10 border border-emerald-200/50 dark:border-emerald-900/30 text-emerald-700 dark:text-emerald-400 font-medium"
                      :title="t('nodeRules.groupTagBadgeTitle')"
                    >
                      {{ tag }}
                    </span>
                    <span v-if="!g.filter_ids.length && !(g.extra_tags ?? []).length" class="text-xs text-gray-400 italic">{{ t('nodeRules.noFilters') }}</span>
                  </div>
                </div>
                <div class="flex gap-0.5 shrink-0">
                  <IconButton
                    action="edit"
                    @click="startEditGroup(g)"
                    :label="t('nodeRules.edit')"
                  />
                  <PopConfirm
                    :message="t('nodeRules.confirmDeleteGroup', { name: g.name })"
                    :confirm-label="t('nodeRules.delete')"
                    tone="danger"
                    @confirm="removeGroup(g)"
                  />
                </div>
              </li>
              <li v-if="!groups.length" class="text-xs text-gray-400 text-center py-4 bg-gray-50/20 dark:bg-slate-900/20 border border-dashed border-border rounded-control">
                {{ t('nodeRules.noGroups') }}
              </li>
            </ul>
          </section>
        </div>
      </div>

      <div v-else key="preview" class="space-y-3">
        <!--
          Preview never re-renders the editor's data, so a run that returns
          nothing has to say so — an empty page here reads as a broken button.
        -->
        <div v-if="!preview" class="text-center py-10 bg-gray-50/40 dark:bg-slate-900/20 border border-dashed border-border rounded-surface space-y-2">
          <p class="text-sm font-semibold text-gray-500 dark:text-gray-400">{{ t('nodeRules.previewEmpty') }}</p>
          <p class="text-xs text-gray-400 max-w-md mx-auto">{{ t('nodeRules.previewEmptyHint') }}</p>
          <Button variant="secondary" size="sm" action :disabled="loading" @click="store.runPreview()">
            {{ t('nodeRules.previewRun') }}
          </Button>
        </div>

        <!-- Preview Section -->
        <section v-if="preview" class="space-y-2">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <h2 class="text-sm font-bold text-text-primary flex items-center gap-1.5">
              <InformationCircleIcon class="h-4 w-4 text-primary-500" />
              {{ t('nodeRules.previewResult') }}
              <span class="text-xs font-normal text-gray-400">({{ t('nodeRules.endpoints', { n: preview.endpoints }) }})</span>
            </h2>
            <!-- Entering the mode already re-runs it; this is for the second look
              after an edit made in a dialog, which does not change the mode. -->
            <Button variant="secondary" size="sm" action :disabled="loading" @click="store.runPreview()">
              {{ loading ? t('nodeRules.applying') : t('nodeRules.previewRefresh') }}
            </Button>
          </div>
          <p class="text-xs text-gray-400">{{ t('nodeRules.modes.switchHint') }}</p>

          <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-2">
            <div v-for="pf in preview.filters" :key="pf.id" class="node-rule-card rounded-control border border-border bg-bg-secondary px-2.5 py-2 self-start transition-colors duration-200">
              <!--
                No collapse. The whole point of preview mode is reading which
                nodes landed where, and a card that has to be opened first hides
                exactly that behind a click per filter. The list scrolls inside
                its own `max-h-44` instead, so a filter holding 60 nodes still
                costs one card's worth of page.
              -->
              <div class="w-full flex items-center justify-between gap-2">
                <span class="font-bold text-sm text-text-primary truncate flex items-center gap-1.5 min-w-0">
                  <span class="truncate">{{ pf.name }}</span>
                  <span v-if="pf.is_fallback" class="px-2 py-0.5 rounded-pill text-[9px] font-bold bg-amber-100 dark:bg-amber-950/40 text-amber-700 dark:text-amber-400 shrink-0">{{ t('nodeRules.fallback') }}</span>
                </span>
                <span class="px-2 py-0.5 rounded-control text-xs font-bold shrink-0" :class="pf.member_count ? 'bg-primary-100 dark:bg-primary-950 text-primary-700 dark:text-primary-400' : 'bg-gray-100 dark:bg-slate-800 text-gray-400'">
                  {{ pf.member_count }}
                </span>
              </div>
              <div v-if="pf.member_count" class="mt-2 max-h-44 overflow-y-auto space-y-1 border-t border-gray-100 dark:border-gray-800 pt-2 text-xs font-mono">
                <div
                  v-for="(tag, i) in pf.members"
                  :key="i"
                  class="text-gray-600 dark:text-gray-350 truncate hover:text-gray-900 dark:hover:text-white py-0.5"
                  :title="tag"
                >
                  {{ tag }}
                </div>
              </div>
            </div>

            <!--
              Unmatched nodes — the last cell of the same grid, not a full-width
              band under it. It is one more bucket ("what no filter claimed"),
              so it reads as a peer of the filter cards; stretched across the
              row it looked like a separate section, and its own row of
              whitespace was the widest thing on the page.
            -->
            <div v-if="preview.unmatched.length" class="node-rule-card rounded-control border border-amber-200 dark:border-amber-900/40 bg-bg-secondary px-2.5 py-2 self-start transition-colors duration-200">
              <div class="w-full flex items-center justify-between gap-2">
                <span class="font-bold text-sm text-amber-700 dark:text-amber-400 truncate">
                  {{ t('nodeRules.unmatchedTitle') }}
                </span>
                <span class="px-2 py-0.5 rounded-control text-xs font-bold shrink-0 bg-amber-100 dark:bg-amber-950/40 text-amber-700 dark:text-amber-400">
                  {{ preview.unmatched.length }}
                </span>
              </div>
              <p class="mt-0.5 text-[11px] text-amber-600/80 dark:text-amber-500/70">
                {{ t('nodeRules.unmatchedHint') }}
              </p>
              <div class="mt-2 max-h-44 overflow-y-auto space-y-1 border-t border-amber-200/50 dark:border-amber-950/20 pt-2 text-xs font-mono">
                <div
                  v-for="(tag, i) in preview.unmatched"
                  :key="i"
                  class="text-gray-600 dark:text-gray-350 truncate py-0.5 hover:text-gray-900 dark:hover:text-white"
                  :title="tag"
                >
                  {{ tag }}
                </div>
              </div>
            </div>
          </div>
        </section>
      </div>
    </Transition>

    <!-- Create/Edit Filter Modal -->
    <NodeRulesEditors />

  </div>

</template>
<style scoped>
/*
 * Edit ⇄ Preview.
 *
 * 220ms in / 150ms out on the same curve the select panel and the nav pill use
 * — below ~150ms a transition stops reading as motion and just looks like a
 * flicker. The two directions travel opposite ways so the swap
 * reads as one view moving aside for the other rather than two unrelated fades.
 */
.mode-swap-enter-active {
  transition:
    opacity 0.22s ease-out,
    transform 0.22s cubic-bezier(0.32, 0.72, 0, 1);
}

.mode-swap-leave-active {
  transition:
    opacity 0.15s ease-in,
    transform 0.15s ease-in;
}

.mode-swap-enter-from {
  opacity: 0;
  transform: translateY(8px);
}

.mode-swap-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}

/* Same treatment the sidebar gives its pill: keep the cross-fade, drop the
   travel, and shorten it. */
@media (prefers-reduced-motion: reduce) {
  .mode-swap-enter-active,
  .mode-swap-leave-active {
    transition: opacity 0.12s linear;
  }

  .mode-swap-enter-from,
  .mode-swap-leave-to {
    transform: none;
  }
}

.node-rule-card {
  box-shadow:
    0 8px 22px rgba(15, 23, 42, 0.045),
    inset 0 1px 0 rgba(255, 255, 255, 0.42);
}

.node-rule-card:hover {
  box-shadow:
    0 12px 28px rgba(15, 23, 42, 0.065),
    inset 0 1px 0 rgba(255, 255, 255, 0.48);
}

.node-rule-primary-button {
  box-shadow: none;
}
</style>
