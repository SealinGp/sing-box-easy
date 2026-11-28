<script setup lang="ts">
import Popover from '../../volt/Popover.vue'
import StatusIndicator from '../../components/StatusIndicator.vue'
import IconButton from '../../components/IconButton.vue'
import MonacoEditor from '../../components/MonacoEditor.vue'
import Table from '../../components/Table.vue'
import MonacoDiffEditor from '../../components/MonacoDiffEditor.vue'
import { Dialog } from '../../volt'
import { useConfigEditor } from '../../composables/useConfigEditor'
const { t, configContent, loading, saving, rawImpactPanel, saveAnchor, rawReview, saveReviewed, validating, restarting, hasChanges, isFullscreen, isSplit, editorTheme, showVersions, versions, versionsLoading, versionRetentionDays, selectedVersionIds, batchDeleting, allVersionsSelected, showDiff, diffVersionId, diffOriginal, diffModified, editorOptions, handleContentChange, saveConfig, validateConfig, openVersions, closeVersions, toggleSelectAllVersions, toggleSelectVersion, batchDeleteVersions, openDiff, rollbackTo, deleteVersion, restartService, formatTime, formatRelative, formatBytes, resetChanges, toggleFullscreen, toggleTheme, toggleSplit, formatDocument } = useConfigEditor()
</script>

