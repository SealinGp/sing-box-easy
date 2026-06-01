// Settings page (Settings.vue).
export default {
  title: 'Settings',
  versionHistory: {
    title: 'Config version history',
    desc: 'How many historical configurations to keep. Older versions beyond this count are pruned automatically after each save. Range {min}–{max}.',
    versionsToKeep: 'Versions to keep',
  },
  language: {
    title: 'Language',
    desc: 'Choose the interface language.',
  },
  about: {
    title: 'About',
  },
  toast: {
    loadFailed: 'Failed to load settings',
    saveFailed: 'Failed to save settings',
    savedTitle: 'Saved',
    savedDetail: 'Settings updated',
  },
}
