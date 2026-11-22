<script setup lang="ts">
import TooltipTarget from "./TooltipTarget.vue"
import ProxyAutoSelection from './ProxyAutoSelection.vue'
import ProxyNodeHistory from './ProxyNodeHistory.vue'
/**
 * One group of the running sing-box: its current node, and its members either
 * as a strip of latency blocks (collapsed) or as cards (expanded).
 *
 * The header is the toggle — there is no separate chevron button. A whole row
 * is a bigger target than an icon, and the two controls in it stop their own
 * clicks (a member's own latency badge does the same one level down: it tests
 * that one node, and never switches to it): the latency badge, which IS the test button (the number it shows is
 * the thing a test refreshes, so a second button beside it said the same thing
 * twice), and the magnifier, which widens in place into a search over this
 * card's members and becomes its close button while it is open.
 *
 * BOTH bodies are always rendered and the card animates between them with the
 * `grid-template-rows: 0fr → 1fr` technique. `v-if` would swap them in one
 * frame, and animating `height` needs a measured pixel value that is wrong the
 * moment the grid reflows to another column count.
 */
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { AdjustmentsHorizontalIcon, LockClosedIcon, MagnifyingGlassIcon, XMarkIcon } from '@heroicons/vue/24/outline'
import { FOCUS_PAGES, focusLink } from '../utils/focusTarget'
import { displayName, latencyTier, stripFitsOneLine, tierShares, type LatencyTier } from '../utils/proxyGroups'
import type { RuntimeGroup, RuntimeMember } from '../types/runtime'

const props = defineProps<{
  group: RuntimeGroup
  /** Already filtered and sorted by the page. */
  members: RuntimeMember[]
  open: boolean
  testing: boolean
  /** Names of single nodes with a latency test in flight. */
  testingNodes: ReadonlySet<string>
  /** Name of the member being switched to, '' when idle. */
  switchingTo: string
  /** True when the node-rules engine generates this group. */
  managed?: boolean
  focusedNode?: string
}>()

const emit = defineEmits<{
  (e: 'automaticChanged'): void
  (e: 'toggle'): void
  (e: 'test'): void
  (e: 'testNode', name: string): void
  (e: 'select', name: string): void
}>()

const TIER_BADGE: Record<LatencyTier, string> = {
  good: 'text-green-700 bg-green-100 dark:text-green-300 dark:bg-green-900/40',
  fair: 'text-amber-700 bg-amber-100 dark:text-amber-300 dark:bg-amber-900/40',
  poor: 'text-red-700 bg-red-100 dark:text-red-300 dark:bg-red-900/40',
  none: 'text-gray-500 bg-gray-100 dark:text-gray-400 dark:bg-gray-700/60',
}

const TIER_BLOCK: Record<LatencyTier, string> = {
  good: 'bg-green-500',
  fair: 'bg-amber-500',
  poor: 'bg-red-500',
  none: 'bg-gray-300 dark:bg-gray-600',
}

const { t } = useI18n()

const groupTier = computed(() => latencyTier(props.group.delay))

/**
 * Per-card member search, on top of whatever the page-level search left.
 *
 * Local to the card on purpose: the page search answers "which group has this
 * node", this answers "where in THIS 94-node group is it" without hiding the
 * other groups the operator is comparing against.
 */
const searching = ref(false)
const query = ref('')
const searchInput = ref<HTMLInputElement | null>(null)

const shownMembers = computed(() => {
  const needle = query.value.trim().toLowerCase()
  if (needle === '') return props.members
  // The whole tag, not just the display name, so a fingerprint finds its node.
  return props.members.filter((member) => member.name.toLowerCase().includes(needle))
})

/**
 * Collapsed preview: blocks while they fit on one line, otherwise one bar.
 *
 * A block per member is the better picture — each one is a node, hoverable by
 * name — but only while it stays a single row. Past that the card wraps into
 * a slab several rows tall, towering over its neighbours and saying nothing a
 * proportion would not. So the strip's width is measured, and a group too
 * large for it is drawn as one bar split by latency tier instead.
 */
