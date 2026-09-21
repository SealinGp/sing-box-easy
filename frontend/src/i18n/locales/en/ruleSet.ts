// Shared by the DNS and route probes: both report rule sets the same way.
export default {
  unavailable: 'Some rule sets could not be read, so the rules using them could not be decided:',
  reason: {
    unknown_tag: 'No rule set with this tag exists in the config.',
    not_cached: 'Never downloaded. Start sing-box so it fetches them, or update them on the Rule Sets tab.',
    cache_unavailable:
      'The sing-box cache file could not be read. Remote rule sets live inside it, and it only exists where sing-box has actually run with this config.',
    cache_disabled: 'experimental.cache_file is off, so remote rule sets exist only in the running process’s memory.',
    file_missing: 'The local rule-set file does not exist at the configured path.',
    unsupported_srs_version:
      'Built by a newer sing-box than this panel understands, and the installed binary could not be asked either.',
    parse_error: 'The content was found but could not be decoded.',
  },
}
