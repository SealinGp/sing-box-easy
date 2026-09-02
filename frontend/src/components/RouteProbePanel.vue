<script setup lang="ts">
/**
 * Route simulator: where would this destination go, if you opened it now?
 *
 * Deliberately PRE-flight. The Clash dashboard already shows where connections
 * went once they exist; this answers the question asked in the minute after a
 * config edit — "did I break something, and should I roll it back?" — while
 * there is still nothing to observe.
 *
 * Nothing here is measured, so the panel never states the verdict as a fact.
 * When a rule ahead of the decision could not be evaluated it says so, names
 * the condition, and offers the fix (start sing-box, download the rule set)
 * rather than reporting the first rule it happened to be able to decide.
 */
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  ArrowPathIcon,
  ArrowRightIcon,
  ExclamationTriangleIcon,
  MagnifyingGlassIcon,
} from '@heroicons/vue/24/outline'
import Button from './Button.vue'
import RuleLadder from './RuleLadder.vue'
import { Select } from '../volt'
import { FILTER_THRESHOLD } from '../utils/selectFilter'
import { routeService, inboundService } from '../services'
import { useNotify } from '../composables/useNotify'
import { markUnreached, type LadderRung } from '../types/ruleLadder'
import {
  ROUTE_NETWORKS,
  SNIFFED_PROTOCOLS,
  type RouteProbeResult,
} from '../types/routeprobe'

const props = withDefaults(defineProps<{ compact?: boolean }>(), { compact: false })

const { t } = useI18n()
const notify = useNotify()

const destination = ref('')
const port = ref<number>(443)
const network = ref<string>('tcp')
const inbound = ref<string | null>('')
const sourceIp = ref('')
const protocol = ref<string | null>('')
const showAdvanced = ref(false)
const running = ref(false)
const result = ref<RouteProbeResult | null>(null)

// Inbound tags, because the condition is frequently decisive: the first rule of
// a typical config keys on it, and leaving it blank makes that rule — and so
// the whole prediction — undecidable.
const inboundTags = ref<string[]>([])

const networkOptions = ROUTE_NETWORKS.map((value) => ({ value, label: value.toUpperCase() }))

// Unset means "do not assume", which leaves `protocol` rules undecidable —
// honest, but it marks every prediction on such a config inexact. Naming the
// protocol is usually all it takes to get an exact answer.
const protocolOptions = SNIFFED_PROTOCOLS.map((value) => ({
  value,
  label: value.toUpperCase(),
}))

const inboundOptions = computed(() =>
  inboundTags.value.map((tag) => ({ value: tag, label: tag })),
)

const canRun = computed(() => destination.value.trim().length > 0 && !running.value)

onMounted(async () => {
  try {
    const { data } = await inboundService.getInbounds()
    inboundTags.value = (data.inbounds || []).map((item) => item.tag || '').filter(Boolean)
  } catch {
    // A missing tag list only costs the convenience of the dropdown; the field
    // stays usable and the probe still runs.
    inboundTags.value = []
  }
})

const run = async () => {
  if (!canRun.value) return
  running.value = true
  try {
    result.value = await routeService.probe({
      destination: destination.value.trim(),
      port: port.value || undefined,
      network: network.value,
      inbound: inbound.value || undefined,
      source_ip: sourceIp.value.trim() || undefined,
      protocol: protocol.value || undefined,
    })
  } catch (err) {
    result.value = null
    notify.apiError(err, t('routeProbe.toast.failed'))
  } finally {
    running.value = false
  }
}

/**
 * The ladder, in the shared shape.
 *
 * The per-rule verdicts, the lamp colours and the sequencing all live in
 * <RuleLadder> now — this panel only says which of ITS fields map onto a rung.
 * The DNS diagram renders the same component, which is the point: the two
 * probes answer the same question and used to answer it in different amounts of
 * detail.
 */
