<script setup lang="ts">
/**
 * Draws the route topology: inbounds on the left, `route.rules` down the
 * middle, outbounds on the right.
 *
 * WHAT THE PICTURE ASSERTS
 * ────────────────────────
 * Two things a flat rule list cannot say. First, that several rules share one
 * destination — this repo's config reaches `🤖 AI` from three unrelated rules,
 * and a reader scrolling the list sees three entries where sing-box has one
 * outbound. The ribbons converge, and the fan across the node's edge is the OR.
 * Second, that the middle column is ORDERED and stops at the first match, which
 * is why it is drawn as a ladder rather than a cloud.
 *
 * SVG, NOT HTML BOXES
 * ───────────────────
 * Ribbons need curves between two boxes, and getting those from HTML means
 * measuring the DOM on every reflow. A fixed `viewBox` gives the whole picture
 * one coordinate system that `utils/flowLayout.ts` computes without a DOM at
 * all, and the browser scales it. Text is fitted by estimate rather than
 * measured (`fitToBox`), so every label also carries a `<title>` with the
 * untruncated value.
 *
 * Colour is carried by Tailwind `fill-*`/`stroke-*` utilities so both themes
 * come from the same classes; nothing here hard-codes a hex.
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { FONT, NODE_PAD, fitToBox, layoutRouteFlow } from '../utils/flowLayout'
import { formatCondition } from '../utils/routeTopology'
import type { RouteTopology, TopologyRule } from '../types/routeTopology'
import type { ExitBox, Ribbon, RuleBox } from '../utils/flowLayout'

const props = defineProps<{ topology: RouteTopology }>()

const { t } = useI18n()

const layout = computed(() => layoutRouteFlow(props.topology))

/* ── Hover: one highlight model for all three columns ─────────────────────── */

type Focus =
  | { kind: 'rule'; id: number }
  | { kind: 'exit'; id: string }
  | { kind: 'inbound'; id: string }
  | null

const focus = ref<Focus>(null)

/**
 * Which exit the current hover implicates.
 *
 * Hovering a rule lights its exit, and hovering an exit lights every rule
 * feeding it — the same relation read from either end, which is how someone
 * checks "what else lands here?" without tracing a curve by eye.
 */
const focusedExit = computed<string | null>(() => {
  const current = focus.value
  if (!current) return null
  if (current.kind === 'exit') return current.id
  if (current.kind === 'rule') {
    return props.topology.rules.find((rule) => rule.index === current.id)?.exitId ?? null
  }
  return null
})

const focusedRules = computed<Set<number>>(() => {
  const current = focus.value
  if (!current) return new Set()
  if (current.kind === 'rule') return new Set([current.id])
  if (current.kind === 'exit') {
    const exit = props.topology.exits.find((candidate) => candidate.id === current.id)
    return new Set(exit?.ruleIndices ?? [])
  }
  // An inbound lights the rules scoped to it. Rules with no `inbound` matcher
  // see every inbound, so they are not "the ones this inbound reaches" —
  // highlighting them all would make the scope chip meaningless.
  return new Set(
    props.topology.rules
      .filter((rule) => rule.scopedInbounds.includes(current.id))
      .map((rule) => rule.index),
  )
})

const dimmed = computed(() => focus.value !== null)

const isRibbonLit = (ribbon: Ribbon): boolean => {
  if (!dimmed.value) return false
  if (ribbon.kind === 'inbound') return focus.value?.kind === 'inbound' && focus.value.id === ribbon.inboundTag
  if (ribbon.kind === 'final') return focusedExit.value === ribbon.exitId && focus.value?.kind === 'exit'
  return ribbon.ruleIndex !== undefined && focusedRules.value.has(ribbon.ruleIndex)
}

/* ── Row rendering ────────────────────────────────────────────────────────── */

/** Text budget inside a rule row, after the `#N` gutter and the action badge. */
const RULE_INDEX_GUTTER = 30
const RULE_BADGE_GUTTER = 54

const conditionText = (condition: { key: string; values: string[] }): string =>
  condition.values.length === 0
    ? condition.key
    : `${condition.key}: ${formatCondition(condition)}`

