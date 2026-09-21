package apiv1

// Where the API answers, and why there is more than one answer.
//
// The original prefix embedded a sing-box version — `/api/1.12.12` — on the
// theory that the panel's HTTP contract and the core it manages move together.
// They do not, and the drift had already produced three different answers to
// one question: the directory was named v1_12_12, the package inside it was
// v1_13_0, and the path said 1.12.12. All three are now just "the API".
//
// They cannot move together, because they are versions of different things:
//
//   - The path is an HTTP CONTRACT version. It changes when this API changes
//     shape, and clients have to be updated when it does.
//   - The core version is a HOST RUNTIME fact. It changes when the operator
//     runs an upgrade, with no API change at all — and the config CRUD is
//     version-agnostic (raw JSON, validated by the host's own `sing-box
//     check`), so there is nothing for the path to select anyway.
//
// Routing on the core version is also not implementable: a client would need
// the host's version before its first call, but the endpoint reporting it sits
// behind the very prefix in question.
//
// So the contract lives at `/api/v1` and the old prefix is kept as a permanent
// alias. Every route is registered on both — the panel ships as one binary
// with an embedded frontend, but an operator's bookmarks, scripts and any
// LuCI integration already point at the old path, and breaking those to fix a
// naming mistake would be a poor trade.
var apiPrefixes = []string{
	// The contract. New clients use this.
	"/api/v1",
	// Retained for compatibility. Not deprecated on a timer: it costs one
	// extra registration per route and nothing at request time.
	"/api/1.12.12",
}
