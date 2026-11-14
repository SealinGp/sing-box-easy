# Overview browser network cards

The latency and network information cards measure the browser's network path.
They do not measure the sing-box host's public IP, ICMP ping, or core outbound
URL-test latency. Routing to different destinations may yield different exits.

## Reference investigation

The running zashboard at `192.168.31.201:9092/ui/` serves
`assets/index-De6b64s-.js`. Its ConnectionStatus/IPCheck logic was inspected,
then checked against the upstream source:

- https://github.com/Zephyruso/zashboard/blob/main/src/components/overview/ConnectionStatus.vue
- https://github.com/Zephyruso/zashboard/blob/main/src/api/latency.ts
- https://github.com/Zephyruso/zashboard/blob/main/src/components/overview/IPCheck.vue
- https://github.com/Zephyruso/zashboard/blob/main/src/api/geoip.ts

The supplied local `zashboard-tracing` checkout contains the Go tracing service;
its `zashboard/` directory was empty at inspection time.

Latency: four targets run concurrently, each taking ten sequential,
cache-busted favicon loads. `performance.now()` measures load completion;
successful samples produce min/average/max. The original returns zero on image
errors; this implementation uses null and reports failed samples separately.
Each request has a five-second timeout and cancels on component unmount.

Network information: IPIP.net and IP.sb are queried directly from the browser,
without application credentials, to display IP, location and organization.
These match the reference's default providers (its optional provider selectors
are not replicated). There is an eight-second timeout per provider. Errors,
including CORS/rate limiting, remain visible with refresh available. IPs start
masked; actual results are never persisted to the database.

Both cards run on mounting and can be refreshed independently. Layout IDs
`latency` and `network-info` are accepted by the backend preference validator.
Existing saved layouts retain their order and append the new cards.

## Validation

- Unit tests: provider payload normalization, malformed responses, failed latency
  samples excluded from statistics, and older layout upgrade behavior.
- Go suite includes database save/reload of an order containing both new IDs.
- Frontend production build passes.
- Browser checks use fixture responses: four targets × ten samples, IP masking,
  failure/retry, Chinese labels, and a 390px mobile viewport. Provider availability
  in a user's network is not guaranteed by fixture tests.