/**
 * One line for a rule row.
 *
 * Spells out every condition when they fit, and degrades to the first plus a
 * count when they do not. Degrading beats letting `fitToBox` clip the joined
 * string: a row ending in `…` says the rule has more conditions but not how
 * many, and "how many" is the part that tells you whether you are looking at
 * the rule you meant to edit. The full sentence is always in the `<title>`.
 */
const ruleLabel = (box: RuleBox): string => {
  const rule = box.rule
  const budget = box.width - RULE_INDEX_GUTTER - RULE_BADGE_GUTTER - NODE_PAD
  if (rule.conditions.length === 0) return t('routeFlow.anyTraffic')

  const full = rule.conditions.map(conditionText).join(' · ')
  if (fitToBox(full, budget, FONT.rule) === full) return full

  const rest = rule.conditions.length - 1
  const short = conditionText(rule.conditions[0]!)
  return fitToBox(rest > 0 ? `${short} +${rest}` : short, budget, FONT.rule)
}

/** The whole rule as a sentence, for the tooltip. Conditions are AND'd. */
const ruleTitle = (rule: TopologyRule): string => {
  const when =
    rule.conditions.length === 0
      ? t('routeFlow.anyTraffic')
      : rule.conditions.map(conditionText).join(` ${t('routeFlow.and')} `)
  const then = rule.exitId ?? `${rule.action}${rule.outcome ? `: ${rule.outcome}` : ''}`
  const lines = [
    `#${rule.index} ${t('routeFlow.when')} ${rule.inverted ? `${t('routeFlow.not')} ` : ''}${when}`,
    `${t('routeFlow.then')} ${then}`,
  ]
  if (rule.scopedInbounds.length > 0) {
    lines.push(t('routeFlow.scopedTo', { inbounds: rule.scopedInbounds.join(', ') }))
  }
  if (!rule.terminal) lines.push(t('routeFlow.legend.passthrough'))
  if (!rule.reachable) lines.push(t('routeFlow.legend.unreachable'))
  if (rule.catchAll) lines.push(t('routeFlow.catchAll'))
  return lines.join('\n')
}

/** Right-hand badge: the action, but only when it is not the plain default. */
const ruleBadge = (rule: TopologyRule): string => (rule.action === 'route' ? '' : rule.action)

const ruleClass = (rule: TopologyRule): string => {
  if (!rule.reachable) return 'fill-gray-100 stroke-gray-300 dark:fill-slate-800 dark:stroke-slate-700'
  if (!rule.terminal) return 'fill-slate-50 stroke-slate-300 dark:fill-slate-800/60 dark:stroke-slate-600'
  if (rule.catchAll) return 'fill-amber-50 stroke-amber-400 dark:fill-amber-950/40 dark:stroke-amber-600'
  return 'fill-white stroke-gray-300 dark:fill-slate-800 dark:stroke-slate-600'
}

/* ── Exit rendering ───────────────────────────────────────────────────────── */

const exitClass = (box: ExitBox): string => {
  switch (box.exit.kind) {
    case 'missing':
      return 'fill-red-50 stroke-red-400 dark:fill-red-950/40 dark:stroke-red-600'
    case 'reject':
      return 'fill-rose-50 stroke-rose-300 dark:fill-rose-950/30 dark:stroke-rose-700'
    case 'hijack-dns':
      return 'fill-violet-50 stroke-violet-300 dark:fill-violet-950/30 dark:stroke-violet-700'
    case 'implicit':
      return 'fill-slate-50 stroke-slate-300 dark:fill-slate-800/60 dark:stroke-slate-600'
    default:
      return 'fill-white stroke-primary-300 dark:fill-slate-800 dark:stroke-primary-700'
  }
}

/**
 * The second line of an exit node: what kind of outbound it is, and how many
 * members a group has. A `urltest` with 6 nodes and a single shadowsocks server
 * behave very differently, and the tag alone does not say which one this is.
 */
