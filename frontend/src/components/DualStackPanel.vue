<script setup lang="ts">
/**
 * Dual-stack domain test.
 *
 * Streamed rather than unary, because the phases carry very different
 * latencies: the config read is instant, each resolve is a round trip, and
 * the traffic phase is a dial plus connection-table polls per address. The
 * DNS answers and the routing predictions are therefore on screen long before
 * any traffic has been sent.
 */
import { computed, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  ArrowPathIcon,
  ExclamationTriangleIcon,
  MagnifyingGlassIcon,
} from '@heroicons/vue/24/outline'
import Button from './Button.vue'
import DualStackFamilyRow from './DualStackFamilyRow.vue'
import { diagnosticsService } from '../services'
import { useNotify } from '../composables/useNotify'
import { MAX_DOMAINS, type DualStackResult } from '../types/dualstack'

const { t } = useI18n()
const notify = useNotify()

const input = ref('')
const port = ref<number>(443)
const tls = ref(true)
const skipDial = ref(false)
const timeoutMs = ref<number>(8000)
const showAdvanced = ref(false)
const running = ref(false)
const stage = ref('')
const result = ref<DualStackResult | null>(null)

let controller: AbortController | null = null

/**
 * Domains as typed: newline or comma separated, both accepted.
 *
 * Pasting a list from a terminal gives newlines; typing one inline gives
 * commas. Accepting only one of them turns a correct list into an empty run.
 */
const domains = computed(() =>
  input.value
    .split(/[\s,]+/)
    .map((entry) => entry.trim())
    .filter(Boolean),
)

const canRun = computed(
  () => domains.value.length > 0 && domains.value.length <= MAX_DOMAINS && !running.value,
)

const run = async () => {
  if (!canRun.value) return
  running.value = true
  stage.value = ''
  controller?.abort()
  controller = new AbortController()

  const request = {
    domains: domains.value,
    port: port.value || undefined,
    tls: tls.value,
    skip_dial: skipDial.value,
    timeout_ms: timeoutMs.value || undefined,
  }

  try {
    const probe = await diagnosticsService.probeStream(
      request,
      (name, partial) => {
        stage.value = name
        result.value = partial
      },
      controller.signal,
    )
    // A stream that closed without a terminal event told us nothing complete;
    // finishing with the unary call is honest, where presenting the last
    // partial as final would not be.
    result.value = probe ?? (await diagnosticsService.probe(request))
  } catch (err) {
    try {
      result.value = await diagnosticsService.probe(request)
    } catch {
      result.value = null
      notify.apiError(err, t('dualStack.toast.failed'))
    }
  } finally {
    stage.value = ''
    running.value = false
  }
}

// The server dials real addresses for as long as this stream is open. An
// abandoned request must stop it, not just stop rendering it.
onUnmounted(() => controller?.abort())

/** How a client's DNS reaches sing-box, in one line. */
const endpointMessage = computed(() => {
  const endpoint = result.value?.dns_endpoint
  if (!endpoint) return ''
  if (endpoint.source === 'inbound') {
    if (!endpoint.address) {
      return t('dualStack.endpoint.unresolved', { inbound: endpoint.inbound ?? '' })
    }
    return t('dualStack.endpoint.inbound', {
      inbound: endpoint.inbound ?? '',
      address: endpoint.address,
    })
  }
  return t(`dualStack.endpoint.${endpoint.source}`)
})

/**
 * A missing hijack is a real finding, not a footnote: nothing else in the
 * report makes sense if client queries never reach sing-box at all.
 */
const endpointIsProblem = computed(() => result.value?.dns_endpoint.source === 'none')

/** The honest SKIP, surfaced once rather than repeated on every row. */
const clientCaveat = computed(() => {
  if (!result.value) return ''
  if (!result.value.client_ipv6) return t('dualStack.client.noIpv6')
  if (!result.value.client_ipv4) return t('dualStack.client.noIpv4')
  return ''
})
</script>

