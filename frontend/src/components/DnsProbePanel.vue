<script setup lang="ts">
/**
 * DNS route inspector: resolve a domain the way sing-box does, and explain
 * which rule produced the answer.
 *
 * The panel is careful about what it claims. The live answer comes from
 * sing-box itself and is stated as fact. The rule attribution is reconstructed
 * offline and is labelled as a prediction whenever a `rule_set` (or any other
 * condition needing runtime state) sat ahead of the decision — a confidently
 * wrong "rule #4 matched" would be worse than admitting uncertainty.
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  ArrowPathIcon,
  CheckBadgeIcon,
  ExclamationTriangleIcon,
  MagnifyingGlassIcon,
  QuestionMarkCircleIcon,
} from '@heroicons/vue/24/outline'
import Button from './Button.vue'
import { dnsService } from '../services'
import { useNotify } from '../composables/useNotify'
import { DNS_QUERY_TYPES, type DnsProbeResult, type MatchState } from '../types/dnsprobe'

const props = withDefaults(defineProps<{ compact?: boolean }>(), { compact: false })

const emit = defineEmits<{ (e: 'result', result: DnsProbeResult | null): void }>()

const { t } = useI18n()
const notify = useNotify()

const domain = ref('')
const queryType = ref<string>('A')
const compareServers = ref(false)
const running = ref(false)
const result = ref<DnsProbeResult | null>(null)

const canRun = computed(() => domain.value.trim().length > 0 && !running.value)

const run = async () => {
  if (!canRun.value) return
  running.value = true
  try {
    const probe = await dnsService.probe({
      domain: domain.value.trim(),
      type: queryType.value,
      compare_servers: compareServers.value,
    })
    result.value = probe
    emit('result', probe)
  } catch (err) {
    result.value = null
    emit('result', null)
    notify.apiError(err, t('dnsProbe.toast.failed'))
  } finally {
    running.value = false
  }
}

/** The rule the query is attributed to, if any. */
const matchedRule = computed(() => {
  const attribution = result.value?.attribution
  if (!attribution || attribution.matched_index < 0) return null
  return attribution.rules[attribution.matched_index] ?? null
})

/**
 * sing-box's own decision, when debug logging made it available. This is
 * evidence rather than inference, so the UI leads with it when present.
 */
const verifiedMatch = computed(() => {
  const matches = result.value?.logged_matches ?? []
  return matches.length > 0 ? matches[matches.length - 1] : null
})

const stateClass = (state: MatchState) => {
  switch (state) {
    case 'matched':
      return 'border-primary-500 bg-primary-50 dark:bg-primary-950/40'
    case 'unevaluated':
      return 'border-amber-400 bg-amber-50/60 dark:bg-amber-950/20'
    default:
      return 'border-gray-200 dark:border-gray-700 opacity-60'
  }
}

const stateLabel = (state: MatchState) => t(`dnsProbe.state.${state}`)

const answersText = computed(() => result.value?.live?.answers ?? [])

/**
 * The backend reports the log situation as a code so it can be translated
 * here; sending prose would pin it to one language.
 */
const logStatusMessage = computed(() => {
  const status = result.value?.log_status
  if (!status) return ''
  if (status === 'read_error') {
    return t('dnsProbe.log.readError', { error: result.value?.log_error ?? '' })
  }
  return t(`dnsProbe.log.${status}`)
})
</script>