const exitMeta = (box: ExitBox): string => {
  const exit = box.exit
  if (exit.kind === 'missing') return t('routeFlow.exitKind.missing')
  if (exit.kind === 'reject') return t('routeFlow.exitKind.reject', { method: exit.detail ?? '' })
  if (exit.kind === 'hijack-dns') return t('routeFlow.exitKind.hijackDns')
  if (exit.kind === 'implicit') return t('routeFlow.exitKind.implicit')
  const parts = [exit.type || 'outbound']
  if (exit.memberCount !== undefined)
    parts.push(t('routeFlow.members', { n: exit.memberCount }, exit.memberCount))
  return fitToBox(parts.join(' · '), box.width - NODE_PAD * 2, FONT.meta)
}

const exitTitle = (box: ExitBox): string => {
  const exit = box.exit
  const lines = [exit.label]
  const reached = exit.ruleIndices.length
  if (reached > 0) {
    lines.push(
      t(
        'routeFlow.reachedBy',
        { n: reached, rules: exit.ruleIndices.map((i) => `#${i}`).join(' ') },
        reached,
      ),
    )
  }
  if (exit.isFinal) lines.push(t(`routeFlow.fallthroughSource.${sourceKey.value}`))
  return lines.join('\n')
}

const sourceKey = computed(() => {
  const source = props.topology.fallthrough.source
  return source === 'first_outbound' ? 'firstOutbound' : source === 'implicit_direct' ? 'implicitDirect' : 'final'
})

const inboundMeta = (box: { type: string; listenPort?: number }): string =>
  box.listenPort ? `${box.type} · :${box.listenPort}` : box.type

const fallthroughLabel = computed(() =>
  fitToBox(
    `${t('routeFlow.everythingElse')} → ${props.topology.fallthrough.exitId}`,
    layout.value.fallthrough.width - RULE_INDEX_GUTTER - NODE_PAD,
    FONT.rule,
  ),
)
</script>

