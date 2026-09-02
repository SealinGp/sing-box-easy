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
import { computed, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  ArrowPathIcon,
  ExclamationTriangleIcon,
  MagnifyingGlassIcon,
} from '@heroicons/vue/24/outline'
import Button from './Button.vue'
import Table from './Table.vue'
import DnsProbeTimeline from './DnsProbeTimeline.vue'
import { Select } from '../volt'
import { dnsService } from '../services'
import { useNotify } from '../composables/useNotify'
import {
  DNS_QUERY_TYPES,
  type DnsProbeResult,
  type DnsServerResult,
  type MatchState,
} from '../types/dnsprobe'

const props = withDefaults(defineProps<{ compact?: boolean }>(), { compact: false })

const emit = defineEmits<{
  /**
   * `runToken` changes once per probe run, on the first stage that lands.
   *
   * The result object alone cannot serve as the identity: a streamed probe
   * replaces it once per PHASE, so anything keyed on it would restart four
   * times per run — and two consecutive probes of the same domain produce
   * equal-looking results, so nothing derived from the content can tell them
   * apart either. A counter is the only honest answer.
   */
  (e: 'result', result: DnsProbeResult | null, runToken: number): void
}>()

const { t } = useI18n()
const notify = useNotify()

// Mutable copy: DNS_QUERY_TYPES is a readonly tuple, which Select's
// `options` prop (any[]) will not accept.
const queryTypeOptions = [...DNS_QUERY_TYPES]

const domain = ref('')
const queryType = ref<string>('A')
const compareServers = ref(false)
const running = ref(false)
const result = ref<DnsProbeResult | null>(null)

const canRun = computed(() => domain.value.trim().length > 0 && !running.value)

/**
 * The phase the probe is in, for the progress line.
 *
 * These are the server's own stage names (dnsprobe.Stage) — the phases where
 * the time actually goes — not a decomposition invented here.
 */
const stage = ref<string>('')

/**
 * Bumped when the first data of a run arrives — NOT when the run starts.
 *
 * Starting the clock at submit would re-key the panel while the PREVIOUS
 * result is still on screen (a re-probe deliberately keeps it there rather than
 * blanking), so the old content would play the entrance animation. Waiting for
 * the first stage means the animation always marks the arrival of something
 * new, which is the only thing it is meant to say.
 */
const runToken = ref(0)

let controller: AbortController | null = null
/** Whether this run has already claimed its token. */
let tokenClaimed = false

/** Publishes a result upward, taking a fresh token on the run's first data. */
const publish = (probe: DnsProbeResult | null) => {
  if (!tokenClaimed) {
    tokenClaimed = true
    runToken.value += 1
  }
  result.value = probe
  emit('result', probe, runToken.value)
}

/**
 * Runs the probe over SSE, publishing each phase as it lands.
 *
 * The partial result is emitted upward too, so the rule ladder next to this
 * panel renders from the attribution the moment it arrives — before the live
 * query has returned, and long before compare_servers has finished walking
 * every upstream.
 *
 * Falls back to the unary endpoint if the stream cannot be used: a probe that
 * answers all at once is worse than one that answers progressively, and much
 * better than one that does not answer.
 */
const run = async () => {
  if (!canRun.value) return
  running.value = true
  stage.value = ''
  tokenClaimed = false
  controller?.abort()
  controller = new AbortController()

  const request = {
    domain: domain.value.trim(),
    type: queryType.value,
    compare_servers: compareServers.value,
  }

  try {
    const probe = await dnsService.probeStream(
      request,
      (name, partial) => {
        stage.value = name
        publish(partial)
      },
      controller.signal,
    )
    // A stream that closed without a terminal event told us nothing complete;
    // the unary call is the honest way to finish rather than presenting the
    // last partial as final.
    publish(probe ?? (await dnsService.probe(request)))
  } catch (err) {
    try {
      publish(await dnsService.probe(request))
    } catch {
      publish(null)
      notify.apiError(err, t('dnsProbe.toast.failed'))
    }
  } finally {
    stage.value = ''
    running.value = false
  }
}