const rungs = computed<LadderRung[]>(() => {
  if (!result.value) return []
  const matchedIndex = result.value.matched_index
  const mapped: LadderRung[] = visibleRules.value.map((rule) => ({
    index: rule.index,
    state: rule.state,
    summary: rule.summary,
    outcome: rule.outcome ? `${rule.action}(${rule.outcome})` : rule.action,
    deciding: rule.index === matchedIndex,
    // A matched rule that does not decide is the confusing case: `sniff` and
    // `resolve` change what the rules below them see and then hand over.
    continues: rule.state === 'matched' && !rule.terminal,
    unevaluated: rule.unevaluated,
  }))
  return markUnreached(mapped, matchedIndex)
})

/**
 * What the ladder shows.
 *
 * The full walk on a real config is 27 rows, which would swamp the Overview
 * page and bury the one line the reader came for. Compact keeps only the rungs
 * that carry information: the rule that decided, plus any that matched without
 * deciding (a `sniff` or `resolve` that changed what the rules below it saw)
 * and any that could not be evaluated — those are the reason the verdict might
 * be wrong, so hiding them would hide the caveat while keeping the claim.
 */
const visibleRules = computed(() => {
  if (!result.value) return []
  if (!props.compact) return result.value.rules
  return result.value.rules.filter(
    (rule) =>
      rule.index === result.value!.matched_index ||
      rule.state === 'matched' ||
      rule.state === 'unevaluated',
  )
})

const hiddenRuleCount = computed(() => {
  if (!result.value) return 0
  return result.value.rules.length - visibleRules.value.length
})

/** Rule sets that could not be read, across every rule. */
const brokenRuleSets = computed(() => {
  if (!result.value) return []
  return result.value.rules
    .flatMap((rule) => rule.rule_sets ?? [])
    .filter((set) => set.reason)
})

const outboundSourceLabel = computed(() => {
  if (!result.value) return ''
  return t(`routeProbe.outboundSource.${result.value.outbound_source}`)
})
</script>

