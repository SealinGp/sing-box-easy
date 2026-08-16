/**
 * Shared vocabulary for the "when … then …" rule preview.
 *
 * Its own namespace because both the route rule and DNS rule dialogs render the
 * same `RuleFlowPreview`, and the connective tissue — the AND, the "any of", the
 * catch-all warning — is identical for both. Only the OUTCOME phrase differs,
 * and that is resolved by each caller from its own namespace.
 */
export default {
  flow: {
    // The connector this preview exists to state. sing-box ANDs every matcher
    // in a rule; the OR is one level down, inside a single field.
    and: 'and',
    anyOf: 'is any of',
    more: '+{count} more',
    when: 'When',
    everything: 'every request — this rule has no conditions',
    invertPrefix: 'NOT',
    catchAll:
      'This rule matches everything and stops here, so no rule below it can ever run. Add a condition, or move it to the bottom of the list.',
    continues: '(matching continues to the next rule)',
    then: { label: 'Then' },
  },
}
