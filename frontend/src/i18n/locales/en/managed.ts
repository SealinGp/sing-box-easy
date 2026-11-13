export default {
 missing_dependency: 'A referenced DNS server, outbound or rule set is missing',
  detachGroup: 'Stop managing group and its policies · keep all rules',
  accept: 'Accept compatible DNS changes',
  description:
    'Manage reusable DNS settings and linked traffic rules. Review configuration changes before saving.',
  refresh: 'Refresh',
  preview: 'Preview changes',
  groupMetadata:
    'An unused group only stores settings; it adds no config rules.',
  saved:
    'Saved. Restart sing-box from Settings when ready to activate configuration changes.',
  noGroups: 'No managed DNS groups yet. Create one to reuse across policies.',
  usedBy: 'Used by {n} traffic policies',
  missingServers: 'Missing DNS servers: {names}',
  deleteNamed: 'Delete {name}',
  policyName: 'Policy name',
  addCondition: 'Add condition',
  resolveWith: 'Resolve DNS with',
  searchGroups: 'Search DNS groups',
  priority:
    'Positions are zero-based. Rules above the insertion point run first and may take priority. New policies default to the end; inspect existing rules before applying.',
  routePosition: 'Route insertion position',
  dnsPosition: 'DNS insertion position',
  existingRules: 'Inspect existing rule order',
  noPolicies: 'No managed traffic policies yet.',
  synced: 'In sync',
  modified: 'Manually modified or partially missing',
  missing: 'Generated configuration missing',
  review: 'Review changes',
  proposed: 'config.json · proposed changes',
  validation:
    'This is a proposed change. The complete configuration is validated by the installed core when you apply it. If dependencies changed, preview again.',
  before: 'Before',
  after: 'After',
  apply: 'Apply changes',
  replacement: 'Replacement DNS group',
  deleteExplanation:
    'Unused groups remove metadata only. For referenced groups, choose a replacement; all dependent policies are updated together.',
  driftHelp:
    'Manual changes are preserved. Removing configuration is allowed only when owned rules still match. Stop managing removes the policy record and keeps all current rules.',
  savedRules: 'Previously generated rules',
  removeOwned: 'Preview rule removal',
  restore: 'Preview restoration',
  detach: 'Stop managing · keep rules',
  affected: 'Affected policies',
  metadataOnly: 'Metadata only; no config changes',
  back: 'Back',
  rawImpact: 'This raw edit affects managed traffic policies',
  rawImpactHelp:
    'The edit is kept as written. Missing or modified policies will need review; nothing is recreated automatically.',
  saveRaw: 'Save this configuration',
  cancel: 'Cancel',
  recovery: 'Interrupted operation needs review',
  recoveryHelp:
    'Configuration changed during an interrupted save. Keep the current config and choose the planned or previous metadata. Any differences will remain visible for review.',
  recoverAccept: 'Retain planned metadata',
  recoverDiscard: 'Retain previous metadata',
}
