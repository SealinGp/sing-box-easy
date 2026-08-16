// Shared strings for the schema-driven forms (inbounds, DNS servers, outbounds).
// Version wording lives here rather than per-domain because the gate is the
// same everywhere: the generated inventory describes the pinned sing-box
// library, and these say what the INSTALLED binary will actually accept.
export default {
  field: {
    retired: 'Removed in sing-box {removed}; this host runs {version}, which will reject it.',
    deprecated: 'Deprecated since sing-box {since}.',
    deprecatedUnversioned: 'Deprecated upstream — still accepted, but new configs should not use it.',
    retiredHidden: '{count} more field(s) hidden: removed in versions at or below the installed sing-box {version}.',
    typeRetired: 'Removed in sing-box {removed} — this host runs {version} and will reject it.',
  },
}
