# Dashboard configuration replaces the setup wizard

The dashboard is available immediately after authentication, including on hosts without sing-box. `/` opens Overview; legacy `/init` bookmarks redirect to Settings → sing-box core. No frontend request to `/init/status` or wizard completion flag gates access. The backend's legacy initialization endpoints are unchanged for compatibility.

| Former step / action | Dashboard owner | API / behavior |
| --- | --- | --- |
| Detect, install, reinstall sing-box; monitor task | Settings → sing-box core | GET `/install/status`, POST `/install`, GET `/install/task/:id`; target version and beta channel are editable |
| Update sing-box core | Settings → sing-box core | POST `/update`, then poll its returned `task_id`; distinct from the panel's self-update |
| Log options | Overview → Log settings (`/dashboard/log`) | GET/PUT log configuration |
| Clash API and cache options, including disable | Experimental → Clash API / Cache File | Existing get/update services; disabling Clash sends an empty configuration |
| Choose/download a web dashboard and optionally use a proxy | Experimental → Clash API → External UI | POST `/dashboard/download`, GET `/dashboard/task/:id`, GET `/dashboard/status` |
| Upload a dashboard archive, target directory and folder | Same External UI section → Upload archive | POST multipart `/dashboard/upload`, shared task polling; installed path updates the parent form |
| Parse/import nodes and create direct/block outbounds | Outbounds → Import / Add | Existing parse, batch-add and typed outbound editor |
| Register and refresh subscriptions | Subscriptions → Add / Update | Existing subscription create/update APIs, including auto-update and interval |
| Select bundled rule sets | Route → Rule sets → Presets | Existing read/add/delete rule-set calls; custom sets remain untouched |
| DNS presets, resolver strategy, FakeIP and host overrides | DNS → DNS Servers → Presets; DNS settings / server editors | Full DNS get/update preserved in the preset editor; ordinary editors remain available |
| Select inbound presets | Inbounds → Presets | Existing read/add/delete operations; snapshot rollback retained |
| Apply routing presets and final outbound | Route → Rules → Presets; Final Policy | Existing rule/final get/add/delete/update operations; snapshot rollback retained |
| Finish / skip / previous step | Removed | These changed wizard navigation or completion bookkeeping, not network functionality |

Preset editors are independent contextual panels, with their effect stated before applying and confirmed at the Apply button. Successful application refreshes the owning module instead of advancing to another step. Nothing is automatically applied on visiting a page.

Core installation progress reconnects after returning to Settings using the task ID stored in this browser. An uncertain polling failure blocks another install until status can be reconnected; a missing task clears the stale ID. No install or update is triggered by navigation.

Validation uses isolated browser response fixtures for install/update/upload and preset writes, so the real core, accounts and network configuration are not changed by UI verification.

The migrated presets use DNS/route reject actions instead of references to an undefined `dns-block` server or the retired block outbound. FakeIP uses a typed DNS server with an explicit A/AAAA route. Inbound payloads omit fields retired by the detected installed core; legacy cores retain their supported fields. Installing a different core updates the shared version used by schema forms without requiring a page reload.

The combined smart DNS (with FakeIP), mixed inbound and smart routing payloads pass `sing-box check` against the locally installed 1.14.0 binary. The validation fixture supplies inline rule sets to avoid network downloads. Other preset combinations have not been started as live services.
