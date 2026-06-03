import { apiService } from './api'
import { ConfigService } from './config'
import { DashboardService } from './dashboard'
import { DNSService } from './dns'
import { ExperimentalService } from './experimental'
import { InboundService } from './inbound'
import { LogService } from './log'
import { NodeRulesService } from './noderules'
import { NodesService } from './nodes'
import { OutboundService } from './outbound'
import { RouteService } from './route'
import { ServiceControlService } from './service'
import { SettingsService } from './settings'
import { SubscriptionService } from './subscription'
import { TemplateService } from './template'

// Export all service classes
export { ApiService } from './api'
export { ConfigService } from './config'
export { DashboardService } from './dashboard'
export { DNSService } from './dns'
export { ExperimentalService } from './experimental'
export { InboundService } from './inbound'
export { LogService } from './log'
export { NodeRulesService } from './noderules'
export { NodesService } from './nodes'
export { OutboundService } from './outbound'
export { RouteService } from './route'
export { ServiceControlService } from './service'
export { SettingsService } from './settings'
export { SubscriptionService } from './subscription'
export { TemplateService } from './template'

// Create singleton instances
export const configService = new ConfigService(apiService)
export const dashboardService = new DashboardService(apiService)
export const dnsService = new DNSService(apiService)
export const experimentalService = new ExperimentalService(apiService)
export const inboundService = new InboundService(apiService)
export const logService = new LogService(apiService)
export const nodeRulesService = new NodeRulesService(apiService)
export const nodesService = new NodesService(apiService)
export const outboundService = new OutboundService(apiService)
export const routeService = new RouteService(apiService)
export const serviceControlService = new ServiceControlService(apiService)
export const settingsService = new SettingsService(apiService)
export const subscriptionService = new SubscriptionService(apiService)
export const templateService = new TemplateService(apiService)

// Re-export the base API service
export { apiService }
