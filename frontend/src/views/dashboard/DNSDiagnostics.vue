<script setup lang="ts">
/**
 * DNS diagnostics: probe a domain, and see the routing logic it travelled
 * through. The two halves are deliberately on one page — a probe result is
 * most legible when the rule ladder next to it highlights the rung that fired.
 *
 * WHY THE RESULT ANIMATES IN
 * ──────────────────────────
 * Since the probe became a stream, its answer arrives in pieces: the rule
 * attribution lands immediately, the live answer after a network round trip,
 * sing-box's own logged decision after a 250ms settle, and the upstream
 * comparison after a query to every configured resolver. Those gaps are real
 * and sometimes seconds long.
 *
 * Without an entrance, each piece simply blinks into a page the reader is
 * already looking at, and a re-probe of the same domain changes nothing visible
 * at all — the most dangerous case, because it is exactly what "I fixed the
 * rule, let me check again" looks like when the fix did NOT work. The animation
 * is there to say *this is new*, and it is keyed to the run rather than to the
 * data so it says that once per query and not once per phase.
 */
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import DnsProbePanel from '../../components/DnsProbePanel.vue'
import DnsRuleFlow from '../../components/DnsRuleFlow.vue'
import Card from '../../components/Card.vue'
import { dnsService } from '../../services'
import { useNotify } from '../../composables/useNotify'
import type { DnsProbeResult } from '../../types/dnsprobe'

const { t } = useI18n()
const notify = useNotify()

const dns = ref<any | null>(null)
const loading = ref(true)
const probe = ref<DnsProbeResult | null>(null)

/**
 * Which probe run the displayed result belongs to.
 *
 * Passed down to the ladder so its walk restarts on every query. Deriving it
 * from the result instead would fail in both directions: the object identity
 * changes once per streamed phase (four restarts per query), while the content
 * of two consecutive probes of the same domain is identical (no restart at all,
 * leaving a finished ladder from the previous run reading as a fresh answer).
 */
const runToken = ref(0)

onMounted(async () => {
  try {
    const response = await dnsService.getDNS()
    dns.value = response.data
  } catch (err) {
    notify.apiError(err, t('dnsFlow.loadFailed'))
  } finally {
    loading.value = false
  }
})

const onResult = (result: DnsProbeResult | null, token: number) => {
  probe.value = result
  runToken.value = token
}

/**
 * The ladder card's own entrance, played when a probe first attaches to it.
 *
 * Keyed on the run so the card marks the arrival once; the rungs inside then
 * light one at a time on their own clock (useRuleSequencer). Two animations,
 * because they say different things — the card says "this is about a query
 * now", the lamps say "and here is how it was decided".
 */
const flowKey = computed(() => (probe.value ? `probe-${runToken.value}` : 'config'))
</script>

<template>
  <!--
    Both panels are <Card>, not a hand-rolled `bg-white dark:bg-gray-800 shadow`
    block: that spelling paints an opaque white slab on the glass background and
    uses Tailwind's raw `shadow` rather than the `--shadow-surface` tier.
  -->
  <div class="grid grid-cols-1 @container xl:grid-cols-2 gap-3 items-start">
    <Card>
      <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">
        {{ $t('dnsProbe.title') }}
      </h3>
      <p class="text-xs text-gray-500 dark:text-gray-400 mb-3">{{ $t('dnsProbe.desc') }}</p>
      <DnsProbePanel @result="onResult" />
    </Card>

    <Card>
      <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">
        {{ $t('dnsFlow.title') }}
      </h3>
      <p class="text-xs text-gray-500 dark:text-gray-400 mb-3">{{ $t('dnsFlow.desc') }}</p>

      <div v-if="loading" class="flex items-center justify-center py-6">
        <div class="animate-spin rounded-pill h-6 w-6 border-b-2 border-primary-600"></div>
      </div>
      <!--
        `:key` remounts on each run, which re-arms the CSS entrance — a class on
        its own fires only at first mount, so every probe after the first would
        swap the verdicts in silently.
      -->
      <DnsRuleFlow
        v-else
        :key="flowKey"
        class="animate-reveal"
        :dns="dns"
        :probe="probe"
        :run-token="runToken"
      />
    </Card>
  </div>
</template>
