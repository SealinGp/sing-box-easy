<script setup lang="ts">
/**
 * The life of one DNS lookup, as a timeline.
 *
 * The probe result is a set of independent facts — an answer, a rule verdict,
 * a server, some upstream comparisons. Read as separate blocks they do not say
 * how the lookup unfolded; laid out in order they do: the query entered the
 * rule list, some rules were checked, one decided, a server (or a predefined
 * answer) produced the records.
 *
 * Each step carries its own confidence. A step sourced from sing-box's own
 * decision log is marked confirmed; one reconstructed offline is marked
 * predicted. The marker colour follows that, so uncertainty is visible at a
 * glance rather than buried in a footnote.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  ArrowRightCircleIcon,
  CheckCircleIcon,
  ExclamationTriangleIcon,
  MagnifyingGlassIcon,
  QuestionMarkCircleIcon,
  ServerStackIcon,
} from '@heroicons/vue/24/outline'
import { Timeline } from '../volt'
import type { DnsProbeResult } from '../types/dnsprobe'

const props = defineProps<{ result: DnsProbeResult }>()

const { t } = useI18n()

/** How much the UI can vouch for a step. */
type Confidence = 'confirmed' | 'predicted' | 'problem' | 'neutral'

interface Step {
  key: string
  icon: any
  confidence: Confidence
  title: string
  /**
   * The config key this step came from, e.g. `dns.rules[0]` or `dns.final`.
   * Not translated — it is a literal path into the user's config.
   *
   * This is what makes the timeline stand on its own: the Overview card shows
   * it without the rule ladder or the flow diagram beside it, so a step that
   * cannot name its own source leaves the reader with nowhere to look.
   */
  source?: string
  /** Primary value, rendered monospace. */
  detail?: string
  /** Secondary explanation. */
  note?: string
  /** Extra rows, e.g. one per answer record. */
  records?: string[]
}

/** sing-box's own decision for this query, when debug logging captured it. */
const loggedMatch = computed(() => {
  const matches = props.result.logged_matches ?? []
  return matches.length > 0 ? matches[matches.length - 1] : null
})

const steps = computed<Step[]>(() => {
  const result = props.result
  const attribution = result.attribution
  const logged = loggedMatch.value
  const list: Step[] = []

  // 1. The question.
  list.push({
    key: 'query',
    icon: MagnifyingGlassIcon,
    confidence: 'neutral',
    title: t('dnsTimeline.query'),
    detail: `${result.domain} ${result.query_type}`,
  })

  // 2. What the rule list did with it. Counts rather than a per-rule replay —
  //    the rule ladder alongside already shows every rule.
  const rules = attribution.rules ?? []
  const unevaluated = rules.filter((rule) => rule.state === 'unevaluated').length
  list.push({
    key: 'rules',
    icon: ArrowRightCircleIcon,
    confidence: unevaluated > 0 ? 'predicted' : 'neutral',
    title: t('dnsTimeline.rulesChecked', { count: rules.length }, rules.length),
    source: 'dns.rules',
    note: unevaluated > 0 ? t('dnsTimeline.someUnevaluated', { count: unevaluated }) : undefined,
  })

  // 3. The decision, preferring sing-box's own record over the reconstruction.
  if (logged) {
    list.push({
      key: 'decision',
      icon: CheckCircleIcon,
      confidence: 'confirmed',
      title:
        logged.config_index >= 0
          ? t('dnsTimeline.matchedRule', { index: logged.config_index })
          : t('dnsTimeline.matchedRuleUnknownIndex'),
      source: logged.config_index >= 0 ? `dns.rules[${logged.config_index}]` : 'dns.rules',
      detail: logged.description || undefined,
      note: t('dnsTimeline.confirmed'),
    })
  } else if (attribution.matched_index >= 0) {
    const rule = rules[attribution.matched_index]
    list.push({
      key: 'decision',
      icon: attribution.exact ? CheckCircleIcon : QuestionMarkCircleIcon,
      confidence: attribution.exact ? 'neutral' : 'predicted',
      title: t('dnsTimeline.matchedRule', { index: attribution.matched_index }),
      source: `dns.rules[${attribution.matched_index}]`,
      detail: rule?.summary,
      note: attribution.exact ? undefined : t('dnsTimeline.predicted'),
    })
  } else {
    list.push({
      key: 'decision',
      icon: attribution.exact ? ArrowRightCircleIcon : QuestionMarkCircleIcon,
      confidence: attribution.exact ? 'neutral' : 'predicted',
      title: t('dnsTimeline.noRuleMatched'),
      source: 'dns.final',
      detail: t('dnsTimeline.usesFinal'),
      note: attribution.exact ? undefined : t('dnsTimeline.predicted'),
    })
  }

  // 4. Where the answer came from. A predefined/reject action never reaches an
  //    upstream, so claiming a server there would be wrong.
  //
  //    The action is taken from the attributed rule when sing-box's log is not
  //    available. Reading it only from the log left this step missing entirely
  //    for a predefined rule without debug logging — precisely the tea.tparts
  //    case this tool exists to explain.
  const matchedRule = attribution.matched_index >= 0 ? rules[attribution.matched_index] : undefined
  const action = logged?.action || matchedRule?.action || ''
  const isSynthetic = action.startsWith('predefined') || action.startsWith('reject')
  if (isSynthetic) {
    list.push({
      key: 'source',
      icon: ServerStackIcon,
      confidence: 'neutral',
      title: t('dnsTimeline.answeredLocally'),
      source: matchedRule ? `dns.rules[${matchedRule.index}].action` : undefined,
      note: t('dnsTimeline.answeredLocallyNote'),
    })
  } else if (result.resolved_server) {
    const server = result.resolved_server
    // The tag alone does not say where the query went, which is the whole
    // question — so the address is shown beside it.
    const target = server.address ? `${server.tag}  ${server.type} ${server.address}` : server.tag
    // The strategy names the key it came from, so "ipv4_only" is traceable to
    // either the rule that set it or the global dns.strategy default.
    const strategyKey =
      attribution.strategy_source === 'rule' && attribution.matched_index >= 0
        ? `dns.rules[${attribution.matched_index}].strategy`
        : 'dns.strategy'
    const notes = [
      server.found ? '' : t('dnsTimeline.serverMissing'),
      server.detour ? t('dnsTimeline.viaDetour', { detour: server.detour }) : '',
      attribution.strategy ? `${strategyKey} = ${attribution.strategy}` : '',
    ].filter(Boolean)
    list.push({
      key: 'source',
      icon: ServerStackIcon,
      confidence: server.found ? 'neutral' : 'problem',
      title: t('dnsTimeline.sentToServer'),
      source: `dns.servers[${server.tag}]`,
      detail: target,
      note: notes.join(' · ') || undefined,
    })
  }

  // 5. What came back.
  const answers = result.live?.answers ?? []
  if (result.live_error) {
    list.push({
      key: 'answer',
      icon: ExclamationTriangleIcon,
      confidence: 'problem',
      title: t('dnsTimeline.noLiveAnswer'),
      note: result.live_error,
    })
  } else {
    list.push({
      key: 'answer',
      icon: CheckCircleIcon,
      confidence: answers.length > 0 ? 'confirmed' : 'problem',
      title:
        answers.length > 0
          ? t('dnsTimeline.answered', { ms: result.live?.elapsed_ms ?? 0 })
          : t('dnsTimeline.emptyAnswer'),
      records: answers.map((answer) => `${answer.type}  ${answer.data}`),
    })
  }

  // 6. Only when the upstream comparison actually ran.
  const answered = (result.servers ?? []).filter(
    (server) => !server.skip_reason && !server.error && server.records.length > 0,
  )
  if (answered.length > 0) {
    // One resolver cannot agree with anything, so saying so would be
    // meaningless — and worse, would imply a cross-check that never happened.
    const isComparison = answered.length > 1
    list.push({
      key: 'compare',
      icon: result.disagreement ? ExclamationTriangleIcon : CheckCircleIcon,
      confidence: result.disagreement ? 'problem' : 'neutral',
      title: result.disagreement
        ? t('dnsTimeline.upstreamsDisagree')
        : isComparison
          ? t('dnsTimeline.upstreamsAgree', { count: answered.length })
          : t('dnsTimeline.singleUpstream'),
      note: result.disagreement
        ? t('dnsTimeline.disagreeNote')
        : isComparison
          ? undefined
          : t('dnsTimeline.singleUpstreamNote'),
    })
  }

  return list
})

