/**
 * The API prefix every client in this app uses.
 *
 * It was `/api/1.12.12` — a sing-box version in the path, on the theory that
 * the panel's HTTP contract and the core it manages move together. They do
 * not: the path changes when THIS API changes shape, while the core version
 * changes whenever the operator upgrades, with no API change at all. The
 * config endpoints are version-agnostic anyway (raw JSON, validated by the
 * host's own `sing-box check`), so there was never anything for the version in
 * the path to select.
 *
 * The server still answers on the old prefix — see apiPrefixes in the Go
 * routes — so nothing that already points there breaks.
 *
 * Defined in one module because two clients need it and they must not diverge:
 * the axios instance and the SSE reader, which cannot share the axios config
 * because it uses fetch() to send an Authorization header.
 */
export const API_BASE_URL = '/api/v1'
