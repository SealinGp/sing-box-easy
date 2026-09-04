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
 *
 * MOTION, AND WHAT EACH PIECE OF IT IS FOR
 * ────────────────────────────────────────
 * Nothing here moves to look alive. The pulse encodes RATE (its speed is a
 * function of bytes/s, and it travels linearly because a flow does not
 * accelerate); the rank marks are ANCHORS, and do not animate at all,
 * because they trade places between close flows every second and a frequent
 * change is one that must not draw attention. Every state transition — lighting, quieting,
 * hover — is at or under 300ms on the shell's own curve, and lighting up
 * (200ms) is faster than going quiet (300ms): a system's response is quick,
 * and a decay reads as a decay.
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { FONT, NODE_PAD, fitToBox, layoutRouteFlow } from '../utils/flowLayout'
import { formatCondition } from '../utils/routeTopology'
import type { RouteTopology, TopologyRule } from '../types/routeTopology'
import type { CollapsedBand, ExitBox, InboundBox, Ribbon, RuleBox } from '../utils/flowLayout'
import { formatRate, isLit } from '../utils/flowOverlay'
import type { FlowOverlay, Heat, LiveEdge } from '../utils/flowOverlay'
import { HEAT_FILL, HEAT_PULSE, HEAT_STROKE, heatClass } from '../utils/flowHeat'

const props = defineProps<{
  topology: RouteTopology
  /**
   * The live overlay, when the card is watching real traffic. Null keeps the
   * drawing static. The picture never changes shape between the two — the
   * overlay only lights, widens and animates what is already there, so an
   * operator toggling Live compares actual against expected on ONE diagram.
   */
  live?: FlowOverlay | null
  /**
   * Fill the container's HEIGHT, keeping the aspect ratio, instead of
   * rendering at the canvas's natural size. Used full-window, where the goal is
   * the whole ladder on screen at once; letterboxing on a wide screen is fine,
   * a scrollbar is not.
   */
  fit?: boolean
  /**
   * An explicit zoom factor, overriding both default sizings.
   *
   * `null`/undefined keeps them: fit-to-width docked, fit-to-height with `fit`.
   * Both of those SHRINK a 1290px canvas to whatever room there is, which is
   * the right first look and unreadable once the question is what one rule
   * says. A number here renders the same `viewBox` at `scale` and lets the
   * wrapper scroll — see `composables/useDiagramZoom.ts`.
   */
  scale?: number | null
  /**
   * Rules to fold into collapsed bands instead of drawing — the card's "busy
   * only" mode. A band is clickable and emits `expand-band`, so the fold is
   * never a dead end.
   */
  hiddenRuleIndices?: ReadonlySet<number>
}>()

const emit = defineEmits<{ (event: 'expand-band', indices: number[]): void }>()

const { t } = useI18n()

const layout = computed(() =>
  layoutRouteFlow(props.topology, { hiddenRuleIndices: props.hiddenRuleIndices }),
)

/**
 * Rendered pixel size when zoomed.
 *
 * `max-w-full` and `h-auto`/`w-auto` are exactly what makes the diagram shrink
 * to its container, so an explicit scale must drop them rather than fight them
 * — left on, `max-w-full` silently caps the zoom at the card's width and the
 * control appears to do nothing past 100%.
 */
const scaled = computed(() => {
  const factor = props.scale
  if (factor === null || factor === undefined) return null
  return {
    width: Math.round(layout.value.width * factor),
    height: Math.round(layout.value.height * factor),
  }
})

const svgClass = computed(() => {
  if (scaled.value) return 'mx-auto block select-none'
  return props.fit
    ? 'h-full w-auto max-w-full mx-auto block select-none'
    : 'max-w-full h-auto mx-auto block select-none'
})

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

/**
 * How fast opacity and width settle. Hover is a direct manipulation and must
 * feel instant; a live change is a fact arriving once a second, and easing it
 * in over half a second is what makes "this ribbon just lit up" readable
 * instead of a flicker between two frames.
 */
