<script setup lang="ts">
/**
 * Flow diagram of the current DNS routing logic.
 *
 * sing-box evaluates `dns.rules` top-down and stops at the first match, so the
 * configuration is a decision ladder rather than an arbitrary graph. Rendering
 * it as a ladder — query enters at the top, falls through each rung, exits into
 * `dns.final` — matches the runtime semantics exactly and needs no graph
 * library. That keeps the embedded frontend small, which matters on the OpenWrt
 * builds where the whole binary has to fit in the router's overlay.
 *
 * WHAT THIS USED TO THROW AWAY
 * ────────────────────────────
 * `probe.attribution.rules[]` has always carried a verdict per rule — `state`,
 * a rendered `summary`, and the conditions that could not be decided — and this
 * component ignored all of it. It re-derived its own summary from the raw config
 * (a second `summarize()` that could drift from the backend's), re-derived its
 * own list of runtime-only fields, and then highlighted exactly ONE rung. A rule
 * the query was tested against and failed rendered identically to a rule that
 * was never reached, which are opposite facts about the same lookup.
 *
 * The per-rule data now drives the ladder, and the ladder's lamps distinguish
 * the four outcomes. The standalone case — no probe yet — still renders from the
 * config, because the diagram has to be readable before anything is probed; that
 * path is the only reason a local `summarize` survives.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import RuleLadder from './RuleLadder.vue'
import { markUnreached, type LadderRung } from '../types/ruleLadder'
import type { DnsProbeResult } from '../types/dnsprobe'

const props = defineProps<{
  /** The `dns` section of the sing-box config. */
  dns: any | null
  /** Optional probe to highlight against the ladder. */
  probe?: DnsProbeResult | null
  /**
   * Changes once per probe run, so the walk restarts even when the new result
   * is identical to the last — see useRuleSequencer.
   */
  runToken?: number
}>()

const { t } = useI18n()

/**
 * Renders a rule's conditions compactly.
 *
 * Used ONLY when there is no probe. Once one arrives the backend's own summary
 * wins, so the diagram and the probe panel cannot describe the same rule two
 * different ways.
 */
const summarize = (rule: Record<string, any>): string => {
  const parts: string[] = []
  const list = (key: string) => {
    const value = rule[key]
    if (value === undefined || value === null) return
    const values = Array.isArray(value) ? value : [value]
    if (values.length === 0) return
    parts.push(values.length > 3
      ? `${key}=[${values.slice(0, 3).join(' ')} ...+${values.length - 3}]`
      : `${key}=${values.length === 1 ? values[0] : `[${values.join(' ')}]`}`)
  }

  for (const key of ['domain', 'domain_suffix', 'domain_keyword', 'domain_regex', 'rule_set', 'geosite', 'geoip', 'ip_cidr', 'outbound']) {
    list(key)
  }
  for (const flag of ['ip_accept_any', 'ip_is_private', 'invert']) {
    if (rule[flag]) parts.push(`${flag}=true`)
  }
  if (rule.clash_mode) parts.push(`clash_mode=${rule.clash_mode}`)

  return parts.length ? parts.join(' ') : t('dnsFlow.noConditions')
}

/** Condition keys the panel cannot evaluate without sing-box's runtime state. */
const RUNTIME_ONLY = [
  'rule_set', 'geosite', 'geoip', 'source_geoip', 'ip_cidr', 'ip_is_private',
  'ip_accept_any', 'source_ip_cidr', 'inbound', 'outbound', 'clash_mode',
  'process_name', 'process_path', 'package_name', 'user', 'auth_user',
  'protocol', 'network', 'port', 'port_range', 'query_type',
]

const servers = computed<Record<string, any>[]>(() => props.dns?.servers ?? [])
const finalServer = computed<string>(() => props.dns?.final ?? '')
const finalStrategy = computed<string>(() => props.dns?.strategy ?? '')

/**
 * Which rung handled the query.
 *
 * sing-box's OWN logged decision wins over the offline reconstruction — it is
 * the only source here that observed the lookup rather than predicting it.
 */
const matchedIndex = computed(() => {
  const probe = props.probe
  if (!probe) return -1
  const logged = probe.logged_matches?.[probe.logged_matches.length - 1]
  if (logged && logged.config_index >= 0) return logged.config_index
  return probe.attribution?.matched_index ?? -1
})

const highlightIsConfirmed = computed(() => {
  const logged = props.probe?.logged_matches ?? []
  const last = logged[logged.length - 1]
  return Boolean(last && last.config_index >= 0)
})

/** Tag → server, so a rung can show what it routes to. */
const serverByTag = computed(() => {
  const map = new Map<string, Record<string, any>>()
  for (const server of servers.value) {
    if (server.tag) map.set(server.tag, server)
  }
  return map
})

const describeServer = (tag: string) => {
  const server = serverByTag.value.get(tag)
  if (!server) return ''
  const address = server.server ? `${server.server}${server.server_port ? ':' + server.server_port : ''}` : ''
  return [server.type, address].filter(Boolean).join(' ')
}

/** "route(local)" / "reject" — what the rung does, in one token. */
const outcomeOf = (action: string, server: string, strategy: string) => {
  const head = action === 'route' && server ? `${action}(${server})` : action
  return strategy ? `${head} · ${strategy}` : head
}