<template>
  <Popover ref="rawImpactPanel" panel-class="w-[min(34rem,calc(100vw-2rem))]">
    <div v-if="rawReview" class="space-y-4">
      <h2 class="font-semibold">{{ t('managed.rawImpact') }}</h2>
      <p class="text-sm text-text-secondary">{{ t('managed.rawImpactHelp') }}</p>
      <ul class="space-y-2 text-sm"><li v-for="p in rawReview.policies" :key="p.id">{{ p.name }} · {{ t('managed.'+p.status) }}</li></ul>
      <RouterLink to="/dashboard/route/policies" class="text-sm text-primary underline">{{ t('lab.trafficPolicies') }}</RouterLink>
      <div class="flex justify-end gap-3"><button type="button" :disabled="saving" @click="rawImpactPanel?.hide()">{{ t('managed.cancel') }}</button><button type="button" class="rounded-control bg-primary px-3 py-2 text-on-primary" :disabled="saving" @click="saveReviewed">{{ t('managed.saveRaw') }}</button></div>
    </div>
  </Popover>

  <!--
    Outer wrapper is pinned to 100vh and laid out as a single flex column so
    the editor child can claim all the leftover viewport height instead of
    waiting for a parent `h-full` chain that may or may not resolve. The
    fullscreen toggle still wins via `fixed inset-0 z-50` when active.
  -->
  <div
    class="p-4 h-full min-h-[24rem] flex flex-col overflow-hidden"
    :class="{ 'fixed inset-0 z-50 bg-gray-50 dark:bg-gray-900': isFullscreen }"
  >
      <!-- Header -->
      <div class="flex flex-wrap justify-between items-center gap-3 mb-4 shrink-0">
        <h2 class="text-3xl font-bold text-gray-900 dark:text-gray-100">{{ $t('config.title') }}</h2>
        <div class="flex flex-wrap items-center gap-2">
          <!-- Theme Toggle -->
          <button
            @click="toggleTheme"
            class="p-2 text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-control transition-colors"
            :title="$t('config.toggleTheme')"
          >
            <svg v-if="editorTheme === 'vs-dark'" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z" />
            </svg>
            <svg v-else class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z" />
            </svg>
          </button>

          <!-- Split View Toggle -->
          <button
            @click="toggleSplit"
            :aria-pressed="isSplit"
            class="p-2 rounded-control transition-colors"
            :class="isSplit
              ? 'text-primary-600 dark:text-primary-400 bg-primary-100 dark:bg-primary-900/30'
              : 'text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800'"
            :title="isSplit ? $t('config.splitOn') : $t('config.splitOff')"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 5a1 1 0 011-1h14a1 1 0 011 1v14a1 1 0 01-1 1H5a1 1 0 01-1-1V5zM12 4v16" />
            </svg>
          </button>

          <!-- Fullscreen Toggle -->
          <button
            @click="toggleFullscreen"
            :aria-pressed="isFullscreen"
            class="p-2 text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-control transition-colors"
            :title="isFullscreen ? $t('config.exitFullscreen') : $t('config.enterFullscreen')"
          >
            <svg v-if="!isFullscreen" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4" />
            </svg>
            <svg v-else class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 9V4m0 0H4m5 0l-5 5m11-5h5m0 0v5m0-5l-5 5m-5 6v5m0 0h5m-5 0l5-5m5 5v-5m0 5h-5m5 0l-5-5" />
            </svg>
          </button>

          <!-- Actions -->
          <button
            @click="formatDocument"
            :disabled="loading"
            class="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-control hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors disabled:opacity-50"
          >
            {{ $t('config.format') }}
          </button>

          <button
            @click="validateConfig"
            :disabled="loading || validating || !configContent"
            class="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-control hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors disabled:opacity-50"
          >
            <span v-if="validating">{{ $t('config.validating') }}</span>
            <span v-else>{{ $t('config.validate') }}</span>
          </button>

          <button
            @click="restartService"
            :disabled="restarting"
            class="px-4 py-2 text-sm font-medium text-emerald-700 dark:text-emerald-300 bg-emerald-50 dark:bg-emerald-900/20 border border-emerald-300 dark:border-emerald-700 rounded-control hover:bg-emerald-100 dark:hover:bg-emerald-900/40 transition-colors disabled:opacity-50"
            :title="$t('config.restartTitle')"
          >
            <span v-if="restarting">{{ $t('config.restarting') }}</span>
            <span v-else>{{ $t('config.restart') }}</span>
          </button>

          <button
            @click="openVersions"
            :disabled="loading"
            class="px-4 py-2 text-sm font-medium text-amber-700 dark:text-amber-300 bg-amber-50 dark:bg-amber-900/20 border border-amber-300 dark:border-amber-700 rounded-control hover:bg-amber-100 dark:hover:bg-amber-900/40 transition-colors disabled:opacity-50"
            :title="$t('config.versionsTitle')"
          >
            {{ $t('config.versions') }}
          </button>

          <button
            @click="resetChanges"
            :disabled="loading || saving || !hasChanges"
            class="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-control hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors disabled:opacity-50"
          >
            {{ $t('config.reset') }}
          </button>

          <button
            ref="saveAnchor"
            @click="saveConfig"
            :disabled="loading || saving || !hasChanges"
            class="px-4 py-2 text-sm font-medium text-white bg-primary-600 rounded-control hover:bg-primary-700 transition-colors disabled:opacity-50 flex items-center gap-2"
          >
            <span v-if="saving">{{ $t('config.saving') }}</span>
            <span v-else>{{ $t('config.save') }}</span>
            <span v-if="!saving && hasChanges" class="text-xs opacity-75">(Ctrl+S)</span>
          </button>
        </div>
      </div>

      <!--
        Editor Container.
        `min-h-0` is critical: inside a flex column, `flex-1` children
        default to min-height:auto and refuse to shrink past their
        intrinsic content height, which breaks the "fill remaining
        space" behaviour. Setting min-h-0 lets the editor expand AND
        shrink to fit whatever viewport is left after header + status.
      -->
      <div class="flex-1 min-h-0 bg-white dark:bg-gray-800 rounded-surface shadow-float overflow-hidden">
        <div v-if="loading" class="flex items-center justify-center h-full">
          <div class="animate-spin rounded-pill h-8 w-8 border-b-2 border-primary-600"></div>
        </div>
        <div v-else class="h-full">
          <MonacoEditor
            v-model="configContent"
            @update:modelValue="handleContentChange"
            language="json"
            :theme="editorTheme"
            :options="editorOptions"
            :split="isSplit"
            class="h-full"
          />
        </div>
      </div>

      <!-- Status Bar -->
      <div class="mt-2 px-4 py-2 bg-gray-100 dark:bg-gray-800 rounded-control flex items-center justify-between text-sm shrink-0">
        <div class="flex items-center gap-4">
          <span class="text-gray-600 dark:text-gray-400">
            {{ $t('config.jsonConfig') }}
          </span>
          <span v-if="hasChanges" class="flex items-center gap-1 text-amber-600 dark:text-amber-400">
            <span class="w-2 h-2 bg-amber-600 dark:bg-amber-400 rounded-pill"></span>
            {{ $t('config.modified') }}
          </span>
          <span v-else class="flex items-center gap-1 text-green-600 dark:text-green-400">
            <span class="w-2 h-2 bg-green-600 dark:bg-green-400 rounded-pill"></span>
            {{ $t('config.saved') }}
          </span>
        </div>
        <div class="text-gray-600 dark:text-gray-400">
          {{ $t('config.lines', { n: configContent.split('\n').length }) }}
        </div>
      </div>

      <!--
        Versions history.

        Was a hand-rolled `fixed inset-0` overlay at `w-[90vw] h-[80vh]` — a
        full-bleed sheet for a five-column table, which left most of the width
        empty and re-implemented the mask, the escape handling and the header
        chrome that `volt/Dialog` already owns. It is the shared dialog now, and
        the width follows the view: the list is a table (`max-w-3xl`), the diff is
        two Monaco panes side by side and genuinely needs the room (`max-w-5xl`).
      -->
      <Dialog
        :visible="showVersions"
        @update:visible="(v: boolean) => { if (!v) closeVersions() }"
        modal
        :class="showDiff ? 'w-full max-w-5xl' : 'w-full max-w-3xl'"
      >
        <!-- The header slot rather than `:header`: this one carries a back
             button, and its title depends on which view is showing. -->
        <template #header>
          <div class="flex items-center gap-3">
            <button
              v-if="showDiff"
              @click="showDiff = false"
              class="text-sm text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200"
            >
              &larr; {{ $t('common.back') }}
            </button>
            <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">
              <span v-if="!showDiff">{{ $t('config.versionsModal.title') }}</span>
              <span v-else>{{ $t('config.versionsModal.diffTitle', { id: diffVersionId }) }}</span>
            </h3>
          </div>
        </template>

        <!-- List view -->
        <div v-if="!showDiff">
          <!-- Retention tip -->
          <div class="mb-4 flex items-start gap-2 rounded-control border border-blue-200 dark:border-blue-800 bg-blue-50 dark:bg-blue-900/20 px-3 py-2 text-xs text-blue-700 dark:text-blue-300">
            <svg class="w-4 h-4 mt-0.5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <span>
              {{ $t('config.versionsModal.retentionTip', { days: versionRetentionDays }) }}
              <RouterLink
                to="/dashboard/settings"
                class="font-medium underline hover:no-underline whitespace-nowrap"
                @click="closeVersions"
              >
                {{ $t('config.versionsModal.retentionSettingsLink') }}
              </RouterLink>
            </span>
          </div>

          <div v-if="versionsLoading" class="flex items-center justify-center h-32">
            <div class="animate-spin rounded-pill h-7 w-7 border-b-2 border-primary-600"></div>
          </div>
          <div v-else-if="versions.length === 0" class="text-center text-gray-500 dark:text-gray-400 py-12">
            {{ $t('config.versionsModal.empty') }}
          </div>
          <div v-else>
            <!-- Batch toolbar -->
            <div class="flex items-center justify-between mb-3">
              <span class="text-xs text-gray-500 dark:text-gray-400">
                <template v-if="selectedVersionIds.size">{{ $t('config.versionsModal.selected', { n: selectedVersionIds.size }) }}</template>
              </span>
              <IconButton
                action="delete"
                :label="$t('config.versionsModal.deleteSelected')"
                @click="batchDeleteVersions"
                :disabled="selectedVersionIds.size === 0 || batchDeleting || versionsLoading"
              />
            </div>
            <!--
              The dialog's own content area scrolls, so `scroll` is off: a second
              scroll container inside it would trap the wheel.
            -->
            <Table :scroll="false">
              <template #head>
                <th class="w-8">
                  <input
                    type="checkbox"
                    class="cursor-pointer"
                    :checked="allVersionsSelected"
                    @change="toggleSelectAllVersions"
                    :aria-label="$t('config.versionsModal.selectAll')"
                    :title="$t('config.versionsModal.selectAll')"
                  />
                </th>
                <th>{{ $t('config.versionsModal.colVersion') }}</th>
                <th>{{ $t('config.versionsModal.colSavedAt') }}</th>
                <th>{{ $t('config.versionsModal.colSize') }}</th>
                <th class="col-actions">{{ $t('config.versionsModal.colActions') }}</th>
              </template>

              <tr
                v-for="(v, i) in versions"
                :key="v.id"
                class="text-gray-800 dark:text-gray-200"
                :class="selectedVersionIds.has(v.id) ? 'bg-primary-50/60 dark:bg-primary-900/10' : ''"
              >
                <td>
                  <input
                    type="checkbox"
                    class="cursor-pointer"
                    :checked="selectedVersionIds.has(v.id)"
                    @change="toggleSelectVersion(v.id)"
                    :aria-label="$t('config.versionsModal.selectOne', { id: v.id })"
                  />
                </td>
                <td>
                  #{{ v.id }}
                  <StatusIndicator v-if="i === 0" variant="primary" class="ml-2">{{ $t('config.versionsModal.latest') }}</StatusIndicator>
                </td>
                <td class="text-gray-600 dark:text-gray-400">
                  {{ formatTime(v.created_at) }}
                  <span class="ml-1 text-xs text-gray-400 dark:text-gray-500">({{ formatRelative(v.created_at) }})</span>
                </td>
                <td class="text-gray-600 dark:text-gray-400">{{ formatBytes(v.size) }}</td>
                <td>
                  <div class="flex items-center justify-end gap-1">
                    <button
                      @click="openDiff(v)"
                      class="px-3 py-1.5 text-xs font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-control hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
                    >
                      {{ $t('config.versionsModal.diff') }}
                    </button>
                    <button
                      @click="rollbackTo(v)"
                      :disabled="loading"
                      class="px-3 py-1.5 text-xs font-medium text-amber-700 dark:text-amber-300 bg-amber-50 dark:bg-amber-900/20 border border-amber-300 dark:border-amber-700 rounded-control hover:bg-amber-100 dark:hover:bg-amber-900/40 transition-colors disabled:opacity-50"
                    >
                      {{ $t('config.versionsModal.rollback') }}
                    </button>
                    <IconButton action="delete" @click="deleteVersion(v)" :disabled="loading || versionsLoading" :label="$t('config.versionsModal.delete')" />
                  </div>
                </td>
              </tr>
            </Table>
          </div>
        </div>

        <!-- Diff view -->
        <div v-else class="flex flex-col">
          <!-- Left/right legend so it's clear which side is which (and that
               rollback restores the left). -->
          <div class="flex items-center justify-between pb-2 text-xs border-b border-gray-200 dark:border-gray-700 shrink-0">
            <span class="flex items-center gap-1.5 font-medium text-amber-700 dark:text-amber-300">
              <span class="w-2 h-2 rounded-pill bg-amber-500"></span>
              {{ $t('config.versionsModal.diffLeft', { id: diffVersionId }) }}
            </span>
            <span class="flex items-center gap-1.5 font-medium text-gray-600 dark:text-gray-400">
              {{ $t('config.versionsModal.diffRight') }}
              <span class="w-2 h-2 rounded-pill bg-gray-400"></span>
            </span>
          </div>
          <!-- Monaco needs a resolved height; the dialog caps itself at 90vh, so
               this sits just inside that with room for the header and footer. -->
          <div class="h-[62vh] mt-2">
            <MonacoDiffEditor
              :original="diffOriginal"
              :modified="diffModified"
              language="json"
              :theme="editorTheme"
              class="h-full"
            />
          </div>
        </div>

        <template v-if="showDiff" #footer>
          <div class="flex w-full items-center justify-between gap-2">
            <!-- Rollback sits on the LEFT, aligned with the left pane it restores. -->
            <div class="flex items-center gap-3">
              <button
                v-if="diffVersionId !== null"
                @click="rollbackTo({ id: diffVersionId, size: 0, created_at: '' })"
                :disabled="loading"
                class="px-4 py-2 text-sm font-medium text-amber-700 dark:text-amber-300 bg-amber-50 dark:bg-amber-900/20 border border-amber-300 dark:border-amber-700 rounded-control hover:bg-amber-100 dark:hover:bg-amber-900/40 transition-colors disabled:opacity-50"
              >
                {{ $t('config.versionsModal.rollbackToLeft', { id: diffVersionId }) }}
              </button>
              <span class="text-xs text-gray-500 dark:text-gray-400 hidden sm:inline">
                {{ $t('config.versionsModal.diffRollbackHint', { id: diffVersionId }) }}
              </span>
            </div>
            <button
              @click="showDiff = false"
              class="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-control hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
            >
              {{ $t('config.versionsModal.backToList') }}
            </button>
          </div>
        </template>
      </Dialog>
  </div>
</template>