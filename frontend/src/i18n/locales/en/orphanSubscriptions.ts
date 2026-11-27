export default {
  "checking": "Checking for orphan subscription configuration…",
  "found": "{subscriptions} missing subscriptions still own {nodes} configured nodes.",
  "review": "Review and clean up",
  "title": "Clean up orphan subscriptions",
  "saved": "Removed and saved {count} nodes.",
  "backup": "Configuration backup",
  "restartHint": "Saved. Restart sing-box to apply; restarting interrupts existing connections.",
  "restart": "Restart and apply",
  "applied": "sing-box restarted.",
  "explanation": "These subscription IDs appear in configured node tags but are absent from the subscription database.",
  "nodes": "{count} nodes",
  "groups": "Groups with member references to remove",
  "emptyGroups": "Groups to remove after becoming empty",
  "blocked": "These settings still reference removed nodes or groups. Update them before checking again.",
  "consequence": "Configuration is backed up and validated before saving. sing-box is not restarted automatically.",
  "adminOnly": "Only administrators can clean up configuration.",
  "clean": "Clean up and save"
}