<template>
  <div class="space-y-4">
    <!-- Query form -->
    <form class="flex flex-wrap items-end gap-3" @submit.prevent="run">
      <div class="flex-1 min-w-56">
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1" for="dns-probe-domain">
          {{ $t('dnsProbe.domain') }}
        </label>
        <input
          id="dns-probe-domain"
          v-model="domain"
          type="text"
          autocomplete="off"
          spellcheck="false"
          placeholder="example.com"
          class="w-full rounded-control border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-900 px-3 py-2 text-sm text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
        />
      </div>

      <div>
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1" for="dns-probe-type">
          {{ $t('dnsProbe.type') }}
        </label>
        <select
          id="dns-probe-type"
          v-model="queryType"
          class="rounded-control border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-900 px-3 py-2 text-sm text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
        >
          <option v-for="type in DNS_QUERY_TYPES" :key="type" :value="type">{{ type }}</option>
        </select>
      </div>

      <Button type="submit" :disabled="!canRun" :loading="running">
        <MagnifyingGlassIcon v-if="!running" class="h-4 w-4" />
        {{ $t('dnsProbe.run') }}
      </Button>
    </form>

    <label v-if="!props.compact" class="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400">
      <input
        v-model="compareServers"
        type="checkbox"
        class="rounded border-gray-300 dark:border-gray-600 text-primary-600 focus:ring-primary-500 dark:bg-gray-700"
      />
      <span>{{ $t('dnsProbe.compareServers') }}</span>
    </label>

    <div v-if="result" class="space-y-4">
      <!-- Live answer: ground truth from sing-box -->
      <section>
        <h4 class="text-sm font-semibold text-gray-900 dark:text-gray-100 mb-2">
          {{ $t('dnsProbe.answer') }}
        </h4>

        <p v-if="result.live_error" class="text-sm text-amber-600 dark:text-amber-400">
          {{ result.live_error }}
        </p>

        <div v-else-if="answersText.length === 0" class="text-sm text-gray-500 dark:text-gray-400">
          {{ $t('dnsProbe.noAnswer') }}
        </div>

        <ul v-else class="space-y-1">
          <li
            v-for="(answer, index) in answersText"
            :key="index"
            class="flex flex-wrap items-baseline gap-2 text-sm font-mono"
          >
            <span class="px-1.5 py-0.5 rounded-pill bg-gray-100 dark:bg-gray-700 text-xs">{{ answer.type }}</span>
            <span class="text-gray-900 dark:text-gray-100 font-semibold">{{ answer.data }}</span>
            <span class="text-xs text-gray-400">ttl {{ answer.ttl }}</span>
          </li>
        </ul>
      </section>

      <!-- Where it was routed -->
      <section>
        <h4 class="text-sm font-semibold text-gray-900 dark:text-gray-100 mb-2">
          {{ $t('dnsProbe.routing') }}
        </h4>

        <!--
          When sing-box logged its own decision there is no need to guess, so
          that is shown first and marked as confirmed.
        -->
        <div
          v-if="verifiedMatch"
          class="rounded-control border border-primary-300 dark:border-primary-800 bg-primary-50 dark:bg-primary-950/40 p-3"
        >
          <div class="flex items-center gap-1.5 text-xs font-semibold text-primary-700 dark:text-primary-300 mb-1">
            <CheckBadgeIcon class="h-4 w-4" />
            {{ $t('dnsProbe.confirmedBySingBox') }}
          </div>
          <p class="text-sm text-gray-800 dark:text-gray-200">
            <template v-if="verifiedMatch.config_index >= 0">
              {{ $t('dnsProbe.ruleNumber', { index: verifiedMatch.config_index }) }} —
            </template>
            <span class="font-mono">{{ verifiedMatch.description || $t('dnsProbe.noConditions') }}</span>
          </p>
          <p class="mt-1 text-sm font-mono text-primary-700 dark:text-primary-300">{{ verifiedMatch.action }}</p>
        </div>

        <!-- Otherwise the reconstructed answer, honestly labelled. -->
        <div v-else class="rounded-control border border-gray-200 dark:border-gray-700 p-3">
          <p v-if="matchedRule" class="text-sm text-gray-800 dark:text-gray-200">
            {{ $t('dnsProbe.ruleNumber', { index: matchedRule.index }) }} —
            <span class="font-mono">{{ matchedRule.summary }}</span>
          </p>
          <p v-else class="text-sm text-gray-800 dark:text-gray-200">
            {{ $t('dnsProbe.noRuleMatched') }}
          </p>
          <p class="mt-1 text-sm text-gray-600 dark:text-gray-400">
            {{ $t('dnsProbe.server') }}:
            <span class="font-mono font-semibold">{{ result.attribution.server || '—' }}</span>
            <span v-if="result.attribution.strategy" class="ml-2 text-xs">
              {{ result.attribution.strategy }}
            </span>
          </p>

          <p
            v-if="!result.attribution.exact"
            class="mt-2 flex items-start gap-1.5 text-xs text-amber-700 dark:text-amber-400"
          >
            <QuestionMarkCircleIcon class="h-4 w-4 flex-shrink-0 mt-px" />
            <span>{{ $t('dnsProbe.inexact', { count: result.attribution.unevaluated_before }) }}</span>
          </p>
        </div>

        <p v-if="logStatusMessage" class="mt-2 text-xs text-gray-500 dark:text-gray-400">
          {{ logStatusMessage }}
        </p>
      </section>

      <!-- Rule ladder -->
      <section v-if="!props.compact && result.attribution.rules.length">
        <h4 class="text-sm font-semibold text-gray-900 dark:text-gray-100 mb-2">
          {{ $t('dnsProbe.ruleEvaluation') }}
        </h4>
        <ol class="space-y-1.5">
          <li
            v-for="rule in result.attribution.rules"
            :key="rule.index"
            class="rounded-control border-l-4 px-3 py-2"
            :class="stateClass(rule.state)"
          >
            <div class="flex flex-wrap items-baseline gap-2">
              <span class="text-xs font-mono text-gray-400">#{{ rule.index }}</span>
              <span class="text-xs font-medium" :class="rule.state === 'matched' ? 'text-primary-700 dark:text-primary-300' : 'text-gray-500 dark:text-gray-400'">
                {{ stateLabel(rule.state) }}
              </span>
              <span class="text-sm font-mono text-gray-700 dark:text-gray-300 break-all">{{ rule.summary }}</span>
            </div>
            <div class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
              {{ rule.action }}<template v-if="rule.server">({{ rule.server }})</template>
              <span v-if="rule.unevaluated?.length" class="ml-2 text-amber-600 dark:text-amber-400">
                {{ $t('dnsProbe.cannotEvaluate', { fields: rule.unevaluated.join(', ') }) }}
              </span>
            </div>
          </li>
        </ol>
      </section>

      <!-- Upstream comparison -->
      <section v-if="result.servers.length">
        <h4 class="flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-gray-100 mb-2">
          {{ $t('dnsProbe.upstreams') }}
          <span
            v-if="result.disagreement"
            class="inline-flex items-center gap-1 px-2 py-0.5 rounded-pill bg-red-100 dark:bg-red-900/40 text-xs font-medium text-red-700 dark:text-red-300"
          >
            <ExclamationTriangleIcon class="h-3.5 w-3.5" />
            {{ $t('dnsProbe.disagreement') }}
          </span>
        </h4>
        <div class="overflow-x-auto">
          <table class="min-w-full text-sm">
            <tbody class="divide-y divide-gray-100 dark:divide-gray-700">
              <tr v-for="server in result.servers" :key="server.tag">
                <td class="py-1.5 pr-3 font-medium text-gray-700 dark:text-gray-300 whitespace-nowrap">
                  {{ server.tag }}
                  <span class="ml-1 text-xs text-gray-400">{{ server.type }}</span>
                </td>
                <td class="py-1.5 text-gray-600 dark:text-gray-400">
                  <span v-if="server.skipped" class="text-xs italic">{{ server.skipped }}</span>
                  <span v-else-if="server.error" class="text-xs text-red-600 dark:text-red-400 break-all">
                    {{ server.error }}
                  </span>
                  <span v-else class="font-mono text-xs break-all">{{ server.records.join(', ') || '—' }}</span>
                </td>
                <td class="py-1.5 pl-3 text-right text-xs text-gray-400 whitespace-nowrap">
                  <span v-if="!server.skipped">{{ server.elapsed_ms }} ms</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>

    <p v-else-if="!running" class="text-sm text-gray-500 dark:text-gray-400">
      {{ $t('dnsProbe.hint') }}
    </p>

    <div v-if="running && !result" class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
      <ArrowPathIcon class="h-4 w-4 animate-spin" />
      {{ $t('dnsProbe.running') }}
    </div>
  </div>
</template>
