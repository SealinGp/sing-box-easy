// Config pre-check dialog (ConfigProblemDialog.vue): a setting names an outbound that is missing or empty.
export default {
  title: 'Configuration problem',
  intro: 'The settings below point at an outbound that cannot be used. sing-box\'s check command does not detect this, but sing-box fails to start because of it, so the panel has blocked this action.',
  unchanged: 'The configuration currently in use has not been changed.',
  fields: {
    'route.final': 'Default outbound (route.final)',
    'route.rules': 'Route rules',
    'route.rule_set': 'Rule-set download outbound',
    'dns.servers': 'DNS server outbound',
  },
  fix: {
    'route.final': 'Choose a default outbound',
    'route.rules': 'Edit route rules',
    'route.rule_set': 'Edit rule sets',
    'dns.servers': 'Edit DNS servers',
  },
  kinds: {
    missing_outbound: 'Outbound "{tag}" does not exist',
    empty_group: 'Group "{tag}" has no members',
  },
}
