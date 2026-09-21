<script setup lang="ts">
/**
 * One domain, one address family, read left to right as
 * claimed → predicted → observed.
 *
 * That order is the point. Each column is a different source — sing-box's
 * resolver, the offline route walk, the live connection table — and the value
 * of the row is entirely in where they stop agreeing. A layout that merged
 * them into a single verdict would throw away the only thing worth reading.
 */
import { computed } from 'vue'
import {
  ArrowRightIcon,
  CheckCircleIcon,
  ExclamationTriangleIcon,
  MinusCircleIcon,
  XCircleIcon,
} from '@heroicons/vue/24/outline'
import type { DualStackFamily } from '../types/dualstack'

const props = defineProps<{ family: DualStackFamily }>()

/**
 * Colour by outcome — and `no_address` is deliberately NOT red.
 *
 * A proxied domain with its AAAA suppressed is a correctly configured IPv6
 * split. Painting that like a failure would flag working configs as broken
 * and teach the reader to ignore the column.
 */
const tone = computed(() => {
  switch (props.family.status) {
    case 'reachable':
      return {
        icon: CheckCircleIcon,
        chip: 'bg-emerald-100 dark:bg-emerald-900/40 text-emerald-700 dark:text-emerald-300',
        text: 'text-emerald-600 dark:text-emerald-400',
      }
    case 'unreachable':
      return {
        icon: XCircleIcon,
        chip: 'bg-red-100 dark:bg-red-900/40 text-red-700 dark:text-red-300',
        text: 'text-red-600 dark:text-red-400',
      }
    case 'no_address':
      return {
        icon: MinusCircleIcon,
        chip: 'bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-300',
        text: 'text-gray-500 dark:text-gray-400',
      }
    default:
      return {
        icon: ExclamationTriangleIcon,
        chip: 'bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-300',
        text: 'text-amber-600 dark:text-amber-400',
      }
  }
})

const predicted = computed(() => props.family.predicted?.outbound ?? '')
const observed = computed(() => props.family.observed?.outbound ?? '')

/** A disagreement is the headline finding, so it gets the loud treatment. */
const disagrees = computed(() => props.family.agreement === 'differ')

const agreementClass = computed(() => {
  switch (props.family.agreement) {
    case 'agree':
      return 'bg-emerald-100 dark:bg-emerald-900/40 text-emerald-700 dark:text-emerald-300'
    case 'differ':
      return 'bg-red-600 text-white'
    default:
      return 'bg-gray-100 dark:bg-gray-700 text-gray-500 dark:text-gray-400'
  }
})

/** Extra addresses beyond the one tested, so the row can stay one line. */
const extraAddresses = computed(() => Math.max(0, props.family.addresses.length - 1))

/**
 * Why nothing was observed.
 *
 * Only shown when a dial actually succeeded: with no traffic there is nothing
 * to correlate, and saying "no connection appeared" would read as a fault.
 */
const missingObservation = computed(
  () => props.family.status === 'reachable' && !props.family.observed,
)
</script>

<template>
  <div
    class="rounded-control border px-3 py-2"
    :class="
      disagrees
        ? 'border-red-400 dark:border-red-700 bg-red-50/60 dark:bg-red-950/20'
        : 'border-gray-200 dark:border-gray-700'
    "
  >
    <div class="flex flex-wrap items-center gap-2">
      <component :is="tone.icon" class="h-4 w-4 flex-shrink-0" :class="tone.text" />
      <span class="text-xs font-semibold w-12 text-gray-700 dark:text-gray-300">
        {{ $t(`dualStack.family.${family.family}`) }}
      </span>

      <!-- Claimed: what sing-box resolved. -->
      <span v-if="family.tested" class="font-mono text-xs text-gray-700 dark:text-gray-300 break-all">
        {{ family.tested }}
      </span>
      <span v-else class="text-xs italic text-gray-400">—</span>
      <span v-if="extraAddresses" class="text-[10px] text-gray-400">+{{ extraAddresses }}</span>

      <span class="text-[10px] px-1.5 py-0.5 rounded-pill" :class="tone.chip">
        {{ $t(`dualStack.status.${family.status}`) }}
      </span>
      <span v-if="family.dial?.status === 'reachable'" class="text-[10px] text-gray-400">
        {{ family.dial.elapsed_ms }} ms
      </span>

      <!-- Predicted → observed. -->
      <template v-if="predicted || observed">
        <span class="ml-auto flex items-center gap-1.5 text-xs">
          <span v-if="predicted" class="text-gray-500 dark:text-gray-400">
            {{ predicted }}
          </span>
          <ArrowRightIcon v-if="predicted && observed" class="h-3 w-3 text-gray-400" />
          <span v-if="observed" class="font-semibold text-gray-800 dark:text-gray-200">
            {{ observed }}
          </span>
          <span class="text-[10px] px-1.5 py-0.5 rounded-pill" :class="agreementClass">
            {{ $t(`dualStack.agreement.${family.agreement}`) }}
          </span>
        </span>
      </template>
    </div>

    <!-- The loud case, spelled out rather than left to the two tags above. -->
    <p v-if="disagrees" class="mt-1 text-xs text-red-700 dark:text-red-300">
      {{ $t('dualStack.differHelp', { predicted, observed }) }}
    </p>

    <!--
      Why this row says "unverified" rather than agreeing or disagreeing: an
      undecidable rule sat ahead of the decision.
      Deliberately NOT shown when the two AGREE. Agreement is evidence — the
      traffic went where the config said — and a caveat there is noise on the
      row that needs it least, repeated on every row of a config whose rule
      sets cannot all be read.
    -->
    <p
      v-else-if="family.agreement === 'unknown' && predicted && observed"
      class="mt-1 text-xs text-amber-700 dark:text-amber-300"
    >
      {{ $t('dualStack.inexact') }}
    </p>

    <p v-if="family.status === 'no_address'" class="mt-1 text-xs text-gray-500 dark:text-gray-400">
      {{ $t('dualStack.statusHelp.no_address') }}
    </p>

    <p v-if="family.dns_error" class="mt-1 text-xs text-amber-700 dark:text-amber-300">
      {{ family.dns_error }}
    </p>
    <!--
      The dial's own failure. Suppressed for `untested`, whose only reason is
      that this host lacks the address family — already stated once, at the
      top, in the viewer's language. Repeating it per row would print server
      prose in one language under a translated banner saying the same thing.
    -->
    <p
      v-if="family.dial?.error && family.status !== 'untested'"
      class="mt-1 text-xs break-all"
      :class="tone.text"
    >
      {{ family.dial.error }}
    </p>
    <p v-if="missingObservation" class="mt-1 text-xs text-gray-500 dark:text-gray-400">
      {{ $t('dualStack.noObservation') }}
    </p>

    <!--
      The leaf actually dialled. For a selector or urltest the group name says
      nothing about which node carried the bytes, which is usually the thing
      being asked.
    -->
    <p v-if="family.observed?.via" class="mt-1 text-xs text-gray-500 dark:text-gray-400">
      {{ $t('dualStack.via') }} <span class="font-mono">{{ family.observed.via }}</span>
      <span v-if="family.observed.rule" class="ml-2 font-mono text-[10px] text-gray-400 break-all">
        {{ family.observed.rule }}
      </span>
    </p>
    <p
      v-else-if="family.observed?.rule"
      class="mt-1 font-mono text-[10px] text-gray-400 break-all"
    >
      {{ family.observed.rule }}
    </p>
  </div>
</template>