/**
 * The ladder.
 *
 * With a probe, every rung's verdict comes from the backend. Without one, the
 * rungs render from the config with no verdict at all — `unevaluated` rather
 * than a fabricated "no match", because nothing has been compared yet.
 */
const rungs = computed<LadderRung[]>(() => {
  const probe = props.probe
  const configRules: Record<string, any>[] = props.dns?.rules ?? []

  if (!probe?.attribution?.rules?.length) {
    return configRules.map((rule, index) => ({
      index,
      state: 'unevaluated' as const,
      summary: summarize(rule),
      outcome: outcomeOf(rule.action || 'route', rule.server ?? '', rule.strategy ?? ''),
      deciding: false,
      unevaluated: RUNTIME_ONLY.filter((key) => {
        const value = rule[key]
        return Array.isArray(value) ? value.length > 0 : Boolean(value)
      }),
    }))
  }

  const mapped: LadderRung[] = probe.attribution.rules.map((rule) => ({
    index: rule.index,
    state: rule.state,
    summary: rule.summary,
    outcome: outcomeOf(rule.action, rule.server ?? '', rule.strategy ?? ''),
    deciding: rule.index === matchedIndex.value,
    unevaluated: rule.unevaluated,
  }))

  // Everything below the decision was never consulted — see ruleLadder.ts.
  return markUnreached(mapped, matchedIndex.value)
})

/** A probe ran and nothing matched, so `dns.final` answered it. */
const finalHighlighted = computed(
  () => Boolean(props.probe) && matchedIndex.value < 0,
)
</script>

<template>
  <div>
    <div v-if="!dns || rungs.length === 0" class="text-sm text-gray-500 dark:text-gray-400">
      {{ $t('dnsFlow.empty') }}
    </div>

    <div v-else class="space-y-2">
      <!-- Entry -->
      <div class="flex items-center gap-2">
        <span class="px-3 py-1.5 rounded-pill bg-gray-900 dark:bg-gray-100 text-white dark:text-gray-900 text-xs font-semibold">
          {{ probe ? probe.domain : $t('dnsFlow.query') }}
        </span>
        <span class="text-xs text-gray-500 dark:text-gray-400">{{ $t('dnsFlow.entersRules') }}</span>
        <!-- Whether the highlighted rung is sing-box's own record or our
             reconstruction. The distinction is the whole reason the probe reads
             the log at all, so it is not buried in a tooltip. -->
        <span
          v-if="probe && matchedIndex >= 0"
          class="ml-auto px-2 py-0.5 rounded-pill text-[10px] font-semibold"
          :class="
            highlightIsConfirmed
              ? 'bg-emerald-600 text-white'
              : 'bg-amber-400/25 text-amber-800 dark:text-amber-300'
          "
        >
          {{ highlightIsConfirmed ? $t('dnsFlow.matched') : $t('dnsFlow.predicted') }}
        </span>
      </div>

      <RuleLadder :rungs="rungs" :matched-index="matchedIndex" :run-token="runToken">
        <template #fallthrough="{ revealed }">
          <div
            class="rounded-control border border-dashed px-3 py-2 transition-all duration-200"
            :class="
              finalHighlighted && revealed
                ? 'border-emerald-500 bg-emerald-50/60 dark:bg-emerald-950/30'
                : 'border-gray-300 dark:border-gray-600'
            "
          >
            <div class="flex flex-wrap items-baseline gap-2">
              <span class="text-xs font-semibold text-gray-500 dark:text-gray-400">{{ $t('dnsFlow.final') }}</span>
              <span class="text-sm font-mono font-medium text-gray-800 dark:text-gray-200">
                {{ finalServer || '—' }}
              </span>
              <span v-if="finalServer && describeServer(finalServer)" class="text-xs text-gray-400">
                ({{ describeServer(finalServer) }})
              </span>
              <span v-if="finalStrategy" class="px-1.5 py-0.5 rounded-pill bg-gray-100 dark:bg-gray-700 text-[10px]">
                {{ finalStrategy }}
              </span>
            </div>
          </div>
        </template>
      </RuleLadder>

      <!-- Server legend -->
      <div v-if="servers.length" class="pt-3 mt-3 border-t border-gray-100 dark:border-gray-700">
        <h5 class="text-xs font-semibold text-gray-500 dark:text-gray-400 mb-2">{{ $t('dnsFlow.servers') }}</h5>
        <div class="flex flex-wrap gap-1.5">
          <span
            v-for="server in servers"
            :key="server.tag || server.type"
            class="inline-flex items-center gap-1 px-2 py-1 rounded-pill bg-gray-100 dark:bg-gray-700 text-xs"
          >
            <span class="font-mono font-medium text-gray-800 dark:text-gray-200">{{ server.tag || server.type }}</span>
            <span class="text-gray-500 dark:text-gray-400">{{ describeServer(server.tag) || server.type }}</span>
            <span v-if="server.detour" class="text-primary-600 dark:text-primary-400">via {{ server.detour }}</span>
          </span>
        </div>
      </div>
    </div>
  </div>
</template>