const strip = ref<HTMLElement | null>(null)
const stripWidth = ref(0)
let stripObserver: ResizeObserver | null = null

onMounted(() => {
  const el = strip.value
  if (!el) return
  // contentRect excludes the padding, which is what the blocks have to fit in.
  stripObserver = new ResizeObserver((entries) => {
    stripWidth.value = entries[0]?.contentRect.width ?? 0
  })
  stripObserver.observe(el)
})
onBeforeUnmount(() => stripObserver?.disconnect())

const asBlocks = computed(() => stripFitsOneLine(shownMembers.value.length, stripWidth.value))
const shares = computed(() => tierShares(shownMembers.value))
const sharesSummary = computed(() =>
  shares.value.map((share) => t('proxies.tierCount', { tier: t(`proxies.tier.${share.tier}`), n: share.count })).join(' · '))

const openSearch = async () => {
  searching.value = true
  // Results belong in the member cards; a search into a collapsed card would
  // only recolour a strip of anonymous blocks.
  if (!props.open) emit('toggle')
  await nextTick()
  // preventScroll: the field is already where the pointer is; focusing it must
  // not also nudge the page.
  searchInput.value?.focus({ preventScroll: true })
}

const closeSearch = () => {
  searching.value = false
  query.value = ''
}

const testNode = (name: string) => {
  if (!props.testingNodes.has(name)) emit('testNode', name)
}

const toggleSearch = () => (searching.value ? closeSearch() : void openSearch())
</script>

