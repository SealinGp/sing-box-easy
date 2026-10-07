/**
 * Cross-page "take me to that card" links.
 *
 * Several pages name something that is edited or inspected on another page: an
 * outbound owned by a node rule, the proxy group a connection left through,
 * the rule sets its rule matched. A plain link to the other page leaves the
 * operator to find the item again by eye, in a list that may hold dozens.
 *
 * So the link carries the item's name as `?focus=<name>`, and the destination
 * scrolls to that card and tints it (`useFocusTarget`). A query parameter, not
 * a hash or a path segment: these names are free text — spaces, emoji, `|`,
 * sometimes `/` — which a query value carries and the router encodes for us.
 */

/** Pages that understand `?focus=`. */
export const FOCUS_PAGES = {
  nodeRules: '/dashboard/outbounds/node-rules',
  proxies: '/dashboard/proxies',
  ruleSets: '/dashboard/route/rule-sets',
} as const

export type FocusPage = (typeof FOCUS_PAGES)[keyof typeof FOCUS_PAGES]

/** Router location for "open this page with that item highlighted". */
export const focusLink = (path: FocusPage, name: string) => ({ path, query: { focus: name } })

/** The focused name from a route's query, or '' when there is none. */
export function focusFromQuery(value: unknown): string {
  const first: unknown = Array.isArray(value) ? value[0] : value
  return typeof first === 'string' ? first : ''
}

/** One stretch of a rule string: plain text, or a rule set that can be linked. */
export interface RulePart {
  text: string
  /** Set when this part is a rule-set tag the config actually defines. */
  ruleSet?: string
}

/**
 * `rule_set=[a b c]` (several sets) or `rule_set=a` (one), as sing-box prints
 * them. Sampled from a running 1.14.1: a rule string is
 * `<rule.String()> => <action.String()>`, and the rule-set item is the tags
 * space-separated inside brackets.
 */
const RULE_SET_CLAUSE = /rule_set=(\[[^\]]*\]|\S+)/g

/**
 * Split a connection's rule string so its rule-set tags can be rendered as
 * links, without changing a character of it.
 *
 * Only tags in `known` (the rule sets the config defines) become links. The
 * string is sing-box's own formatting, not a contract: matching against real
 * tags means a format this does not anticipate degrades to plain text instead
 * of producing a link to nothing.
 */
export function splitRuleByRuleSets(rule: string, known: ReadonlySet<string>): RulePart[] {
  const parts: RulePart[] = []
  const pushText = (text: string) => {
    if (text === '') return
    const last = parts[parts.length - 1]
    // Adjacent plain stretches merge, so a rule with nothing linkable is one part.
    if (last && last.ruleSet === undefined) {
      parts[parts.length - 1] = { text: last.text + text }
      return
    }
    parts.push({ text })
  }

  let cursor = 0
  for (const match of rule.matchAll(RULE_SET_CLAUSE)) {
    const clauseStart = match.index ?? 0
    const value = match[1] ?? ''
    const valueStart = clauseStart + match[0].length - value.length
    pushText(rule.slice(cursor, valueStart))

    // Walk the value token by token, keeping the separators as text.
    for (const token of value.split(/([[\]\s]+)/)) {
      if (token !== '' && known.has(token)) parts.push({ text: token, ruleSet: token })
      else pushText(token)
    }
    cursor = valueStart + value.length
  }
  pushText(rule.slice(cursor))

  return parts
}
