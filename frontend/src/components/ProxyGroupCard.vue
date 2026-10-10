<script setup lang="ts">
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
import { AdjustmentsHorizontalIcon, MagnifyingGlassIcon, XMarkIcon } from '@heroicons/vue/24/outline'
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
}>()

const emit = defineEmits<{
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
          <button
            type="button"
            class="group-search-toggle"
            :aria-expanded="searching"
            :aria-label="searching ? $t('proxies.closeSearch') : $t('proxies.searchInGroup', { group: group.name })"
            :title="searching ? $t('proxies.closeSearch') : $t('proxies.searchInGroup', { group: group.name })"
            @click="toggleSearch"
          >
            <MagnifyingGlassIcon class="group-search-glyph glyph-search h-3.5 w-3.5" />
            <XMarkIcon class="group-search-glyph glyph-close h-3.5 w-3.5" />
          </button>
        </span>
        <button
          type="button"
          class="lat-badge lat-button"
          :class="[TIER_BADGE[groupTier], { 'animate-pulse': testing }]"
          :disabled="testing"
          :aria-label="$t('proxies.testGroup', { group: group.name })"
          :title="$t('proxies.testGroup', { group: group.name })"
          @click.stop="emit('test')"
        >
          {{ testing ? $t('proxies.testing') : group.delay > 0 ? $t('proxies.ms', { n: group.delay }) : $t('proxies.noResult') }}
        </button>
      </span>
      <span class="flex min-w-0 basis-full items-center gap-1.5 text-xs text-gray-500 dark:text-gray-400">
        {{ $t('proxies.current') }}
        <span class="truncate text-gray-900 dark:text-gray-100" :title="group.now">
          {{ group.now ? displayName(group.now) : $t('proxies.noneSelected') }}
        </span>
      </span>
    </div>

    <!--
      Collapsed: one block per member while they fit on a line; one bar split
      by latency tier when they do not. Never both, and never a wrapped slab.
    -->
    <div class="fold fold-strip" :inert="open">
      <div>
        <div ref="strip" class="mx-3 mb-3 flex h-[15px] items-center gap-1 pt-0.5">
          <div v-if="!asBlocks" class="tier-bar" role="img" :aria-label="sharesSummary" :title="sharesSummary">
            <span
              v-for="share in shares"
              :key="share.tier"
              :class="TIER_BLOCK[share.tier]"
              :style="{ flexGrow: share.count }"
              :title="$t('proxies.tierCount', { tier: $t(`proxies.tier.${share.tier}`), n: share.count })"
            />
          </div>
          <template v-else>
          <i
            v-for="member in shownMembers"
            :key="member.name"
            class="h-[11px] w-[11px] flex-none rounded-[3px]"
            :class="[
              TIER_BLOCK[latencyTier(member.delay)],
              member.name === group.now ? 'outline outline-2 outline-offset-1 outline-primary-500' : '',
            ]"
            :title="`${displayName(member.name)} · ${member.delay > 0 ? $t('proxies.ms', { n: member.delay }) : $t('proxies.noResult')}`"
          />
          </template>
          <span v-if="shownMembers.length === 0" class="text-xs text-gray-400">{{ query ? $t('proxies.noMatchInGroup') : $t('proxies.emptyGroup') }}</span>
        </div>
      </div>
    </div>

    <!--
      Expanded: member cards. Buttons only where a click does something.

      In an automatic group (urltest, fallback) the cards look the same as a
      selector's, so the pointer has to say they are not: `not-allowed` over
      each one, with the reason as its tooltip. A plain default cursor left the
      operator clicking a node and wondering why nothing switched.
    -->
    <div class="fold fold-nodes" :inert="!open">
      <div>
        <div class="group-scroll quiet-scrollbar grid grid-cols-[repeat(auto-fill,minmax(150px,1fr))] gap-1.5 px-3 pt-0.5">
          <component
            :is="group.switchable ? 'button' : 'div'"
            v-for="member in shownMembers"
            :key="member.name"
            :type="group.switchable ? 'button' : undefined"
            :aria-pressed="group.switchable ? member.name === group.now : undefined"
            :disabled="group.switchable ? switchingTo !== '' : undefined"
            :aria-disabled="group.switchable ? undefined : true"
            :title="group.switchable ? undefined : $t('proxies.hintAutomatic')"
            class="flex min-w-0 flex-col gap-1 rounded-control border px-2.5 py-2 text-left transition-colors"
            :class="[
              member.name === group.now
                ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/30'
                : 'border-gray-200 bg-gray-50 dark:border-gray-700 dark:bg-gray-900/40',
              group.switchable ? 'cursor-pointer hover:border-primary-500 disabled:cursor-wait' : 'cursor-not-allowed',
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
            <span class="truncate text-xs font-medium text-gray-900 dark:text-gray-100" :title="member.name">
              {{ displayName(member.name) }}
            </span>
            <span class="flex items-center justify-between gap-1.5 text-[11px] text-gray-500 dark:text-gray-400">
              <span class="truncate">{{ member.type || $t('proxies.notRunning') }}</span>
              <!--
                The badge tests THIS node. A span with a button role, not a
                <button>: in a selector the card around it already is one, and
                a button inside a button is invalid markup whose click the
                parser hands to the outer one. It stops its own click and keys
                so testing a node never also switches to it.
              -->
              <span
                role="button"
                tabindex="0"
                class="lat-badge lat-button"
                :class="[TIER_BADGE[latencyTier(member.delay)], { 'animate-pulse': testingNodes.has(member.name) }]"
                :aria-disabled="testingNodes.has(member.name)"
                :aria-label="$t('proxies.testNode', { node: displayName(member.name) })"
                :title="$t('proxies.testNode', { node: displayName(member.name) })"
                @click.stop="testNode(member.name)"
                @keydown.enter.stop.prevent="testNode(member.name)"
                @keydown.space.stop.prevent="testNode(member.name)"
              >
                {{ testingNodes.has(member.name) ? $t('proxies.testing') : member.delay > 0 ? $t('proxies.ms', { n: member.delay }) : $t('proxies.noResult') }}
              </span>
            </span>
          </component>
          <span v-if="shownMembers.length === 0" class="col-span-full py-2 text-xs text-gray-400">
            {{ query ? $t('proxies.noMatchInGroup') : $t('proxies.emptyGroup') }}
          </span>
        </div>
        <p class="px-3 pb-2.5 pt-2 text-[11px] text-gray-400 dark:text-gray-500">
          {{ group.switchable ? $t('proxies.hintSwitchable') : $t('proxies.hintAutomatic') }}
        </p>
      </div>
    </div>
  </article>
