<script setup lang="ts">
import TooltipTarget from "./TooltipTarget.vue"
import type { RouteFlowDiagramProps } from '../types/routeFlowDiagram'
import { useRouteFlowDiagram } from '../composables/useRouteFlowDiagram'
const props = defineProps<RouteFlowDiagramProps>()
const emit = defineEmits<{ (event: 'expand-band', indices: number[]): void }>()
const {
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
} = useRouteFlowDiagram(props)
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
    <TooltipTarget as="g" :text="(box.tag || box.type) + (liveInboundTitle(box))"
      v-for="box in layout.inbounds"
      :key="`in-${box.tag}-${box.y}`"
      class="cursor-default"
      @mouseenter="focus = { kind: 'inbound', id: box.tag }"
    >

      <rect
        :x="box.x"
        :y="box.y"
        :width="box.width"
        :height="box.height"
        rx="7"
        stroke-width="1"
        class="fill-[var(--control-bg)] stroke-border"
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
    </TooltipTarget>

    <!-- Middle column: the rule ladder, in config order. -->
    <TooltipTarget as="g" :text="(ruleTitle(box.rule)) + (liveRuleTitle(box.index))"
      v-for="box in layout.rules"
      :key="`rule-${box.index}`"
      class="cursor-default"
      @mouseenter="focus = { kind: 'rule', id: box.index }"
    >

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
    </TooltipTarget>

    <!--
      Collapsed runs of rules carrying nothing. Drawn at their own place in the
      ladder and shorter than a rule row, so the band reads as a gap that is
      accounted for rather than as another rung. Clicking one puts its rules
      back — a fold the operator cannot undo is a fold that hides the rule they
      were looking for.
    -->
    <TooltipTarget as="g" :text="(bandTitle(band))"
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
    </TooltipTarget>

    <!-- The fall-through: everything no rule claimed. -->
    <TooltipTarget as="g" :text="($t(`routeFlow.fallthroughSource.${sourceKey}`))">

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
    </TooltipTarget>

    <!-- Right column: exits. Several rules may land on one. -->
    <TooltipTarget as="g" :text="(exitTitle(box)) + (liveExitTitle(box.id))"
      v-for="box in layout.exits"
      :key="`exit-${box.id}`"
      class="cursor-default"
      @mouseenter="focus = { kind: 'exit', id: box.id }"
    >

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
    </TooltipTarget>

    <!--
      Rank marks: the anchors. On the wire where the flow leaves its rule, so
      the mark and the ribbon it ranks are one thing. No transition on purpose:
      two close flows swap places every second, and a change that frequent
      must not be one the eye is pulled to.
    -->
    <TooltipTarget as="g" :text="($t('routeFlow.live.rank', { n: mark.rank }))" v-for="mark in rankMarks" :key="`rank-${mark.id}`" class="pointer-events-none">

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
    </TooltipTarget>
  </svg>
</template>

<style scoped src="../style/route-flow-diagram.css"></style>
