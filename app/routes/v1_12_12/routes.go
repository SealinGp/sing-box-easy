package v1_13_0

import (
	"github.com/cloudwego/hertz/pkg/app/server"
)

// RegisterRoutes registers all v1.12.12 API routes
func RegisterRoutes(h *server.Hertz, handler *Handler) {
	v1 := h.Group("/api/1.12.12")

	// Configuration Management APIs
	v1.GET("/config", handler.GetConfig)
	v1.PUT("/config", handler.UpdateConfig)
	v1.POST("/config/validate", handler.ValidateConfig)
	v1.GET("/config/backup", handler.GetBackupConfig)
	v1.POST("/config/rollback", handler.RollbackConfig)

	// Config version history
	v1.GET("/config/versions", handler.ListConfigVersions)
	v1.DELETE("/config/versions/batch", handler.DeleteConfigVersionsBatch)
	v1.GET("/config/versions/:id", handler.GetConfigVersion)
	v1.POST("/config/versions/:id/rollback", handler.RollbackToConfigVersion)
	v1.DELETE("/config/versions/:id", handler.DeleteConfigVersion)

	// Application settings
	v1.GET("/settings", handler.GetSettings)
	v1.PUT("/settings", handler.UpdateSettings)

	// Node Parsing API
	v1.POST("/nodes/parse", handler.ParseNodes)

	// Outbound Management APIs
	v1.GET("/outbounds", handler.GetOutbounds)
	v1.POST("/outbounds", handler.AddOutbound)
	v1.POST("/outbounds/batch", handler.AddOutboundsBatch)
	v1.DELETE("/outbounds/batch", handler.DeleteOutboundsBatch)
	v1.GET("/outbounds/groups", handler.GetOutboundGroups)
	v1.GET("/outbounds/:tag", handler.GetOutboundByTag)
	v1.PUT("/outbounds/:tag", handler.UpdateOutbound)
	v1.DELETE("/outbounds/:tag", handler.DeleteOutbound)
	v1.PUT("/outbounds/:tag/members", handler.UpdateOutboundMembers)

	// DNS Management APIs
	v1.GET("/dns", handler.GetDNS)
	v1.PUT("/dns", handler.UpdateDNS)

	v1.GET("/dns/servers", handler.GetDNSServers)
	v1.POST("/dns/servers", handler.AddDNSServer)
	v1.GET("/dns/servers/:tag", handler.GetDNSServerByTag)
	v1.PUT("/dns/servers/:tag", handler.UpdateDNSServer)
	v1.DELETE("/dns/servers/:tag", handler.DeleteDNSServer)

	v1.GET("/dns/hosts", handler.GetDNSHosts)
	v1.PUT("/dns/hosts", handler.UpdateDNSHosts)

	v1.GET("/dns/rules", handler.GetDNSRules)
	v1.POST("/dns/rules", handler.AddDNSRule)
	v1.PUT("/dns/rules/:index", handler.UpdateDNSRule)
	v1.DELETE("/dns/rules/:index", handler.DeleteDNSRule)

	// Inbound Management APIs
	v1.GET("/inbounds", handler.GetInbounds)
	v1.POST("/inbounds", handler.AddInbound)
	v1.GET("/inbounds/:tag", handler.GetInboundByTag)
	v1.PUT("/inbounds/:tag", handler.UpdateInbound)
	v1.DELETE("/inbounds/:tag", handler.DeleteInbound)

	// Route Management APIs
	v1.GET("/route/rules", handler.GetRouteRules)
	v1.POST("/route/rules", handler.AddRouteRule)
	v1.PUT("/route/rules/:index", handler.UpdateRouteRule)
	v1.DELETE("/route/rules/:index", handler.DeleteRouteRule)

	v1.GET("/route/rule-sets", handler.GetRuleSets)
	v1.POST("/route/rule-sets", handler.AddRuleSet)
	v1.GET("/route/rule-sets/:tag", handler.GetRuleSetByTag)
	v1.PUT("/route/rule-sets/:tag", handler.UpdateRuleSet)
	v1.DELETE("/route/rule-sets/:tag", handler.DeleteRuleSet)

	v1.GET("/route/final", handler.GetRouteFinal)
	v1.PUT("/route/final", handler.UpdateRouteFinal)

	// Log Configuration APIs
	v1.GET("/log", handler.GetLog)
	v1.PUT("/log", handler.UpdateLog)

	// Experimental Configuration APIs
	v1.GET("/experimental/clash-api", handler.GetClashAPI)
	v1.PUT("/experimental/clash-api", handler.UpdateClashAPI)

	v1.GET("/experimental/cache-file", handler.GetCacheFile)
	v1.PUT("/experimental/cache-file", handler.UpdateCacheFile)

	v1.GET("/experimental/v2ray-api", handler.GetV2RayAPI)
	v1.PUT("/experimental/v2ray-api", handler.UpdateV2RayAPI)

	// Service Control APIs
	v1.GET("/service/status", handler.GetServiceStatus)
	v1.GET("/service/logs", handler.GetServiceLogs)
	v1.POST("/service/start", handler.StartService)
	v1.POST("/service/stop", handler.StopService)
	v1.POST("/service/restart", handler.RestartService)

	// Subscription Management APIs
	v1.GET("/subscriptions", handler.GetSubscriptions)
	v1.POST("/subscriptions", handler.AddSubscription)
	v1.GET("/subscriptions/:id", handler.GetSubscriptionByID)
	v1.PUT("/subscriptions/:id", handler.UpdateSubscription)
	v1.DELETE("/subscriptions/:id", handler.DeleteSubscription)
	v1.POST("/subscriptions/:id/update", handler.UpdateSubscriptionContent)

	// Outbound Node Rules APIs (Filters + Groups, auto-grouping)
	v1.GET("/node-rules", handler.GetNodeRules)
	v1.POST("/node-rules/apply", handler.ApplyNodeRules)
	v1.POST("/node-rules/preview", handler.PreviewNodeRules)
	v1.GET("/node-rules/keywords", handler.GetNodeRuleKeywords)
	v1.GET("/node-rules/templates", handler.GetNodeRuleTemplates)
	v1.POST("/node-rules/templates/:id/apply", handler.ApplyNodeRuleTemplate)
	v1.GET("/node-rules/filters", handler.GetFilters)
	v1.POST("/node-rules/filters", handler.CreateFilter)
	v1.PUT("/node-rules/filters/:id", handler.UpdateFilter)
	v1.DELETE("/node-rules/filters/:id", handler.DeleteFilter)
	v1.GET("/node-rules/groups", handler.GetGroups)
	v1.POST("/node-rules/groups", handler.CreateGroup)
	v1.PUT("/node-rules/groups/:id", handler.UpdateGroup)
	v1.DELETE("/node-rules/groups/:id", handler.DeleteGroup)

	// Scheduler Management APIs
	v1.GET("/scheduler/status", handler.schedulerHandler.GetStatus)
	v1.POST("/scheduler/start", handler.schedulerHandler.Start)
	v1.POST("/scheduler/stop", handler.schedulerHandler.Stop)
	v1.POST("/scheduler/trigger", handler.schedulerHandler.Trigger)
	v1.GET("/scheduler/jobs", handler.schedulerHandler.GetJobs)

	// Installation APIs
	v1.POST("/install", handler.InstallSingBox)
	v1.GET("/install/task/:task_id", handler.GetInstallTask)
	v1.GET("/install/status", handler.GetInstallStatus)
	v1.POST("/update", handler.UpdateSingBox)

	// Dashboard APIs
	v1.POST("/dashboard/download", handler.DownloadDashboard)
	v1.POST("/dashboard/upload", handler.UploadDashboard)
	v1.GET("/dashboard/task/:task_id", handler.GetDashboardTask)
	v1.GET("/dashboard/status", handler.GetDashboardStatus)

	// Initialization APIs
	v1.GET("/init/status", handler.GetInitStatus)
	v1.POST("/init/complete", handler.CompleteInit)
	v1.POST("/init/reset", handler.ResetInit)

	// Template APIs
	v1.GET("/templates/rule-sets", handler.GetDefaultRuleSets)
}