onUnmounted(() => controller?.abort())

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

/** Why a configured server was left out of the comparison. */
const skipLabel = (server: DnsServerResult) =>
  server.skip_reason ? t(`dnsProbe.skip.${server.skip_reason}`, { detail: server.skip_detail ?? '' }) : ''

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
  <div class="space-y-3">
    <!-- Query form -->
    <form class="flex flex-wrap items-end gap-2" @submit.prevent="run">
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
          class="block w-full px-2.5 py-1.5 text-sm"
        />
      </div>

      <div>
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1" for="dns-probe-type">
          {{ $t('dnsProbe.type') }}
        </label>
        <Select
          id="dns-probe-type"
          v-model="queryType"
          :options="queryTypeOptions"
          filter
          :filterPlaceholder="$t('common.search')"
          :emptyFilterMessage="$t('common.noMatch')"
        />
      </div>

      <Button type="submit" size="sm" :disabled="!canRun" :loading="running">
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

    <div v-if="result" class="space-y-3">
      <!--
        The lookup as a sequence. This replaces separate "answer" and "routing"
        blocks: read in order they explain how the result was reached, which is
        the whole question being asked.
      -->
      <!--
        Keyed on the run, not on the result: a streamed probe replaces the
        result once per phase, and keying on it would replay the entrance four
        times per query. The key remounts the section, which is what re-arms the
        CSS animation — a class alone only fires on first mount, so a second
        probe would update the numbers in place and silently.
      -->
      <section :key="`timeline-${runToken}`" class="animate-reveal">
        <h4 class="text-sm font-semibold text-gray-900 dark:text-gray-100 mb-2">
          {{ $t('dnsTimeline.title') }}
        </h4>
        <DnsProbeTimeline :result="result" />

        <p v-if="logStatusMessage" class="mt-2 text-xs text-gray-500 dark:text-gray-400">
          {{ logStatusMessage }}
        </p>
      </section>

      <!-- Rule ladder -->
      <section
        v-if="!props.compact && result.attribution.rules.length"
        :key="`rules-${runToken}`"
        class="animate-reveal"
        style="animation-delay: 70ms"
      >
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
      <section v-if="result.servers.length" :key="`upstreams-${runToken}`" class="animate-reveal">
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
        <!-- No header: the three columns are self-describing, so no #head slot. -->
        <Table>
          <tr v-for="server in result.servers" :key="server.tag">
            <td class="font-medium text-gray-700 dark:text-gray-300">
              {{ server.tag }}
              <span class="ml-1 text-xs text-gray-400">{{ server.type }}</span>
            </td>
            <td class="cell-wrap text-gray-600 dark:text-gray-400">
              <span v-if="server.skip_reason" class="text-xs italic">{{ skipLabel(server) }}</span>
              <span v-else-if="server.error" class="text-xs text-red-600 dark:text-red-400 break-all">
                {{ server.error }}
              </span>
              <span v-else class="font-mono text-xs break-all">{{ server.records.join(', ') || '—' }}</span>
            </td>
            <td class="col-actions text-xs text-gray-400">
              <span v-if="!server.skip_reason">{{ server.elapsed_ms }} ms</span>
            </td>
          </tr>
        </Table>
      </section>
    </div>

    <p v-else-if="!running" class="text-sm text-gray-500 dark:text-gray-400">
      {{ $t('dnsProbe.hint') }}
    </p>

    <!--
      Kept visible once a result exists, because a streamed probe HAS a partial
      result while it is still working: the ladder renders off the attribution
      while the live query and the upstream comparison are still outstanding.
      Hiding this at the first partial would claim the probe had finished.
    -->
    <div v-if="running" class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
      <ArrowPathIcon class="h-4 w-4 animate-spin" />
      <span v-if="stage">{{ $t(`dnsProbe.stage.${stage}`) }}</span>
      <span v-else>{{ $t('dnsProbe.running') }}</span>
    </div>
  </div>
</template>
