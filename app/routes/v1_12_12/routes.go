package v1_13_0

import (
	"github.com/cloudwego/hertz/pkg/app/server"
)

// RegisterRoutes registers all v1.12.12 API routes
func RegisterRoutes(h *server.Hertz, handler *Handler) {
	v1 := h.Group("/api/1.12.12")

	// Public APIs
	v1.POST("/user/login", handler.Login)
	v1.GET("/auth/status", handler.GetAuthStatus)

	// Authenticated APIs Group
	auth := v1.Group("", AuthMiddleware(handler.userManager, handler.authEnabled))
	auth.POST("/user/logout", handler.Logout)
	auth.GET("/user/me", handler.GetMe)
	auth.PUT("/users/:id", handler.UpdateUser) // Handled inside to allow self-update

	// Admin-only APIs Group
	admin := v1.Group("", AuthMiddleware(handler.userManager, handler.authEnabled), RequireAdmin())
	admin.GET("/users", handler.ListUsers)
	admin.POST("/users", handler.CreateUser)
	admin.DELETE("/users/:id", handler.DeleteUser)

	// Configuration Management APIs
	auth.GET("/config", handler.GetConfig)
	auth.PUT("/config", handler.UpdateConfig)
	auth.POST("/config/validate", handler.ValidateConfig)
	auth.GET("/config/backup", handler.GetBackupConfig)
	auth.POST("/config/rollback", handler.RollbackConfig)

	// Config version history
	auth.GET("/config/versions", handler.ListConfigVersions)
	auth.DELETE("/config/versions/batch", handler.DeleteConfigVersionsBatch)
	auth.GET("/config/versions/:id", handler.GetConfigVersion)
	auth.POST("/config/versions/:id/rollback", handler.RollbackToConfigVersion)
	auth.DELETE("/config/versions/:id", handler.DeleteConfigVersion)

	// Application settings
	auth.GET("/settings", handler.GetSettings)
	auth.PUT("/settings", handler.UpdateSettings)
	// Subscription info-label keywords. Registered under /settings rather than
	// /subscriptions because the latter already owns a ":id" wildcard at that
	// position, which a static sibling segment would collide with.
	auth.GET("/settings/subscription-info-keywords", handler.GetSubscriptionInfoKeywords)
	auth.PUT("/settings/subscription-info-keywords", handler.UpdateSubscriptionInfoKeywords)

	// Node Parsing API
	auth.POST("/nodes/parse", handler.ParseNodes)

	// Outbound Management APIs
	auth.GET("/outbounds", handler.GetOutbounds)
	auth.POST("/outbounds", handler.AddOutbound)
	auth.POST("/outbounds/batch", handler.AddOutboundsBatch)
	auth.DELETE("/outbounds/batch", handler.DeleteOutboundsBatch)
	auth.GET("/outbounds/groups", handler.GetOutboundGroups)
	auth.GET("/outbounds/:tag", handler.GetOutboundByTag)
	auth.PUT("/outbounds/:tag", handler.UpdateOutbound)
	auth.DELETE("/outbounds/:tag", handler.DeleteOutbound)
	auth.PUT("/outbounds/:tag/members", handler.UpdateOutboundMembers)

	// DNS Management APIs
	auth.GET("/dns", handler.GetDNS)
	// Diagnostics: resolve a domain through sing-box and explain which rule
	// handled it. POST because it performs live queries.
	auth.POST("/dns/probe", handler.ProbeDNS)
	// SSE. Same probe, reported phase by phase — see StreamProbeDNS for what
	// is genuinely streamed and what deliberately is not.
	auth.POST("/dns/probe/stream", handler.StreamProbeDNS)
	auth.PUT("/dns", handler.UpdateDNS)

	auth.GET("/dns/servers", handler.GetDNSServers)
	auth.POST("/dns/servers", handler.AddDNSServer)
	auth.GET("/dns/servers/:tag", handler.GetDNSServerByTag)
	auth.PUT("/dns/servers/:tag", handler.UpdateDNSServer)
	auth.DELETE("/dns/servers/:tag", handler.DeleteDNSServer)

	auth.GET("/dns/hosts", handler.GetDNSHosts)
	auth.PUT("/dns/hosts", handler.UpdateDNSHosts)

	auth.GET("/dns/rules", handler.GetDNSRules)
	auth.POST("/dns/rules", handler.AddDNSRule)
	// Collection-level PUT = reorder, as for /route/rules.
	auth.PUT("/dns/rules", handler.ReorderDNSRules)
	auth.PUT("/dns/rules/:index", handler.UpdateDNSRule)
	auth.DELETE("/dns/rules/:index", handler.DeleteDNSRule)

	// Inbound Management APIs
	auth.GET("/inbounds", handler.GetInbounds)
	auth.POST("/inbounds", handler.AddInbound)
	auth.GET("/inbounds/:tag", handler.GetInboundByTag)
	auth.PUT("/inbounds/:tag", handler.UpdateInbound)
	auth.DELETE("/inbounds/:tag", handler.DeleteInbound)

	// Route Management APIs
	auth.GET("/route/rules", handler.GetRouteRules)
	auth.POST("/route/rules", handler.AddRouteRule)
	// Collection-level PUT = reorder. `/route/rules/:index` owns the next path
	// segment, so a static `/route/rules/order` would collide in the router.
	auth.PUT("/route/rules", handler.ReorderRouteRules)
	auth.PUT("/route/rules/:index", handler.UpdateRouteRule)
	auth.DELETE("/route/rules/:index", handler.DeleteRouteRule)
	// Diagnostics: predict the outbound a destination would leave through,
	// BEFORE any connection is made. POST because it may ask the running
	// sing-box to resolve the name. Static sibling of /route/rules, so no
	// wildcard collision — nothing owns `/route/:x`.
	auth.POST("/route/probe", handler.ProbeRoute)

	// Live traffic for the Overview diagram (SSE). Proxied through the panel so
	// the Clash API secret stays server-side — see traffic_handler.go.
	auth.GET("/traffic/flow/stream", handler.StreamTrafficFlow)

	// Route Rule-sets
	auth.GET("/route/rule-sets", handler.GetRuleSets)
	auth.POST("/route/rule-sets", handler.AddRuleSet)
	auth.GET("/route/rule-sets/:tag", handler.GetRuleSetByTag)
	auth.GET("/route/rule-sets/:tag/references", handler.GetRuleSetReferences)
	auth.PUT("/route/rule-sets/:tag", handler.UpdateRuleSet)
	auth.DELETE("/route/rule-sets/:tag", handler.DeleteRuleSet)

	auth.GET("/route/final", handler.GetRouteFinal)
	auth.PUT("/route/final", handler.UpdateRouteFinal)

	// Log Configuration APIs
	auth.GET("/log", handler.GetLog)
	auth.PUT("/log", handler.UpdateLog)

	// Experimental Configuration APIs
	auth.GET("/experimental/clash-api", handler.GetClashAPI)
	auth.PUT("/experimental/clash-api", handler.UpdateClashAPI)

	auth.GET("/experimental/cache-file", handler.GetCacheFile)
	auth.PUT("/experimental/cache-file", handler.UpdateCacheFile)

	auth.GET("/experimental/v2ray-api", handler.GetV2RayAPI)
	auth.PUT("/experimental/v2ray-api", handler.UpdateV2RayAPI)

	// Service Control APIs
	auth.GET("/service/status", handler.GetServiceStatus)
	auth.GET("/service/logs", handler.GetServiceLogs)
	// SSE. Seeded by the request above: the client fetches the backlog, then
	// opens this with the cursor it came back with.
	auth.GET("/service/logs/stream", handler.StreamServiceLogs)
	// The panel's OWN log — the second tab on the Logs page. Same shape as the
	// pair above so the viewer swaps URLs rather than branching.
	auth.GET("/system/logs", handler.GetAppLogs)
	auth.GET("/system/logs/stream", handler.StreamAppLogs)
	auth.POST("/service/start", handler.StartService)
	auth.POST("/service/stop", handler.StopService)
	auth.POST("/service/restart", handler.RestartService)

	// Subscription Management APIs
	auth.GET("/subscriptions", handler.GetSubscriptions)
	auth.POST("/subscriptions", handler.AddSubscription)
	auth.GET("/subscriptions/:id", handler.GetSubscriptionByID)
	auth.PUT("/subscriptions/:id", handler.UpdateSubscription)
	auth.DELETE("/subscriptions/:id", handler.DeleteSubscription)
	auth.POST("/subscriptions/:id/update", handler.UpdateSubscriptionContent)

	// Outbound Node Rules APIs (Filters + Groups, auto-grouping)
	auth.GET("/node-rules", handler.GetNodeRules)
	auth.POST("/node-rules/apply", handler.ApplyNodeRules)
	auth.POST("/node-rules/preview", handler.PreviewNodeRules)
	auth.GET("/node-rules/keywords", handler.GetNodeRuleKeywords)
	auth.GET("/node-rules/templates", handler.GetNodeRuleTemplates)
	auth.POST("/node-rules/templates/:id/apply", handler.ApplyNodeRuleTemplate)
	auth.GET("/node-rules/filters", handler.GetFilters)
	auth.POST("/node-rules/filters", handler.CreateFilter)
	auth.PUT("/node-rules/filters/:id", handler.UpdateFilter)
	auth.DELETE("/node-rules/filters/:id", handler.DeleteFilter)
	auth.GET("/node-rules/groups", handler.GetGroups)
	auth.POST("/node-rules/groups", handler.CreateGroup)
	auth.PUT("/node-rules/groups/:id", handler.UpdateGroup)
	auth.DELETE("/node-rules/groups/:id", handler.DeleteGroup)

	// Scheduler Management APIs
	auth.GET("/scheduler/status", handler.schedulerHandler.GetStatus)
	auth.POST("/scheduler/start", handler.schedulerHandler.Start)
	auth.POST("/scheduler/stop", handler.schedulerHandler.Stop)
	auth.POST("/scheduler/trigger", handler.schedulerHandler.Trigger)
	auth.GET("/scheduler/jobs", handler.schedulerHandler.GetJobs)

	// Installation APIs
	auth.POST("/install", handler.InstallSingBox)
	auth.GET("/install/task/:task_id", handler.GetInstallTask)
	auth.GET("/install/status", handler.GetInstallStatus)
	auth.POST("/update", handler.UpdateSingBox)

	// Dashboard APIs
	auth.POST("/dashboard/download", handler.DownloadDashboard)
	auth.POST("/dashboard/upload", handler.UploadDashboard)
	auth.GET("/dashboard/task/:task_id", handler.GetDashboardTask)
	auth.GET("/dashboard/status", handler.GetDashboardStatus)

	// Host details for the Settings "About" card (arch, kernel, distribution,
	// sing-box + panel versions). Signed-in users only — see
	// SystemInfoResponse for why the public surface stays coarser.
	auth.GET("/system/info", handler.GetSystemInfo)

	// App version / self-update APIs.
	// Reading is available to any signed-in user; performing the update
	// replaces the binary on disk and restarts the process, so it is admin-only.
	auth.GET("/version", handler.GetVersionStatus)
	auth.GET("/version/releases", handler.ListVersions)
	auth.GET("/version/task/:task_id", handler.GetVersionUpdateTask)
	admin.POST("/version/update", handler.StartVersionUpdate)
	// opkg-managed installs cannot self-update (opkg's prerm stops this
	// service), so the panel prepares a verified .ipk and hands back the
	// command instead of running it.
	admin.POST("/version/prepare-package", handler.PrepareVersionPackage)

	// GitHub sign-in (OAuth device flow). The issued token is an instance-wide
	// credential used for every outbound GitHub call, so everything that
	// creates or destroys it is admin-only. Status is readable by any signed-in
	// user so the update card can explain a rate-limited check.
	auth.GET("/github/auth/status", handler.GetGitHubAuthStatus)
	admin.POST("/github/auth/device", handler.StartGitHubLogin)
	admin.GET("/github/auth/device/:session_id", handler.GetGitHubLoginSession)
	admin.DELETE("/github/auth/device/:session_id", handler.CancelGitHubLogin)
	admin.DELETE("/github/auth", handler.SignOutGitHub)

	// Initialization APIs
	auth.GET("/init/status", handler.GetInitStatus)
	auth.POST("/init/complete", handler.CompleteInit)
	auth.POST("/init/reset", handler.ResetInit)

	// Template APIs
	auth.GET("/templates/rule-sets", handler.GetDefaultRuleSets)
}