<template>
  <div class="@container space-y-4">
    <!-- Input -->
    <div class="space-y-2">
      <label class="block">
        <span class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
          {{ $t('dualStack.domains') }}
        </span>
        <textarea
          v-model="input"
          rows="3"
          autocomplete="off"
          spellcheck="false"
          :placeholder="$t('dualStack.placeholder')"
          class="block w-full px-2.5 py-1.5 text-sm font-mono"
          @keydown.ctrl.enter="run"
          @keydown.meta.enter="run"
        ></textarea>
        <span class="block mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ $t('dualStack.domainsHint', { max: MAX_DOMAINS }) }}
        </span>
      </label>

      <div class="flex flex-wrap items-center gap-2">
        <Button :disabled="!canRun" @click="run">
          <ArrowPathIcon v-if="running" class="h-4 w-4 animate-spin" />
          <MagnifyingGlassIcon v-else class="h-4 w-4" />
          {{ $t('dualStack.run') }}
        </Button>
        <button
          type="button"
          class="text-xs text-primary-600 dark:text-primary-400 hover:underline"
          @click="showAdvanced = !showAdvanced"
        >
          {{ showAdvanced ? $t('dualStack.hideAdvanced') : $t('dualStack.showAdvanced') }}
        </button>
      </div>

      <div v-if="showAdvanced" class="grid grid-cols-1 @md:grid-cols-4 gap-2 items-end">
        <label class="block">
          <span class="block text-xs text-gray-500 dark:text-gray-400 mb-1">
            {{ $t('dualStack.fields.port') }}
          </span>
          <input v-model.number="port" type="number" min="1" max="65535" class="block w-full px-2.5 py-1.5 text-sm" />
        </label>
        <label class="block">
          <span class="block text-xs text-gray-500 dark:text-gray-400 mb-1">
            {{ $t('dualStack.fields.timeout') }}
          </span>
          <input v-model.number="timeoutMs" type="number" min="500" max="15000" step="500" class="block w-full px-2.5 py-1.5 text-sm" />
        </label>
        <label class="flex items-start gap-2 text-sm text-gray-600 dark:text-gray-400">
          <input v-model="tls" type="checkbox" class="mt-0.5 rounded border-gray-300 dark:border-gray-600 text-primary-600 focus:ring-primary-500 dark:bg-gray-700" />
          <span>
            {{ $t('dualStack.fields.tls') }}
            <span class="block text-xs text-gray-400">{{ $t('dualStack.tlsHelp') }}</span>
          </span>
        </label>
        <label class="flex items-start gap-2 text-sm text-gray-600 dark:text-gray-400">
          <input v-model="skipDial" type="checkbox" class="mt-0.5 rounded border-gray-300 dark:border-gray-600 text-primary-600 focus:ring-primary-500 dark:bg-gray-700" />
          <span>
            {{ $t('dualStack.fields.skipDial') }}
            <span class="block text-xs text-gray-400">{{ $t('dualStack.skipDialHelp') }}</span>
          </span>
        </label>
      </div>
    </div>

    <template v-if="result">
      <!-- Where a client's DNS lands. Nothing below means much without it. -->
      <div
        class="rounded-surface border px-3 py-2 text-xs"
        :class="
          endpointIsProblem
            ? 'border-amber-300 dark:border-amber-800 bg-amber-50/60 dark:bg-amber-950/20 text-amber-800 dark:text-amber-200'
            : 'border-gray-200 dark:border-gray-700 text-gray-600 dark:text-gray-400'
        "
      >
        <span class="font-semibold">{{ $t('dualStack.endpoint.title') }}:</span>
        {{ endpointMessage }}
      </div>

      <div
        v-if="clientCaveat"
        class="flex items-start gap-2 rounded-control bg-amber-50 dark:bg-amber-950/30 p-2 text-xs text-amber-800 dark:text-amber-200"
      >
        <ExclamationTriangleIcon class="h-4 w-4 flex-shrink-0 mt-0.5" />
        <span>{{ clientCaveat }}</span>
      </div>

      <div
        v-if="result.observe_error"
        class="rounded-control bg-gray-50 dark:bg-gray-800/60 p-2 text-xs text-gray-600 dark:text-gray-400"
      >
        {{ $t('dualStack.observeUnavailable', { error: result.observe_error }) }}
      </div>

      <!-- One block per domain, two rows each. -->
      <div v-for="entry in result.domains" :key="entry.domain" class="space-y-1.5">
        <h4 class="text-sm font-semibold text-gray-900 dark:text-gray-100 font-mono break-all">
          {{ entry.domain }}
        </h4>
        <DualStackFamilyRow
          v-for="family in entry.families"
          :key="family.family"
          :family="family"
        />
        <p v-if="entry.error" class="text-xs text-red-600 dark:text-red-400">{{ entry.error }}</p>
      </div>
    </template>

    <p v-else-if="!running" class="text-sm text-gray-500 dark:text-gray-400">
      {{ $t('dualStack.hint') }}
    </p>

    <!--
      Kept visible once a partial result exists: the stream HAS results while
      it is still working, and hiding this at the first partial would claim
      the probe had finished.
    -->
    <div v-if="running" class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
      <ArrowPathIcon class="h-4 w-4 animate-spin" />
      <span v-if="stage">{{ $t(`dualStack.stage.${stage}`) }}</span>
      <span v-else>{{ $t('dualStack.running') }}</span>
    </div>
  </div>
</template>
