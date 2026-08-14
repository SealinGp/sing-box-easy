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
 * When a probe result is supplied, the rung that handled the query is
 * highlighted so the diagram doubles as an explanation of a concrete lookup.
 */
import { computed } from 'vue'
import { ArrowDownIcon } from '@heroicons/vue/24/outline'
import type { DnsProbeResult } from '../types/dnsprobe'

const props = defineProps<{
  /** The `dns` section of the sing-box config. */
  dns: any | null
  /** Optional probe to highlight against the ladder. */
  probe?: DnsProbeResult | null
}>()

interface Rung {
  index: number
  summary: string
  action: string
  server: string
  strategy: string
  /** Conditions that cannot be decided without sing-box's runtime state. */
  unevaluated: string[]
}

/**
 * Renders a rule's conditions compactly. Mirrors the backend's summary so the
 * diagram and the probe panel describe the same rule the same way, and works
 * standalone so the diagram renders before any probe has run.
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

  return parts.length ? parts.join(' ') : '(no conditions)'
}

/** Condition keys the panel cannot evaluate without sing-box's runtime state. */
const RUNTIME_ONLY = [
  'rule_set', 'geosite', 'geoip', 'source_geoip', 'ip_cidr', 'ip_is_private',
  'ip_accept_any', 'source_ip_cidr', 'inbound', 'outbound', 'clash_mode',
  'process_name', 'process_path', 'package_name', 'user', 'auth_user',
  'protocol', 'network', 'port', 'port_range', 'query_type',
]

const rungs = computed<Rung[]>(() => {
  const rules: Record<string, any>[] = props.dns?.rules ?? []
  return rules.map((rule, index) => ({
    index,
    summary: summarize(rule),
    action: rule.action || 'route',
    server: rule.server ?? '',
    strategy: rule.strategy ?? '',
    unevaluated: RUNTIME_ONLY.filter((key) => {
      const value = rule[key]
      return Array.isArray(value) ? value.length > 0 : Boolean(value)
    }),
  }))
})

const servers = computed<Record<string, any>[]>(() => props.dns?.servers ?? [])
const finalServer = computed<string>(() => props.dns?.final ?? '')
const finalStrategy = computed<string>(() => props.dns?.strategy ?? '')

/** Index of the rung a probe was attributed to, or -1. */
const highlightedIndex = computed(() => {
  const probe = props.probe
  if (!probe) return -1
  // Prefer sing-box's own logged decision over the reconstruction.
  const logged = probe.logged_matches?.[probe.logged_matches.length - 1]
  if (logged && logged.config_index >= 0) return logged.config_index
  return probe.attribution?.matched_index ?? -1
})

const highlightIsConfirmed = computed(() => {
  const logged = props.probe?.logged_matches ?? []
  const last = logged[logged.length - 1]
  return Boolean(last && last.config_index >= 0)
})

const finalHighlighted = computed(
  () => Boolean(props.probe) && highlightedIndex.value < 0,
)

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
      </div>

      <!-- Ladder -->
      <ol class="space-y-1.5">
        <li v-for="rung in rungs" :key="rung.index">
          <div class="flex items-stretch gap-2">
            <div class="flex flex-col items-center pt-1">
              <ArrowDownIcon class="h-3.5 w-3.5 text-gray-300 dark:text-gray-600" />
            </div>

            <div
              class="flex-1 rounded-control border px-3 py-2 transition-colors"
              :class="
                rung.index === highlightedIndex
                  ? 'border-primary-500 bg-primary-50 dark:bg-primary-950/40 shadow-sm'
                  : 'border-gray-200 dark:border-gray-700'
              "
            >
              <div class="flex flex-wrap items-baseline gap-x-2 gap-y-1">
                <span class="text-xs font-mono text-gray-400">#{{ rung.index }}</span>
                <span class="text-sm font-mono text-gray-800 dark:text-gray-200 break-all">{{ rung.summary }}</span>
                <span
                  v-if="rung.index === highlightedIndex"
                  class="ml-auto px-2 py-0.5 rounded-pill text-[10px] font-semibold"
                  :class="
                    highlightIsConfirmed
                      ? 'bg-primary-600 text-white'
                      : 'bg-amber-400/25 text-amber-800 dark:text-amber-300'
                  "
                >
                  {{ highlightIsConfirmed ? $t('dnsFlow.matched') : $t('dnsFlow.predicted') }}
                </span>
              </div>

              <div class="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs">
                <span class="text-gray-500 dark:text-gray-400">&rarr;</span>
                <span class="font-mono font-medium text-gray-700 dark:text-gray-300">
                  {{ rung.action === 'route' && rung.server ? rung.server : rung.action }}
                </span>
                <span v-if="rung.server && describeServer(rung.server)" class="text-gray-400">
                  ({{ describeServer(rung.server) }})
                </span>
                <span v-if="rung.strategy" class="px-1.5 py-0.5 rounded-pill bg-gray-100 dark:bg-gray-700 text-[10px]">
                  {{ rung.strategy }}
                </span>
                <span v-if="rung.unevaluated.length" class="text-amber-600 dark:text-amber-400">
                  {{ $t('dnsFlow.runtimeOnly', { fields: rung.unevaluated.join(', ') }) }}
                </span>
              </div>
            </div>
          </div>
        </li>
      </ol>

      <!-- Fallthrough -->
      <div class="flex items-stretch gap-2">
        <div class="flex flex-col items-center pt-1">
          <ArrowDownIcon class="h-3.5 w-3.5 text-gray-300 dark:text-gray-600" />
        </div>
        <div
          class="flex-1 rounded-control border border-dashed px-3 py-2"
          :class="
            finalHighlighted
              ? 'border-primary-500 bg-primary-50 dark:bg-primary-950/40'
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
      </div>

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