</template>

<style scoped>
/*
 * In-place search: [ field ][ button ], right-anchored. Opening animates the
 * field's WIDTH from zero, so it reads as one control growing rather than one
 * control being swapped for another, and the button keeps its position
 * throughout. The two glyphs are stacked in the button and cross-fade with a
 * quarter turn, which is what says "this is the same button, now meaning
 * close" rather than "a different button appeared here".
 */
.group-search {
  display: inline-flex;
  align-items: center;
  height: 24px;
  border: 1px solid transparent;
  border-radius: 8px;
  transition:
    border-color 0.2s ease,
    background-color 0.2s ease;
}
.group-search.is-open {
  border-color: var(--color-border-dark);
  background: color-mix(in srgb, var(--color-bg-primary) 60%, transparent);
}
.group-search.is-open:focus-within {
  border-color: var(--color-primary);
}
.group-search-toggle {
  position: relative;
  display: inline-flex;
  width: 24px;
  height: 22px;
  flex: none;
  align-items: center;
  justify-content: center;
  border-radius: 7px;
  color: var(--color-text-secondary);
  transition: color 0.15s ease;
}
.group-search-toggle:hover {
  color: var(--color-primary);
}
.group-search-toggle:focus-visible {
  outline: 2px solid var(--color-primary);
  outline-offset: 1px;
}
.group-search-glyph {
  position: absolute;
  transition:
    opacity 0.18s ease,
    transform 0.24s cubic-bezier(0.32, 0.72, 0, 1);
}
.glyph-close {
  opacity: 0;
  transform: rotate(-90deg) scale(0.6);
}
.group-search.is-open .glyph-search {
  opacity: 0;
  transform: rotate(90deg) scale(0.6);
}
.group-search.is-open .glyph-close {
  opacity: 1;
  transform: none;
}
.group-search-input {
  width: 0;
  min-width: 0;
  /*
   * `!important` on these three only. The app's glass theme styles every text
   * input (border, tinted fill, inset highlight) with selectors that outrank a
   * scoped class, which drew a second pill inside this control's own border.
   * The field here is deliberately chromeless: the surrounding box is the
   * control.
   */
  border: 0 !important;
  background: transparent !important;
  box-shadow: none !important;
  padding: 0;
  font-size: 12px;
  color: var(--color-text-primary);
  outline: none;
  opacity: 0;
  transition:
    width 0.26s cubic-bezier(0.32, 0.72, 0, 1),
    opacity 0.16s ease,
    padding 0.26s cubic-bezier(0.32, 0.72, 0, 1);
}
.group-search.is-open .group-search-input {
  width: 136px;
  padding-left: 8px;
  opacity: 1;
}
@media (prefers-reduced-motion: reduce) {
  .group-search-input,
  .group-search-glyph {
    transition: none;
  }
}

