import { computed, ref, watch, type ComputedRef, type Ref } from 'vue'

/**
 * Shared behaviour for rule-condition forms (DNS rules, route rules).
 *
 * Both forms edit a sing-box rule, and sing-box combines the matchers WITHIN
 * one rule using AND — never OR. That single fact drives everything here:
 *
 *   { "rule_set": ["blocklist"], "domain": ["example.com"] }
 *
 * does NOT mean "the blocklist plus example.com". It means "example.com, but
 * only if it is also in the blocklist", which is almost never what someone
 * building a rule intends. Mixing is legal, so neither helper forbids it — they
 * make the consequence visible before the damage is done.
 */

/**
 * Value-driven field visibility.
 *
 * Only matchers carrying a value are rendered; the rest are offered as "add"
 * buttons. A rule typically uses one or two matchers out of a dozen, and
 * rendering them all turns the dialog into a wall of empty boxes. The button
 * row doubles as the discovery mechanism for what CAN be matched on — which a
 * pile of empty inputs communicates just as well but far more noisily.
 *
 * `isFilled` must read reactive state, so the computed views below re-run when
 * the underlying rule changes.
 */
export function useOptionalFields<K extends string>(
  keys: readonly K[],
  isFilled: (key: K) => boolean,
) {
  // Fields the operator explicitly added this session. Replaced, never mutated.
  const added = ref<K[]>([]) as Ref<K[]>

  const isShown = (key: K) => isFilled(key) || added.value.includes(key)

  // A field is removable only while empty. Removing one that holds values would
  // discard those values silently, so the control simply is not offered.
  const isRemovable = (key: K) => !isFilled(key)

  const hidden = computed(() => keys.filter((key) => !isShown(key)))

  const add = (key: K) => {
    if (added.value.includes(key)) return
    added.value = [...added.value, key]
  }

  const remove = (key: K) => {
    added.value = added.value.filter((k) => k !== key)
  }

  return { isShown, isRemovable, hidden, add, remove }
}

/**
 * The rule-set / content-matcher either-or.
 *
 * A rule set already expresses the same thing the content matchers express
 * (domains, geosite, geoip), so using both in one rule is the AND trap above.
 * Whichever style the rule already uses is shown; the other is folded away.
 *
 * Three guards keep this from fighting the operator:
 *   - only the EMPTY side is ever folded, so no value is hidden;
 *   - a side the operator opened on purpose is left alone — that is how
 *     deliberate mixing stays possible;
 *   - clearing the rule out entirely restores both, since neither style is in
 *     use any more and the choice is open again.
 *
 * The watcher is `immediate`, so opening an existing rule lands on the layout
 * its contents imply, and it keeps following the rule as it is built: picking a
 * rule set folds the content matchers away, and adding the first domain folds
 * the rule set away.
 */
export function useExclusiveMatcherGroups(options: {
  hasRuleSet: Ref<boolean> | ComputedRef<boolean>
  hasMatchers: Ref<boolean> | ComputedRef<boolean>
}) {
  const { hasRuleSet, hasMatchers } = options

  const showRuleSet = ref(true)
  const showMatchers = ref(true)

  // Set once the operator deliberately opens the OTHER matching style, so the
  // AND warning appears before they type. Cleared when they fold it back away.
  const ruleSetOpenedByUser = ref(false)
  const matchersOpenedByUser = ref(false)

  watch(
    [hasRuleSet, hasMatchers],
    ([usesRuleSet, usesMatchers]) => {
      if (!usesRuleSet && !usesMatchers) {
        showRuleSet.value = true
        showMatchers.value = true
        ruleSetOpenedByUser.value = false
        matchersOpenedByUser.value = false
        return
      }
      if (usesRuleSet && !usesMatchers && !matchersOpenedByUser.value) {
        showMatchers.value = false
        return
      }
      if (usesMatchers && !usesRuleSet && !ruleSetOpenedByUser.value) {
        showRuleSet.value = false
      }
    },
    { immediate: true },
  )

  const expandRuleSet = () => {
    showRuleSet.value = true
    ruleSetOpenedByUser.value = true
  }

  const expandMatchers = () => {
    showMatchers.value = true
    matchersOpenedByUser.value = true
  }

  const collapseRuleSet = () => {
    showRuleSet.value = false
    ruleSetOpenedByUser.value = false
  }

  const collapseMatchers = () => {
    showMatchers.value = false
    matchersOpenedByUser.value = false
  }

  // Hiding is offered only for a side that is EMPTY while the other one carries
  // the rule — a value is never folded out of sight, and the last remaining
  // matching style is never hideable.
  const canHideRuleSet = computed(() => !hasRuleSet.value && hasMatchers.value)
  const canHideMatchers = computed(() => !hasMatchers.value && hasRuleSet.value)

  // Warn on a real intersection, or as soon as the operator opts into building
  // one. A fresh rule with neither side filled stays quiet.
  const showMixWarning = computed(
    () =>
      (hasRuleSet.value && hasMatchers.value) ||
      ruleSetOpenedByUser.value ||
      matchersOpenedByUser.value,
  )

  return {
    showRuleSet,
    showMatchers,
    expandRuleSet,
    expandMatchers,
    collapseRuleSet,
    collapseMatchers,
    canHideRuleSet,
    canHideMatchers,
    showMixWarning,
  }
}

export type MatchStyle = 'ruleSet' | 'content'

/**
 * The rule-set / content-matcher either-or, as an explicit choice.
 *
 * `useExclusiveMatcherGroups` above infers the same choice from what the rule
 * already holds, and hides the other side behind a "show anyway" button. That
 * reads as the form deciding, and it takes two clicks to find out that the
 * other style exists at all. A radio states the choice up front: one control,
 * both options always visible, and only the chosen editor rendered.
 *
 * WHAT THIS DELIBERATELY DOES NOT COVER
 * ────────────────────────────────────
 * The CONTEXT matchers (inbound/network/port/clash_mode/…). sing-box ANDs every
 * matcher in a rule, and a rule set genuinely cannot express where traffic came
 * from — `rule_set: geosite-cn` narrowed by `network: udp` is an ordinary,
 * correct rule. Folding context into the same radio would make the most common
 * correct route rule in sing-box unwritable in the form, so context sits outside
 * this choice and is always available.
 *
 * STRANDED VALUES
 * ───────────────
 * The unselected style can still hold values — a config loaded from disk may
 * legitimately mix, and switching styles mid-edit strands whatever was typed.
 * Hiding those would make the form lie about what a save writes, so the caller
 * is handed `strandedRuleSet` / `strandedContent` and must surface them. The
 * values are never cleared here: this composable does not own the rule.
 */
export function useMatchStyle(options: {
  hasRuleSet: Ref<boolean> | ComputedRef<boolean>
  hasMatchers: Ref<boolean> | ComputedRef<boolean>
}) {
  const { hasRuleSet, hasMatchers } = options

  // Seeded from the rule as loaded — the component is remounted per dialog open
  // (see `matchersKey` in RoutingRules.vue), so this runs against fresh state.
  // A rule that uses neither style lands on `ruleSet`: it is the style this
  // repo's templates and init wizard emit, and the one that cannot be typo'd.
  const style = ref<MatchStyle>(hasMatchers.value && !hasRuleSet.value ? 'content' : 'ruleSet')

  const select = (next: MatchStyle) => {
    style.value = next
  }

  const strandedRuleSet = computed(() => style.value !== 'ruleSet' && hasRuleSet.value)
  const strandedContent = computed(() => style.value !== 'content' && hasMatchers.value)

  return { style, select, strandedRuleSet, strandedContent }
}
