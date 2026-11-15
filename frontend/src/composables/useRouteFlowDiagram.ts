import type { RouteFlowDiagramProps } from '../types/routeFlowDiagram'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { FONT, NODE_PAD, fitToBox, layoutRouteFlow } from '../utils/flowLayout'
import { formatCondition } from '../utils/routeTopology'
import type { TopologyRule } from '../types/routeTopology'
import type { CollapsedBand, ExitBox, InboundBox, Ribbon, RuleBox } from '../utils/flowLayout'
import { formatRate, isLit } from '../utils/flowOverlay'
import type { Heat, LiveEdge } from '../utils/flowOverlay'
import { HEAT_FILL, HEAT_PULSE, HEAT_STROKE, heatClass } from '../utils/flowHeat'

export function useRouteFlowDiagram(props: RouteFlowDiagramProps) {
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
    if (!rule.reachable) return 'fill-bg-tertiary stroke-border'
    if (!rule.terminal) return 'fill-bg-secondary stroke-border'
    if (rule.catchAll) return 'fill-warning/10 stroke-warning'
    return 'fill-[var(--control-bg)] stroke-border'
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
        return 'fill-bg-secondary'
      default:
        return 'fill-[var(--control-bg)]'
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
  return {
    layout,
    scaled,
    svgClass,
    focus,
    focusedExit,
    focusedRules,
    dimmed,
    fadeSpeed,
    isLive,
    liveRibbon,
    liveRule,
    liveExit,
    liveInbound,
    isCarrying,
    ruleCarrying,
    exitCarrying,
    rate,
    fade,
    isRibbonLit,
    ribbonStroke,
    pulseStroke,
    rateFill,
    RANK_R,
    rankMarks,
    RULE_INDEX_GUTTER,
    ruleLabel,
    ruleTitle,
    ruleBadge,
    ruleClass,
    exitFill,
    exitStroke,
    exitMeta,
    exitTitle,
    sourceKey,
    liveRuleTitle,
    liveExitTitle,
    liveInboundTitle,
    exitLiveMeta,
    bandLabel,
    bandRange,
    bandTitle,
    inboundMeta,
    fallthroughLabel,
    FONT,
    NODE_PAD,
  }
}
