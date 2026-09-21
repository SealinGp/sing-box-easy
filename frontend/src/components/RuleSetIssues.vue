<script setup lang="ts">
/**
 * Rule sets that could not be consulted, and why.
 *
 * Two things this fixes, both learned from a real support question.
 *
 * It shows `detail`. The reason key alone says "the sing-box cache file could
 * not be read", which does not distinguish a missing file from a locked one
 * from a path that points somewhere the panel cannot see — and the underlying
 * message ("stat /etc/sing-box/cache.db: no such file or directory") names the
 * path, which is the entire actionable content. The backend was already
 * sending it and both panels dropped it on the floor.
 *
 * It GROUPS by cause. A config with fourteen remote rule sets and no cache
 * file produced fourteen identical lines for one problem, which reads as
 * fourteen problems and buries the single sentence that explains all of them.
 *
 * Shared by the DNS and route probes because they were rendering the same
 * block from the same payload in two places — the drift was already starting.
 */
import { computed } from 'vue'
import { groupRuleSetIssues } from '../utils/ruleSetIssues'
import type { RuleSetStatus } from '../types/ruleSet'

const props = defineProps<{ sets: RuleSetStatus[] }>()

// The grouping is a pure function in utils/ so it can be tested; this file
// stays markup only, the way the flow diagram's model and layout are split.
const issues = computed(() => groupRuleSetIssues(props.sets))
</script>

<template>
  <section
    v-if="issues.length"
    class="rounded-surface border border-amber-300 dark:border-amber-800 bg-amber-50/60 dark:bg-amber-950/20 p-3 space-y-2"
  >
    <p class="text-xs font-semibold text-amber-800 dark:text-amber-200">
      {{ $t('ruleSet.unavailable') }}
    </p>

    <div v-for="issue in issues" :key="issue.key" class="space-y-1">
      <p class="text-xs text-amber-700 dark:text-amber-300">
        {{ $t(`ruleSet.reason.${issue.reason}`) }}
      </p>
      <!--
        The underlying message, verbatim. It names the path, which is what
        turns "could not read the cache" into something actionable.
      -->
      <p
        v-if="issue.detail"
        class="text-[10px] font-mono text-amber-600 dark:text-amber-400 break-all"
      >
        {{ issue.detail }}
      </p>
      <div class="flex flex-wrap gap-1">
        <code
          v-for="tag in issue.tags"
          :key="tag"
          class="px-1.5 py-0.5 rounded-pill bg-amber-100 dark:bg-amber-900/40 text-[10px] text-amber-800 dark:text-amber-200"
        >
          {{ tag }}
        </code>
      </div>
    </div>
  </section>
</template>