const markerClass = (confidence: Confidence) => {
  switch (confidence) {
    case 'confirmed':
      return 'bg-primary-600 text-white'
    case 'predicted':
      return 'bg-amber-400/25 text-amber-700 dark:text-amber-300'
    case 'problem':
      return 'bg-red-100 dark:bg-red-900/40 text-red-600 dark:text-red-400'
    default:
      return 'bg-gray-100 dark:bg-gray-700 text-gray-500 dark:text-gray-400'
  }
}
</script>

<template>
  <Timeline :value="steps" data-key="key">
    <template #marker="{ item }">
      <span class="flex h-6 w-6 items-center justify-center rounded-pill" :class="markerClass(item.confidence)">
        <component :is="item.icon" class="h-3.5 w-3.5" />
      </span>
    </template>

    <template #content="{ item }">
      <div class="flex flex-wrap items-baseline gap-x-2 gap-y-0.5">
        <p class="text-sm font-medium text-gray-900 dark:text-gray-100">{{ item.title }}</p>
        <!--
          The config key this step came from. Literal, never translated: it is
          a path the reader can search for in their own config.
        -->
        <code
          v-if="item.source"
          class="px-1.5 py-0.5 rounded-control bg-gray-100 dark:bg-gray-700 text-[11px] font-mono text-gray-500 dark:text-gray-400"
        >
          {{ item.source }}
        </code>
      </div>

      <p v-if="item.detail" class="mt-0.5 text-sm font-mono text-gray-700 dark:text-gray-300 break-all">
        {{ item.detail }}
      </p>

      <ul v-if="item.records?.length" class="mt-1 space-y-0.5">
        <li
          v-for="(record, index) in item.records"
          :key="index"
          class="text-sm font-mono font-semibold text-gray-900 dark:text-gray-100 break-all"
        >
          {{ record }}
        </li>
      </ul>

      <p
        v-if="item.note"
        class="mt-0.5 text-xs"
        :class="
          item.confidence === 'problem'
            ? 'text-red-600 dark:text-red-400'
            : item.confidence === 'predicted'
              ? 'text-amber-700 dark:text-amber-400'
              : 'text-gray-500 dark:text-gray-400'
        "
      >
        {{ item.note }}
      </p>
    </template>
  </Timeline>
</template>