<template>
  <article
    class="proxy-group min-w-0 rounded-surface border border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800"
    :class="{ 'is-open': open }"
  >
    <div
      class="group/head flex cursor-pointer select-none flex-wrap items-center gap-x-2.5 gap-y-1 rounded-surface px-3 py-2.5 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary-500"
      role="button"
      tabindex="0"
      :aria-expanded="open"
      @click="emit('toggle')"
      @keydown.enter.self.prevent="emit('toggle')"
      @keydown.space.self.prevent="emit('toggle')"
    >
      <span class="text-sm font-semibold text-gray-900 group-hover/head:text-primary-700 dark:text-gray-100 dark:group-hover/head:text-primary-300">
        {{ group.name }}
      </span>
      <span class="text-[10px] font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
        {{ group.type }} · {{ group.members.length }}
      </span>
      <span v-if="!group.switchable" class="inline-flex items-center gap-1 rounded-pill bg-gray-100 px-2 py-0.5 text-[10px] font-medium text-gray-600 dark:bg-gray-700 dark:text-gray-300">
        <LockClosedIcon class="h-3 w-3" aria-hidden="true" />
        {{ $t('proxies.automaticSelection') }}
      </span>
      <!--
        Where this group comes from. It stops its own click like the other
        controls in the header — the row is the expand toggle.
      -->
      <RouterLink
        v-if="managed"
        :to="focusLink(FOCUS_PAGES.nodeRules, group.name)"
        :title="$t('proxies.fromNodeRulesHint')"
        class="inline-flex items-center gap-1 rounded-pill bg-primary-100 px-2 py-0.5 text-[10px] font-semibold text-primary-700 hover:bg-primary-200 dark:bg-primary-950/40 dark:text-primary-300 dark:hover:bg-primary-900/60"
        @click.stop
        @keydown.enter.stop
        @keydown.space.stop
      >
        <AdjustmentsHorizontalIcon class="h-3 w-3" />
        <span>{{ $t('proxies.fromNodeRules') }}</span>
      </RouterLink>
      <span class="ml-auto flex items-center gap-1.5">
        <ProxyAutoSelection v-if="group.switchable" :group="group" @updated="emit('automaticChanged')" />
        <!--
          The search opens WHERE IT WAS ASKED FOR. The control is anchored on
          its right edge, so the field grows LEFTWARDS out of the button and the
          button itself never moves: the magnifier that was clicked turns, in
          place, into the × that undoes it. One spot opens the search and the
          same spot closes it — the pointer does not have to travel to dismiss
          what it just opened.
        -->
        <span class="group-search" :class="{ 'is-open': searching }" @click.stop>
          <input
            ref="searchInput"
            v-model="query"
            type="text"
            autocomplete="off"
            :tabindex="searching ? 0 : -1"
            :placeholder="$t('proxies.searchInGroupPlaceholder')"
            :aria-label="$t('proxies.searchInGroup', { group: group.name })"
            class="group-search-input"
            @keydown.esc.stop.prevent="closeSearch"
            @keydown.enter.stop
            @keydown.space.stop
          />
          <TooltipTarget as="button"
            type="button"
            class="group-search-toggle"
            :aria-expanded="searching"
            :aria-label="searching ? $t('proxies.closeSearch') : $t('proxies.searchInGroup', { group: group.name })"
            :text="searching ? $t('proxies.closeSearch') : $t('proxies.searchInGroup', { group: group.name })"
            @click="toggleSearch"
          >
            <MagnifyingGlassIcon class="group-search-glyph glyph-search h-3.5 w-3.5" />
            <XMarkIcon class="group-search-glyph glyph-close h-3.5 w-3.5" />
          </TooltipTarget>
        </span>
        <TooltipTarget as="button"
          type="button"
          class="lat-badge lat-button"
          :class="[TIER_BADGE[groupTier], { 'animate-pulse': testing }]"
          :disabled="testing"
          :aria-label="$t('proxies.testGroup', { group: group.name })"
          :text="$t('proxies.testGroup', { group: group.name })"
          @click.stop="emit('test')"
        >
          {{ testing ? $t('proxies.testing') : group.delay > 0 ? $t('proxies.ms', { n: group.delay }) : $t('proxies.noResult') }}
        </TooltipTarget>
      </span>
      <span class="flex min-w-0 basis-full items-center gap-1.5 text-xs text-gray-500 dark:text-gray-400">
        {{ $t('proxies.current') }}
        <TooltipTarget as="span" class="truncate text-gray-900 dark:text-gray-100" :text="group.now">
          {{ group.now ? displayName(group.now) : $t('proxies.noneSelected') }}
        </TooltipTarget>
      </span>
    </div>

    <!--
      Collapsed: one block per member while they fit on a line; one bar split
      by latency tier when they do not. Never both, and never a wrapped slab.
    -->
    <div class="fold fold-strip" :inert="open">
      <div>
        <div ref="strip" class="mx-3 mb-3 flex h-[15px] items-center gap-1 pt-0.5">
          <TooltipTarget as="div" v-if="!asBlocks" class="tier-bar" role="img" :aria-label="sharesSummary" :text="sharesSummary">
            <TooltipTarget as="span"
              v-for="share in shares"
              :key="share.tier"
              :class="TIER_BLOCK[share.tier]"
              :style="{ flexGrow: share.count }"
              :text="$t('proxies.tierCount', { tier: $t(`proxies.tier.${share.tier}`), n: share.count })"
            />
          </TooltipTarget>
          <template v-else>
          <TooltipTarget as="i"
            v-for="member in shownMembers"
            :key="member.name"
            class="h-[11px] w-[11px] flex-none rounded-[3px]"
            :class="[
              TIER_BLOCK[latencyTier(member.delay)],
              member.name === group.now ? 'outline outline-2 outline-offset-1 outline-primary-500' : '',
            ]"
            :text="`${displayName(member.name)} · ${member.delay > 0 ? $t('proxies.ms', { n: member.delay }) : $t('proxies.noResult')}`"
          />
          </template>
          <span v-if="shownMembers.length === 0" class="text-xs text-gray-400">{{ query ? $t('proxies.noMatchInGroup') : $t('proxies.emptyGroup') }}</span>
        </div>
      </div>
    </div>

    <!--
      Expanded: member cards. Buttons only where a click does something.

      Automatic members use dashed outlines and a persistent lock, while
      their latency badges remain independent, available test controls.
    -->
    <div class="fold fold-nodes" :inert="!open">
      <div>
        <div class="group-scroll quiet-scrollbar grid grid-cols-[repeat(auto-fill,minmax(150px,1fr))] gap-1.5 px-3 pt-0.5">
          <TooltipTarget
            :as="group.switchable ? 'button' : 'div'"
            v-for="member in shownMembers"
            :key="member.name"
            :type="group.switchable ? 'button' : undefined"
            :aria-pressed="group.switchable ? member.name === group.now : undefined"
            :disabled="group.switchable ? switchingTo !== '' : undefined"
            :aria-description="group.switchable ? undefined : $t('proxies.hintAutomatic')"
            class="flex min-w-0 flex-col gap-1 rounded-control border px-2.5 py-2 text-left transition-colors"
            :class="[
              member.name === focusedNode ? 'focus-target' : '',
              member.name === group.now
                ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/30'
                : 'border-gray-200 bg-gray-50 dark:border-gray-700 dark:bg-gray-900/40',
              group.switchable ? 'cursor-pointer hover:border-primary-500 disabled:cursor-wait' : 'cursor-default border-dashed',
              // A lighter wash of the SELECTED look, so hovering previews the
              // result of the click. Not on the current node (clicking it does
              // nothing) and not while a switch is already in flight.
              group.switchable && member.name !== group.now
                ? 'enabled:hover:bg-primary-50/70 dark:enabled:hover:bg-primary-900/20'
                : '',
              switchingTo === member.name ? 'animate-pulse' : '',
            ]"
            @click="group.switchable && member.name !== group.now && emit('select', member.name)"
          >
            <span class="flex min-w-0 items-center gap-1.5">
              <TooltipTarget as="span" class="min-w-0 flex-1 truncate text-xs font-medium" :class="group.switchable || member.name === group.now ? 'text-gray-900 dark:text-gray-100' : 'text-gray-600 dark:text-gray-400'" :text="member.name">
                {{ displayName(member.name) }}
              </TooltipTarget>
              <ProxyNodeHistory v-if="!member.group" :name="member.name" :history="member.history" />
            </span>
            <span class="flex items-center justify-between gap-1.5 text-[11px] text-gray-500 dark:text-gray-400">
              <span class="flex min-w-0 items-center gap-1">
                <TooltipTarget v-if="!group.switchable" as="span" tabindex="0" :text="$t('proxies.hintAutomatic')"
                  :aria-label="$t('proxies.hintAutomatic')" class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-control focus-visible:outline-2 focus-visible:outline-primary-500">
                  <LockClosedIcon class="h-3.5 w-3.5" aria-hidden="true" />
                </TooltipTarget>
                <span class="truncate">{{ member.type || $t('proxies.notRunning') }}</span>
              </span>
              <!--
                The badge tests THIS node. A span with a button role, not a
                <button>: in a selector the card around it already is one, and
                a button inside a button is invalid markup whose click the
                parser hands to the outer one. It stops its own click and keys
                so testing a node never also switches to it.
              -->
              <TooltipTarget as="span"
                role="button"
                tabindex="0"
                class="lat-badge lat-button"
                :class="[TIER_BADGE[latencyTier(member.delay)], { 'animate-pulse': testingNodes.has(member.name) }]"
                :aria-disabled="testingNodes.has(member.name)"
                :aria-label="$t('proxies.testNode', { node: displayName(member.name) })"
                :text="$t('proxies.testNode', { node: displayName(member.name) })"
                @click.stop="testNode(member.name)"
                @keydown.enter.stop.prevent="testNode(member.name)"
                @keydown.space.stop.prevent="testNode(member.name)"
              >
                {{ testingNodes.has(member.name) ? $t('proxies.testing') : member.delay > 0 ? $t('proxies.ms', { n: member.delay }) : $t('proxies.noResult') }}
              </TooltipTarget>
            </span>
          </TooltipTarget>
          <span v-if="shownMembers.length === 0" class="col-span-full py-2 text-xs text-gray-400">
            {{ query ? $t('proxies.noMatchInGroup') : $t('proxies.emptyGroup') }}
          </span>
        </div>
      </div>
    </div>
  </article>
</template>

<style scoped src="../style/proxy-group-card.css"></style>