<template>
  <svg
    :viewBox="`0 0 ${layout.width} ${layout.height}`"
    :width="layout.width"
    :height="layout.height"
    class="max-w-full h-auto mx-auto block select-none"
    role="img"
    :aria-label="$t('routeFlow.ariaLabel')"
    @mouseleave="focus = null"
  >
    <!-- Ribbons first, so every node paints over them. -->
    <g fill="none" stroke-linecap="round">
      <path
        v-for="ribbon in layout.ribbons"
        :key="ribbon.id"
        :d="ribbon.d"
        :stroke-width="isRibbonLit(ribbon) ? 2.5 : 1.5"
        :stroke-dasharray="ribbon.kind === 'final' ? '5 4' : undefined"
        :class="[
          ribbon.kind === 'final'
            ? 'stroke-gray-400 dark:stroke-slate-500'
            : 'stroke-primary-400 dark:stroke-primary-600',
          isRibbonLit(ribbon)
            ? 'opacity-100'
            : dimmed
              ? 'opacity-15'
              : 'opacity-60',
        ]"
        class="transition-opacity duration-150"
      />
    </g>

    <!-- Left column: inbounds. -->
    <g
      v-for="box in layout.inbounds"
      :key="`in-${box.tag}-${box.y}`"
      class="cursor-default"
      @mouseenter="focus = { kind: 'inbound', id: box.tag }"
    >
      <title>{{ box.tag || box.type }}</title>
      <rect
        :x="box.x"
        :y="box.y"
        :width="box.width"
        :height="box.height"
        rx="7"
        stroke-width="1"
        class="fill-white stroke-gray-300 dark:fill-slate-800 dark:stroke-slate-600"
        :class="dimmed && !(focus?.kind === 'inbound' && focus.id === box.tag) ? 'opacity-40' : ''"
      />
      <text
        :x="box.x + NODE_PAD"
        :y="box.y + 15"
        :font-size="FONT.node"
        class="fill-gray-800 dark:fill-gray-100 font-medium"
      >
        {{ box.label }}
      </text>
      <text
        :x="box.x + NODE_PAD"
        :y="box.y + 27"
        :font-size="FONT.meta"
        class="fill-gray-500 dark:fill-gray-400"
      >
        {{ inboundMeta(box) }}
      </text>
    </g>

    <!-- Middle column: the rule ladder, in config order. -->
    <g
      v-for="box in layout.rules"
      :key="`rule-${box.index}`"
      class="cursor-default"
      @mouseenter="focus = { kind: 'rule', id: box.index }"
    >
      <title>{{ ruleTitle(box.rule) }}</title>
      <rect
        :x="box.x"
        :y="box.y"
        :width="box.width"
        :height="box.height"
        rx="5"
        stroke-width="1"
        :stroke-dasharray="box.rule.terminal ? undefined : '4 3'"
        :class="[
          ruleClass(box.rule),
          dimmed && !focusedRules.has(box.index) ? 'opacity-40' : '',
        ]"
      />
      <text
        :x="box.x + 8"
        :y="box.y + 17"
        :font-size="FONT.meta"
        class="fill-gray-400 dark:fill-gray-500 tabular-nums"
      >
        #{{ box.index }}
      </text>
      <text
        :x="box.x + RULE_INDEX_GUTTER"
        :y="box.y + 17"
        :font-size="FONT.rule"
        :class="box.rule.reachable ? 'fill-gray-700 dark:fill-gray-200' : 'fill-gray-400 dark:fill-gray-600'"
      >
        {{ ruleLabel(box) }}
      </text>
      <text
        v-if="ruleBadge(box.rule)"
        :x="box.x + box.width - 8"
        :y="box.y + 17"
        :font-size="FONT.meta"
        text-anchor="end"
        class="fill-slate-500 dark:fill-slate-400 italic"
      >
        {{ ruleBadge(box.rule) }}
      </text>
      <!-- A rule restricted to certain inbounds is not reached by the others. -->
      <circle
        v-if="box.rule.scopedInbounds.length > 0"
        :cx="box.x - 5"
        :cy="box.y + box.height / 2"
        r="3"
        class="fill-amber-400 dark:fill-amber-500"
      />
    </g>

    <!-- The fall-through: everything no rule claimed. -->
    <g>
      <title>{{ $t(`routeFlow.fallthroughSource.${sourceKey}`) }}</title>
      <rect
        :x="layout.fallthrough.x"
        :y="layout.fallthrough.y"
        :width="layout.fallthrough.width"
        :height="layout.fallthrough.height"
        rx="5"
        stroke-width="1"
        stroke-dasharray="5 4"
        class="fill-transparent stroke-gray-400 dark:stroke-slate-500"
        :class="dimmed ? 'opacity-40' : ''"
      />
      <text
        :x="layout.fallthrough.x + 8"
        :y="layout.fallthrough.y + 17"
        :font-size="FONT.rule"
        class="fill-gray-500 dark:fill-gray-400"
      >
        {{ fallthroughLabel }}
      </text>
    </g>

    <!-- Right column: exits. Several rules may land on one. -->
    <g
      v-for="box in layout.exits"
      :key="`exit-${box.id}`"
      class="cursor-default"
      @mouseenter="focus = { kind: 'exit', id: box.id }"
    >
      <title>{{ exitTitle(box) }}</title>
      <rect
        :x="box.x"
        :y="box.y"
        :width="box.width"
        :height="box.height"
        rx="7"
        stroke-width="1.5"
        :class="[exitClass(box), dimmed && focusedExit !== box.id ? 'opacity-40' : '']"
      />
      <text
        :x="box.x + NODE_PAD"
        :y="box.y + 17"
        :font-size="FONT.node"
        class="fill-gray-800 dark:fill-gray-100 font-medium"
      >
        {{ box.label }}
      </text>
      <text
        :x="box.x + NODE_PAD"
        :y="box.y + 29"
        :font-size="FONT.meta"
        class="fill-gray-500 dark:fill-gray-400"
      >
        {{ exitMeta(box) }}
      </text>
      <!-- How many rules converge here. The count IS the OR. -->
      <g v-if="box.exit.ruleIndices.length > 1">
        <rect
          :x="box.x + box.width - 26"
          :y="box.y + 7"
          width="20"
          height="14"
          rx="7"
          class="fill-primary-100 dark:fill-primary-900/60"
        />
        <text
          :x="box.x + box.width - 16"
          :y="box.y + 17"
          :font-size="FONT.meta"
          text-anchor="middle"
          class="fill-primary-700 dark:fill-primary-300 tabular-nums font-semibold"
        >
          {{ box.exit.ruleIndices.length }}
        </text>
      </g>
    </g>
  </svg>
</template>