/*
 * The proportional bar. Segments grow by member count (`flex-grow`), so their
 * widths are the tiers' shares with no arithmetic here to drift from the data.
 * A 2px surface gap separates them — adjacent fills need a break to read as
 * parts — and a 6px floor keeps a tier with one node in ninety-four visible:
 * a dead node rounded out of the bar is the one the bar was opened to find.
 */
.tier-bar {
  display: flex;
  height: 8px;
  flex: 1;
  gap: 2px;
  overflow: hidden;
  border-radius: 999px;
}
.tier-bar > span {
  min-width: 6px;
  flex-basis: 0;
  flex-shrink: 1;
}

/* The badge is the test button: it has to look pressable, not like a label. */
.lat-button {
  cursor: pointer;
  transition: box-shadow 0.15s ease, filter 0.15s ease;
}
.lat-button:hover:not(:disabled):not([aria-disabled='true']) {
  box-shadow: 0 0 0 1.5px currentColor;
}
.lat-button:focus-visible {
  outline: 2px solid var(--color-primary);
  outline-offset: 2px;
}
.lat-button:disabled,
.lat-button[aria-disabled='true'] {
  cursor: wait;
}

.lat-badge {
  display: inline-block;
  min-width: 46px;
  flex: none;
  border-radius: 999px;
  padding: 1px 7px;
  text-align: center;
  font-size: 11px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

/*
 * A group's members scroll INSIDE the card past this height. A 94-node group
 * expanded in full is taller than the screen and pushes every other group out
 * of reach; capped, the page stays a page of groups. Same 26rem ceiling the
 * Overview cards use, less the header and hint that stay in normal flow.
 * `.quiet-scrollbar` (style/scrollbars.css) shows the thumb only on hover or
 * focus — fading it in and out — so a column of idle cards is not a column of
 * scrollbars, and moving the pointer across them is not a row of blinks.
 */
.group-scroll {
  max-height: 20rem;
  overflow-y: auto;
  overscroll-behavior: contain;
  /* Room for the current member's focus/selection outline at the edges. */
  padding-bottom: 2px;
}

/* The shell's easing, so the card moves like the rest of the app. */
.fold {
  display: grid;
  grid-template-rows: 0fr;
  opacity: 0;
  transition:
    grid-template-rows 0.32s cubic-bezier(0.32, 0.72, 0, 1),
    opacity 0.2s ease;
}
.fold > div {
  min-height: 0;
  overflow: hidden;
}
.proxy-group:not(.is-open) .fold-strip,
.proxy-group.is-open .fold-nodes {
  grid-template-rows: 1fr;
  opacity: 1;
}
@media (prefers-reduced-motion: reduce) {
  .fold {
    transition: none;
  }
}
</style>