const fadeSpeed = computed(() => (dimmed.value ? 'fade-hover' : 'fade-live'))

/* ── Live overlay ─────────────────────────────────────────────────────────── */

const isLive = computed(() => props.live !== null && props.live !== undefined)

const liveRibbon = (ribbon: Ribbon): LiveEdge | undefined => props.live?.ribbons.get(ribbon.id)

const liveRule = (index: number) => props.live?.rules.get(index)
const liveExit = (tag: string) => props.live?.exits.get(tag)
const liveInbound = (box: InboundBox) => props.live?.ribbons.get(`in:${box.tag || box.type}`)

/**
 * Whether an element counts as carrying traffic, for the dimming.
 *
 * `lit`, not "the frame has an entry": below `RATE_FLOOR` a flow is a DNS
 * lookup or a keepalive, and treating that as busy lights the whole ladder.
 * The numbers stay in the row and the tooltip either way — this only decides
 * whether the element steps forward.
 */
const isCarrying = (edge: LiveEdge | undefined): boolean => edge?.lit === true

const ruleCarrying = (index: number): boolean => isCarrying(props.live?.ribbons.get(`rule:${index}`))

/**
 * An exit is busy on its OWN total, not on any one rule's.
 *
 * Several rules converge here — that is the fact the right-hand column exists
 * to show — so three trickling rules summing past the floor is a busy exit,
 * and reading the floor off one incoming ribbon would call it idle.
 */
const exitCarrying = (tag: string): boolean => {
  const flow = liveExit(tag)
  return flow !== undefined && isLit(flow.down, flow.up)
}

const rate = (bytesPerSec: number): string => t('routeFlow.live.rate', { rate: formatRate(bytesPerSec) })

/**
 * Opacity for an element under the two dimming regimes.
 *
 * Hover wins: while something is focused, only the focused set is bright.
 * Otherwise, in live mode, elements carrying no traffic step back so the
 * moving ones read against a quiet background — but never vanish, since the
 * expected shape is what the live traffic is being compared to.
 */
const fade = (focused: boolean, carrying: boolean): string => {
  if (dimmed.value) return focused ? '' : 'opacity-40'
  // `is-lit` shortens the transition INTO the lit state (see the styles):
  // the class is on the element by the time the new transition duration is
  // read, so lighting takes 200ms and going quiet keeps the base 300ms.
  if (isLive.value) return carrying ? 'is-lit' : 'opacity-35'
  return ''
}

const isRibbonLit = (ribbon: Ribbon): boolean => {
  if (!dimmed.value) return false
  if (ribbon.kind === 'inbound') return focus.value?.kind === 'inbound' && focus.value.id === ribbon.inboundTag
  if (ribbon.kind === 'final') return focusedExit.value === ribbon.exitId && focus.value?.kind === 'exit'
  return ribbon.ruleIndex !== undefined && focusedRules.value.has(ribbon.ruleIndex)
}

/* ── Heat and rank ──────────────────────────────────────────────────── */

/** Tier of an edge; 0 when there is no live data for it. */
const heatOf = (edge: LiveEdge | undefined): Heat => edge?.heat ?? 0

/**
 * Ribbon colour. Quiet, or not live: the brand stroke — dashed grey for the
 * fall-through, which is a default rather than a decision. Carrying: its heat
 * tier, so the hot ribbon is a different COLOUR from its neighbours and the
 * eye lands on it before any number is read.
 */
const ribbonStroke = (ribbon: Ribbon): string => {
  const edge = liveRibbon(ribbon)
  if (isLive.value && isCarrying(edge)) return heatClass(HEAT_STROKE, heatOf(edge))
  return ribbon.kind === 'final'
    ? 'stroke-gray-400 dark:stroke-slate-500'
    : 'stroke-primary-400 dark:stroke-primary-600'
}

