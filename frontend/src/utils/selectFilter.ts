/**
 * When a picker earns a search box.
 *
 * Below this many options the list is read at a glance and a filter row is pure
 * chrome; at or above it, picking turns into scrolling — worst on the dynamic
 * lists (outbounds, DNS servers, releases), which are small on a fresh install
 * and 50+ on a real one. Bound rather than hardcoded per call site so the two
 * kinds of picker — fixed enums and server-fed lists — answer to one rule.
 */
export const FILTER_THRESHOLD = 5
