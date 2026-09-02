<script setup lang="ts">
/**
 * Clash API + V2Ray API at a glance, for the Overview page.
 *
 * Exists for one workflow: the Clash dashboard is where a live connection is
 * inspected — which rule it matched, which outbound it took, whether it is
 * still open — and that is the first thing reached for when something is
 * wrong. Getting there used to mean Experimental → Clash API tab → the small
 * link beside the address field, which is three navigations away from the page
 * an operator is already looking at when they notice the problem.
 *
 * The link itself is not a plain `href` to `external_controller`; see
 * utils/clashDashboardUrl for the three ways that URL is wrong. This card and
 * the settings page share those rules rather than each spelling them out.
 */
import { computed, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import type { ClashAPI, V2RayAPI } from '../types/api'
import { experimentalService } from '../services'
import {
  clashDashboardHref,
  hasDashboard as clashHasDashboard,
  isLinkBlockedBySecret,
  resolveControllerEndpoint,
} from '../utils/clashDashboardUrl'
import {
  ArrowTopRightOnSquareIcon,
  Cog6ToothIcon,
  LockClosedIcon,
} from '@heroicons/vue/24/outline'

const CLASH_ROUTE = '/dashboard/experimental/clash-api'
const V2RAY_ROUTE = '/dashboard/experimental/v2ray-api'

const loading = ref(true)
const clash = ref<ClashAPI | null>(null)
const v2ray = ref<V2RayAPI | null>(null)

const pageHost = () => window.location.hostname

/** The address a browser would actually dial, wildcard binds resolved. */
const controllerEndpoint = computed(() =>
  resolveControllerEndpoint(clash.value?.external_controller, pageHost()),
)

const controllerLabel = computed(() => {
  const endpoint = controllerEndpoint.value
  return endpoint ? `${endpoint.host}:${endpoint.port}` : ''
})

const href = computed(() => (clash.value ? clashDashboardHref(clash.value, pageHost()) : ''))
const hasDashboard = computed(() => !!clash.value && clashHasDashboard(clash.value))
const blockedBySecret = computed(
  () => !!clash.value && isLinkBlockedBySecret(clash.value, pageHost()),
)
const hasSecret = computed(() => !!clash.value?.secret)
const defaultMode = computed(() => clash.value?.default_mode || '')

const v2rayListen = computed(() => v2ray.value?.listen?.trim() || '')
const v2rayStats = computed(() => !!v2ray.value?.stats?.enabled)

/**
 * Both fetches are independent and each swallows its own failure.
 *
 * A missing section is not an error here: `experimental.clash_api` is optional
 * in sing-box, and an operator who never configured V2Ray API should see "not
 * configured", not a toast. The card is read-only, so there is nothing a
 * failure would corrupt — it just renders the empty state.
 */
async function load() {
  loading.value = true
  const [clashResult, v2rayResult] = await Promise.allSettled([
    experimentalService.getClashAPI(),
    experimentalService.getV2RayAPI(),
  ])
  clash.value = clashResult.status === 'fulfilled' ? (clashResult.value.data ?? null) : null
  v2ray.value = v2rayResult.status === 'fulfilled' ? (v2rayResult.value.data ?? null) : null
  loading.value = false
}

onMounted(load)
</script>

<template>
  <div class="bg-white dark:bg-slate-800 p-4 rounded-surface shadow-surface">
    <div class="flex items-start justify-between gap-3 mb-3">
      <h3 class="text-lg font-semibold text-gray-700 dark:text-gray-300">
        {{ $t('overview.apis.title') }}
      </h3>
      <RouterLink
        :to="CLASH_ROUTE"
        class="inline-flex items-center gap-1 text-xs font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300 transition-colors flex-shrink-0"
      >
        <Cog6ToothIcon class="h-3.5 w-3.5" />
        {{ $t('overview.apis.settings') }}
      </RouterLink>
    </div>

    <div v-if="loading" class="flex items-center justify-center py-6">
      <div class="animate-spin rounded-pill h-6 w-6 border-b-2 border-primary-600"></div>
    </div>

    <div v-else class="space-y-4">
      <!-- ── Clash API ─────────────────────────────────────────────────── -->
      <section class="space-y-2">
        <div class="flex items-center justify-between gap-2">
          <span class="text-xs font-semibold uppercase tracking-wide text-gray-400 dark:text-gray-500">
            {{ $t('overview.apis.clash') }}
          </span>
          <span
            v-if="hasSecret"
            class="inline-flex items-center gap-1 text-[11px] text-gray-400 dark:text-gray-500"
            :title="$t('overview.apis.secretSet')"
          >
            <LockClosedIcon class="h-3 w-3" />
            {{ $t('overview.apis.secretSet') }}
          </span>
        </div>

        <template v-if="controllerLabel">
          <p class="font-mono text-sm text-gray-900 dark:text-gray-100 break-all">
            {{ controllerLabel }}
          </p>
          <p v-if="defaultMode" class="text-xs text-gray-500 dark:text-gray-400">
            {{ $t('overview.apis.mode', { mode: defaultMode }) }}
          </p>

          <!-- The reason this card exists: one click to the connection view. -->
          <a
            v-if="href"
            :href="href"
            target="_blank"
            rel="noopener noreferrer"
            class="inline-flex w-full items-center justify-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-control bg-primary-600 text-white hover:bg-primary-700 transition-colors"
          >
            <ArrowTopRightOnSquareIcon class="h-4 w-4" />
            {{ hasDashboard ? $t('overview.apis.openDashboard') : $t('overview.apis.openController') }}
          </a>

          <!--
            No link is rendered when a secret guards a controller with no
            dashboard: sing-box wants an Authorization header that a navigation
            cannot send, so any URL here would land on 401.
          -->
          <p v-else-if="blockedBySecret" class="text-xs text-amber-600 dark:text-amber-400">
            {{ $t('experimental.clash.secretBlocksDirectLink') }}
          </p>
        </template>

        <p v-else class="text-xs text-gray-500 dark:text-gray-400">
          {{ $t('overview.apis.clashUnset') }}
          <RouterLink :to="CLASH_ROUTE" class="text-primary-600 dark:text-primary-400 hover:underline">
            {{ $t('overview.apis.configure') }}
          </RouterLink>
        </p>
      </section>

      <!-- ── V2Ray API ─────────────────────────────────────────────────── -->
      <section class="space-y-1 border-t border-gray-200 dark:border-gray-700 pt-3">
        <div class="flex items-center justify-between gap-2">
          <span class="text-xs font-semibold uppercase tracking-wide text-gray-400 dark:text-gray-500">
            {{ $t('overview.apis.v2ray') }}
          </span>
          <span
            v-if="v2rayListen"
            class="text-[11px]"
            :class="v2rayStats ? 'text-green-600 dark:text-green-400' : 'text-gray-400 dark:text-gray-500'"
          >
            {{ v2rayStats ? $t('overview.apis.statsOn') : $t('overview.apis.statsOff') }}
          </span>
        </div>

        <template v-if="v2rayListen">
          <p class="font-mono text-sm text-gray-900 dark:text-gray-100 break-all">{{ v2rayListen }}</p>
          <!-- No link on purpose: the V2Ray API is gRPC, so a browser cannot
               open it the way it opens the Clash dashboard. -->
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ $t('overview.apis.v2rayHint') }}</p>
        </template>

        <p v-else class="text-xs text-gray-500 dark:text-gray-400">
          {{ $t('overview.apis.v2rayUnset') }}
          <RouterLink :to="V2RAY_ROUTE" class="text-primary-600 dark:text-primary-400 hover:underline">
            {{ $t('overview.apis.configure') }}
          </RouterLink>
        </p>
      </section>
    </div>
  </div>
</template>