/** The pulse, one step brighter than the ribbon it rides. */
const pulseStroke = (ribbon: Ribbon): string => heatClass(HEAT_PULSE, heatOf(liveRibbon(ribbon)))

/** A live number painted in its tier — a rate that is high looks high. */
const rateFill = (heat: Heat | undefined): string => heatClass(HEAT_FILL, heat ?? 0)

/** Radius of a rank mark. */
const RANK_R = 7

/**
 * The anchors: where the busiest few flows LEAVE their rule, marked on the
 * wire itself. A reader scanning eight moving ribbons no longer has to read
 * eight numbers to find the biggest.
 *
 * Placed from the ROW, not the ribbon: a pass-through rule (`sniff`,
 * `resolve`) has no ribbon, and a rank that is assigned but never drawn
 * leaves a "1, 2, 4" that reads as a bug rather than as a rule with no exit.
 */
const rankMarks = computed(() => {
  const rows = [
    ...layout.value.rules.map((box) => ({ key: `rule:${box.index}`, box })),
    { key: 'final', box: layout.value.fallthrough },
  ]
  return rows.flatMap(({ key, box }) => {
    const edge = props.live?.ribbons.get(key)
    if (edge?.rank == null) return []
    return [{ id: key, x: box.x + box.width + RANK_R + 2, y: box.y + box.height / 2, rank: edge.rank, heat: edge.heat }]
  })
})

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

const exitFill = (box: ExitBox): string => {
  switch (box.exit.kind) {
    case 'missing':
      return 'fill-red-50 dark:fill-red-950/40'
    case 'reject':
      return 'fill-rose-50 dark:fill-rose-950/30'
    case 'hijack-dns':
      return 'fill-violet-50 dark:fill-violet-950/30'
    case 'implicit':
      return 'fill-slate-50 dark:fill-slate-800/60'
    default:
      return 'fill-white dark:fill-slate-800'
  }
}

/**
 * An exit's outline is its KIND until it carries traffic, and its heat then:
 * the outbound is where every ribbon lands, so it is the one node whose
 * colour should say how much is landing.
 */