<template>
  <div class="@container space-y-4">
    <!-- Input -->
    <div class="space-y-2">
      <div class="flex flex-col @md:flex-row gap-2">
        <input
          v-model="destination"
          type="text"
          :placeholder="$t('routeProbe.placeholder')"
          class="flex-1 px-2.5 py-1.5 text-sm"
          @keyup.enter="run"
        />
        <Button :disabled="!canRun" @click="run" class="whitespace-nowrap">
          <ArrowPathIcon v-if="running" class="h-4 w-4 animate-spin" />
          <MagnifyingGlassIcon v-else class="h-4 w-4" />
          {{ $t('routeProbe.run') }}
        </Button>
      </div>

      <button
        type="button"
        class="text-xs text-primary-600 dark:text-primary-400 hover:underline"
        @click="showAdvanced = !showAdvanced"
      >
        {{ showAdvanced ? $t('routeProbe.hideAdvanced') : $t('routeProbe.showAdvanced') }}
      </button>

      <!--
        Port, network and inbound are not decoration. Route rules key on all
        three, so a probe that omits them leaves those rules undecidable — the
        defaults stand in for an ordinary browser connection rather than
        leaving the ladder full of "unknown".
      -->
      <div v-if="showAdvanced" class="grid grid-cols-2 @md:grid-cols-5 gap-2">
        <label class="block">
          <span class="block text-xs text-gray-500 dark:text-gray-400 mb-1">{{ $t('routeProbe.fields.port') }}</span>
          <input v-model.number="port" type="number" min="1" max="65535" class="block w-full px-2.5 py-1.5 text-sm" />
        </label>
        <label class="block">
          <span class="block text-xs text-gray-500 dark:text-gray-400 mb-1">{{ $t('routeProbe.fields.network') }}</span>
          <Select v-model="network" :options="networkOptions" optionLabel="label" optionValue="value" class="w-full" />
        </label>
        <label class="block">
          <span class="block text-xs text-gray-500 dark:text-gray-400 mb-1">{{ $t('routeProbe.fields.inbound') }}</span>
          <Select
            v-model="inbound"
            :options="inboundOptions"
            optionLabel="label"
            optionValue="value"
            :filter="inboundOptions.length >= FILTER_THRESHOLD"
            :filterPlaceholder="$t('common.search')"
            :emptyFilterMessage="$t('common.noMatch')"
            :placeholder="$t('routeProbe.anyInbound')"
            showClear
            class="w-full"
          />
        </label>
        <label class="block">
          <span class="block text-xs text-gray-500 dark:text-gray-400 mb-1">{{ $t('routeProbe.fields.protocol') }}</span>
          <Select
            v-model="protocol"
            :options="protocolOptions"
            optionLabel="label"
            optionValue="value"
            filter
            :filterPlaceholder="$t('common.search')"
            :emptyFilterMessage="$t('common.noMatch')"
            :placeholder="$t('routeProbe.unknownProtocol')"
            showClear
            class="w-full"
          />
        </label>
        <label class="block">
          <span class="block text-xs text-gray-500 dark:text-gray-400 mb-1">{{ $t('routeProbe.fields.sourceIp') }}</span>
          <input v-model="sourceIp" type="text" placeholder="192.168.1.10" class="block w-full px-2.5 py-1.5 text-sm" />
        </label>
      </div>
    </div>

    <template v-if="result">
      <!-- Verdict -->
      <div class="rounded-surface border border-gray-200 dark:border-gray-700 p-3">
        <div class="flex flex-wrap items-center gap-2">
          <span class="text-sm text-gray-500 dark:text-gray-400">{{ result.destination }}</span>
          <ArrowRightIcon class="h-4 w-4 text-gray-400" />
          <span class="text-base font-semibold text-gray-900 dark:text-gray-100">{{ result.outbound }}</span>
          <span class="text-xs px-2 py-0.5 rounded-pill bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-300">
            {{ outboundSourceLabel }}
          </span>
        </div>

        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          <span v-if="result.final_used">{{ $t('routeProbe.noRuleMatched') }}</span>
          <span v-else>{{ $t('routeProbe.decidedBy', { index: result.matched_index }) }}</span>
          <span v-if="result.ip">
            · {{ $t('routeProbe.address') }}: {{ result.ip }}
            <span v-if="result.ip_source === 'dns'">({{ $t('routeProbe.resolvedBySingBox') }})</span>
          </span>
        </p>

        <!--
          The honesty line. An unevaluated rule ahead of the decision could have
          matched first, so the verdict above is a best guess and has to say so
          — a confidently wrong answer is worse here than an admitted gap.
        -->
        <div
          v-if="!result.exact"
          class="mt-2 flex items-start gap-2 rounded-control bg-amber-50 dark:bg-amber-950/30 p-2 text-xs text-amber-800 dark:text-amber-200"
        >
          <ExclamationTriangleIcon class="h-4 w-4 flex-shrink-0 mt-0.5" />
          <span>{{ $t('routeProbe.inexact', { count: result.unevaluated_before }) }}</span>
        </div>

        <div
          v-if="result.resolve_error"
          class="mt-2 text-xs text-amber-700 dark:text-amber-300"
        >
          {{ $t('routeProbe.resolveFailed', { error: result.resolve_error }) }}
        </div>
      </div>

      <!-- Rule sets that could not be consulted, with the fix. -->
      <div
        v-if="brokenRuleSets.length"
        class="rounded-surface border border-amber-300 dark:border-amber-800 bg-amber-50/60 dark:bg-amber-950/20 p-3 space-y-1"
      >
        <p class="text-xs font-semibold text-amber-800 dark:text-amber-200">
          {{ $t('routeProbe.ruleSetsUnavailable') }}
        </p>
        <p v-for="set in brokenRuleSets" :key="set.tag" class="text-xs text-amber-700 dark:text-amber-300">
          <code>{{ set.tag }}</code> — {{ $t(`routeProbe.ruleSetReason.${set.reason}`) }}
        </p>
      </div>

      <!-- The ladder. Lamps, states and the top-to-bottom walk all live in
           <RuleLadder>; the DNS diagram renders the same component. -->
      <RuleLadder :rungs="rungs" :matched-index="result.matched_index" :hidden-count="hiddenRuleCount" />
    </template>
  </div>
</template>