const exitStroke = (box: ExitBox): string => {
  const flow = liveExit(box.id)
  if (isLive.value && flow && isLit(flow.down, flow.up)) return heatClass(HEAT_STROKE, flow.heat)
  switch (box.exit.kind) {
    case 'missing':
      return 'stroke-red-400 dark:stroke-red-600'
    case 'reject':
      return 'stroke-rose-300 dark:stroke-rose-700'
    case 'hijack-dns':
      return 'stroke-violet-300 dark:stroke-violet-700'
    case 'implicit':
      return 'stroke-slate-300 dark:stroke-slate-600'
    default:
      return 'stroke-primary-300 dark:stroke-primary-700'
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

const connectionsLabel = (n: number) => t('routeFlow.live.connections', { n }, n)

/** Extra tooltip lines when a rule is carrying traffic. */
const liveRuleTitle = (index: number): string => {
  const flow = liveRule(index)
  if (!flow) return ''
  const lines = [`↓ ${rate(flow.down)} · ↑ ${rate(flow.up)} · ${connectionsLabel(flow.connections)}`]
  if (flow.hosts.length > 0) {
    lines.push(`${t('routeFlow.live.hosts')}: ${flow.hosts.map((h) => `${h.host} (${rate(h.down)})`).join(', ')}`)
  }
  return `\n${lines.join('\n')}`
}

const liveExitTitle = (tag: string): string => {
  const flow = liveExit(tag)
  if (!flow) return ''
  const lines = [`↓ ${rate(flow.down)} · ↑ ${rate(flow.up)} · ${connectionsLabel(flow.connections)}`]
  for (const via of flow.via) {
    lines.push(`${t('routeFlow.live.via', { tag: via.tag })}: ${rate(via.down)} · ${via.connections}`)
  }
  return `\n${lines.join('\n')}`
}

const liveInboundTitle = (box: InboundBox): string => {
  const flow = liveInbound(box)
  if (!flow) return ''
  return `\n↓ ${rate(flow.down)} · ↑ ${rate(flow.up)} · ${connectionsLabel(flow.connections)}`
}

/**
 * Live second line for an exit: the leaf a group actually dialled through.
 * The expected diagram deliberately does not draw group members; the one that
 * is carrying bytes right now is the exception worth naming.
 */
const exitLiveMeta = (box: ExitBox): string => {
  const flow = liveExit(box.id)
  if (!flow) return exitMeta(box)
  const top = flow.via[0]
  const text = top ? t('routeFlow.live.via', { tag: top.tag }) : exitMeta(box)
  return fitToBox(text, box.width - NODE_PAD * 2, FONT.meta)
}

/**
 * A band names how many rules it stands for and where they are, because the
 * two questions it raises are "how much am I not seeing" and "is the rule I
 * came for in here". The full index list is in the `<title>`.
 */
const bandLabel = (band: CollapsedBand): string =>
  t('routeFlow.collapsed.label', { n: band.indices.length }, band.indices.length)

const bandRange = (band: CollapsedBand): string => {
  const first = band.indices[0]
  const last = band.indices[band.indices.length - 1]
  return first === last ? `#${first}` : `#${first}–#${last}`
}

const bandTitle = (band: CollapsedBand): string =>
  [
    t('routeFlow.collapsed.title', { n: band.indices.length }, band.indices.length),
    band.indices.map((index) => `#${index}`).join(' '),
    t('routeFlow.collapsed.expandHint'),
  ].join('\n')

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
    :width="scaled?.width ?? layout.width"
    :height="scaled?.height ?? layout.height"
    :class="svgClass"
    role="img"
    :aria-label="$t('routeFlow.ariaLabel')"
    @mouseleave="focus = null"
  >
    <!-- Ribbons first, so every node paints over them. -->
    <g fill="none" stroke-linecap="round">
      <template v-for="ribbon in layout.ribbons" :key="ribbon.id">
        <!--
          Width and opacity are set as CSS properties, not attributes, so they
          transition: a ribbon that starts carrying traffic brightens and
          widens over half a second rather than snapping, and one that goes
          quiet fades back the same way. Hover keeps its own, faster pace.
        -->
        <path
          :d="ribbon.d"
          :stroke-dasharray="ribbon.kind === 'final' ? '5 4' : undefined"
          :style="{ strokeWidth: isRibbonLit(ribbon) ? 2.5 : liveRibbon(ribbon)?.width ?? 1.5 }"
          :class="[
            fadeSpeed,
            ribbonStroke(ribbon),
            isRibbonLit(ribbon)
              ? 'opacity-100'
              : dimmed
                ? 'opacity-15'
                : isLive
                  ? isCarrying(liveRibbon(ribbon))
                    ? 'opacity-80 is-lit'
                    : 'opacity-15'
                  : 'opacity-60',
          ]"
          class="ribbon"
        />
        <!--
          The pulse: a comet — a faint tail and a bright head — that travels
          the ribbon from inbound to exit, then goes again. `pathLength="100"`
          normalises every ribbon to the same unit so one dash pattern works
          for all of them without measuring the DOM. Speed is the whole
          message: one traversal per `durationSec`, log-scaled with bytes/s
          (flowOverlay.ts), and LINEAR, because a flow does not slow down as
          it nears its outbound. Only the top N ribbons get one, and it fades
          in and out (the <Transition> on the group) rather than popping when
          a ribbon enters or leaves the top N.
        -->
        <Transition name="pulse">
          <!-- Under hover the comet dims with its ribbon: a bright pulse on a dimmed ribbon is a second focus. -->
          <g
            v-if="liveRibbon(ribbon)?.animated"
            :class="[fadeSpeed, dimmed && !isRibbonLit(ribbon) ? 'opacity-15' : '']"
          >
            <path
              :d="ribbon.d"
              pathLength="100"
              :stroke-width="liveRibbon(ribbon)?.width ?? 1.5"
              stroke-dasharray="18 100"
              class="flow-tail"
              :class="pulseStroke(ribbon)"
              :style="{ animationDuration: `${liveRibbon(ribbon)?.durationSec ?? 2}s` }"
            />
            <path
              :d="ribbon.d"
              pathLength="100"
              :stroke-width="(liveRibbon(ribbon)?.width ?? 1.5) + 1"
              stroke-dasharray="5 100"
              class="flow-head"
              :class="pulseStroke(ribbon)"
              :style="{ animationDuration: `${liveRibbon(ribbon)?.durationSec ?? 2}s` }"
            />
          </g>
        </Transition>
      </template>
    </g>

    <!-- Left column: inbounds. -->
    <g
      v-for="box in layout.inbounds"
      :key="`in-${box.tag}-${box.y}`"
      class="cursor-default"
      @mouseenter="focus = { kind: 'inbound', id: box.tag }"
    >
      <title>{{ box.tag || box.type }}{{ liveInboundTitle(box) }}</title>
      <rect
        :x="box.x"
        :y="box.y"
        :width="box.width"
        :height="box.height"
        rx="7"
        stroke-width="1"
        class="fill-white stroke-gray-300 dark:fill-slate-800 dark:stroke-slate-600"
        :class="[fadeSpeed, fade(focus?.kind === 'inbound' && focus.id === box.tag, isCarrying(liveInbound(box)))]"
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
      <text
        v-if="liveInbound(box)"
        :x="box.x + box.width - NODE_PAD"
        :y="box.y + 27"
        :font-size="FONT.meta"
        text-anchor="end"
        class="tabular-nums font-semibold"
        :class="[fadeSpeed, rateFill(liveInbound(box)!.heat)]"
      >
        ↓{{ rate(liveInbound(box)!.down) }}
      </text>
    </g>

    <!-- Middle column: the rule ladder, in config order. -->
    <g
      v-for="box in layout.rules"
      :key="`rule-${box.index}`"
      class="cursor-default"
      @mouseenter="focus = { kind: 'rule', id: box.index }"
    >
      <title>{{ ruleTitle(box.rule) }}{{ liveRuleTitle(box.index) }}</title>
      <rect
        :x="box.x"
        :y="box.y"
        :width="box.width"
        :height="box.height"
        rx="5"
        stroke-width="1"
        :stroke-dasharray="box.rule.terminal ? undefined : '4 3'"
        :class="[fadeSpeed, ruleClass(box.rule), fade(focusedRules.has(box.index), ruleCarrying(box.index))]"
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
        v-if="liveRule(box.index)"
        :x="box.x + box.width - 8"
        :y="box.y + 17"
        :font-size="FONT.meta"
        text-anchor="end"
        class="tabular-nums font-semibold"
        :class="[fadeSpeed, rateFill(live?.ribbons.get(`rule:${box.index}`)?.heat)]"
      >
        ↓{{ rate(liveRule(box.index)!.down) }} · {{ liveRule(box.index)!.connections }}
      </text>
      <text
        v-else-if="ruleBadge(box.rule)"
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

    <!--
      Collapsed runs of rules carrying nothing. Drawn at their own place in the
      ladder and shorter than a rule row, so the band reads as a gap that is
      accounted for rather than as another rung. Clicking one puts its rules
      back — a fold the operator cannot undo is a fold that hides the rule they
      were looking for.
    -->
    <g
      v-for="band in layout.bands"
      :key="`band-${band.indices[0]}`"
      class="cursor-pointer"
      role="button"
      tabindex="0"
      :aria-label="bandTitle(band)"
      @click="emit('expand-band', band.indices)"
      @keydown.enter.prevent="emit('expand-band', band.indices)"
      @keydown.space.prevent="emit('expand-band', band.indices)"
    >
      <title>{{ bandTitle(band) }}</title>
      <rect
        :x="band.x"
        :y="band.y"
        :width="band.width"
        :height="band.height"
        rx="4"
        stroke-width="1"
        stroke-dasharray="3 3"
        class="fill-gray-50 stroke-gray-300 dark:fill-slate-800/40 dark:stroke-slate-700 hover:fill-gray-100 dark:hover:fill-slate-800"
      />
      <text
        :x="band.x + RULE_INDEX_GUTTER"
        :y="band.y + band.height - 5"
        :font-size="FONT.meta"
        class="fill-gray-500 dark:fill-gray-400"
      >
        {{ bandLabel(band) }}
      </text>
      <text
        :x="band.x + band.width - 8"
        :y="band.y + band.height - 5"
        :font-size="FONT.meta"
        text-anchor="end"
        class="fill-gray-400 dark:fill-gray-500 tabular-nums"
      >
        {{ bandRange(band) }}
      </text>
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
        :class="[fadeSpeed, fade(false, isCarrying(live?.ribbons.get('final')))]"
      />
      <text
        :x="layout.fallthrough.x + 8"
        :y="layout.fallthrough.y + 17"
        :font-size="FONT.rule"
        class="fill-gray-500 dark:fill-gray-400"
      >
        {{ fallthroughLabel }}
      </text>
      <text
        v-if="live?.finalFlow"
        :x="layout.fallthrough.x + layout.fallthrough.width - 8"
        :y="layout.fallthrough.y + 17"
        :font-size="FONT.meta"
        text-anchor="end"
        class="tabular-nums font-semibold"
        :class="[fadeSpeed, rateFill(live?.ribbons.get('final')?.heat)]"
      >
        ↓{{ rate(live!.finalFlow!.down) }} · {{ live!.finalFlow!.connections }}
      </text>
    </g>

    <!-- Right column: exits. Several rules may land on one. -->
    <g
      v-for="box in layout.exits"
      :key="`exit-${box.id}`"
      class="cursor-default"
      @mouseenter="focus = { kind: 'exit', id: box.id }"
    >
      <title>{{ exitTitle(box) }}{{ liveExitTitle(box.id) }}</title>
      <rect
        :x="box.x"
        :y="box.y"
        :width="box.width"
        :height="box.height"
        rx="7"
        :stroke-width="exitCarrying(box.id) ? 2 : 1.5"
        :class="[fadeSpeed, exitFill(box), exitStroke(box), fade(focusedExit === box.id, exitCarrying(box.id))]"
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
        {{ liveExit(box.id) ? exitLiveMeta(box) : exitMeta(box) }}
      </text>
      <!-- How many rules converge here. The count IS the OR. -->
      <g v-if="box.exit.ruleIndices.length > 1 && !liveExit(box.id)">
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
      <!-- Live: the rate leaving here, in the badge's place. -->
      <text
        v-if="liveExit(box.id)"
        :x="box.x + box.width - NODE_PAD"
        :y="box.y + 17"
        :font-size="FONT.meta"
        text-anchor="end"
        class="tabular-nums font-semibold"
        :class="[fadeSpeed, rateFill(liveExit(box.id)!.heat)]"
      >
        ↓{{ rate(liveExit(box.id)!.down) }}
      </text>
    </g>

    <!--
      Rank marks: the anchors. On the wire where the flow leaves its rule, so
      the mark and the ribbon it ranks are one thing. No transition on purpose:
      two close flows swap places every second, and a change that frequent
      must not be one the eye is pulled to.
    -->
    <g v-for="mark in rankMarks" :key="`rank-${mark.id}`" class="pointer-events-none">
      <title>{{ $t('routeFlow.live.rank', { n: mark.rank }) }}</title>
      <circle
        :cx="mark.x"
        :cy="mark.y"
        :r="RANK_R"
        stroke-width="1.5"
        class="stroke-white dark:stroke-slate-900"
        :class="rateFill(mark.heat)"
      />
      <text
        :x="mark.x"
        :y="mark.y + 3"
        font-size="9"
        text-anchor="middle"
        class="fill-white dark:fill-slate-900 font-bold tabular-nums"
      >
        {{ mark.rank }}
      </text>
    </g>
  </svg>
</template>

<style scoped>
/*
 * One curve for everything that eases. The shell's own (DESIGN.md §8), so a
 * ribbon settling and a menu opening feel like one system. CSS's keyword
 * `ease` is too shallow to read as a settle at these durations.
 */
svg {
  --ease-out: cubic-bezier(0.32, 0.72, 0, 1);
}

/*
 * Two paces for the same properties. Opacity, width and colour are what
 * "lit" means here, so all three ease together — a width that snaps while
 * the colour fades reads as a glitch.
 *
 * Asymmetric on purpose. A live change is a fact arriving once a second:
 * lighting up is the system RESPONDING and takes 200ms; going quiet is a
 * decay and takes the base 300ms. `is-lit` is on the element by the time the
 * new duration is read, which is what makes the direction distinguishable
 * with one transition declaration.
 */
.fade-live {
  transition:
    opacity 300ms var(--ease-out),
    stroke-width 300ms var(--ease-out),
    stroke 300ms var(--ease-out),
    fill 300ms var(--ease-out);
}
.fade-live.is-lit {
  transition-duration: 200ms;
}
/* Hover is direct manipulation; 150ms is the floor below which motion reads as flicker. */
.fade-hover {
  transition:
    opacity 150ms var(--ease-out),
    stroke-width 150ms var(--ease-out),
    stroke 150ms var(--ease-out),
    fill 150ms var(--ease-out);
}

/*
 * The comet. With `pathLength="100"` the tail is an 18-unit dash and the
 * head a 5-unit one; both slide the same distance in the same time, the
 * head's offsets shifted by 13 so it rides the tail's leading edge. The tail
 * starts fully off the path (offset = its own length) and ends fully past
 * it, so the loop restarts with nothing on screen and there is no seam.
 *
 * LINEAR. The pulse is a rate made visible, and a rate is constant along the
 * path; an ease would show the flow slowing as it nears the outbound.
 * Opacity is ramped over the first and last stretch so the comet is born and
 * dies softly rather than being cut by the path's ends. Duration is set
 * inline per ribbon from its bytes/s.
 */
.flow-tail,
.flow-head {
  animation-timing-function: linear;
  animation-iteration-count: infinite;
  stroke-linecap: round;
}
.flow-tail {
  animation-name: comet-tail;
  opacity: 0.4;
}
.flow-head {
  animation-name: comet-head;
}
@keyframes comet-tail {
  from {
    stroke-dashoffset: 18;
  }
  to {
    stroke-dashoffset: -100;
  }
}
@keyframes comet-head {
  0% {
    stroke-dashoffset: 5;
    opacity: 0;
  }
  12% {
    opacity: 1;
  }
  84% {
    opacity: 1;
  }
  100% {
    stroke-dashoffset: -113;
    opacity: 0;
  }
}

/*
 * A ribbon entering or leaving the top N fades its comet in or out. In at
 * 200ms — the system answering — out at 300ms, a decay.
 */
.pulse-enter-active {
  transition: opacity 200ms var(--ease-out);
}
.pulse-leave-active {
  transition: opacity 300ms var(--ease-out);
}
.pulse-enter-from,
.pulse-leave-to {
  opacity: 0;
}

@media (prefers-reduced-motion: reduce) {
  /* The comet becomes a steady brighter core. */
  .flow-tail,
  .flow-head {
    animation: none;
    stroke-dasharray: none;
    opacity: 0.5;
  }
}
</style>
