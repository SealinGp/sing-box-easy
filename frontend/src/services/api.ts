import axios, { type AxiosInstance } from 'axios'
import type {
  InitState,
  InstallTask,
  DashboardTask,
  ServiceStatus,
  Outbound,
  ParsedNode,
  Subscription,
  DNS,
  DNSServer,
  DNSRule,
  RouteRule,
  RuleSet,
  Inbound,
  LogConfig,
  ClashAPI,
  CacheFile,
  SingBoxConfig,
  DefaultRuleSet,
} from '../types/api'

class ApiService {
  private client: AxiosInstance

  constructor() {
    this.client = axios.create({
      baseURL: '/api/1.12.12',
      timeout: 30000,
      headers: {
        'Content-Type': 'application/json',
      },
    })
  }

  // Initialization APIs
  async getInitStatus(): Promise<InitState> {
    const response = await this.client.get<InitState>('/init/status')
    return response.data
  }

  async completeInit(): Promise<void> {
    await this.client.post('/init/complete')
  }

  async resetInit(): Promise<void> {
    await this.client.post('/init/reset')
  }

  // Installation APIs
  async installSingBox(version?: string, beta?: boolean): Promise<{ task_id: string; message: string }> {
    const response = await this.client.post<{ task_id: string; message: string }>('/install', {
      version,
      beta,
    })
    return response.data
  }

  async getInstallTask(taskId: string): Promise<InstallTask> {
    const response = await this.client.get<InstallTask>(`/install/task/${taskId}`)
    return response.data
  }

  async getInstallStatus(): Promise<{ installed: boolean; version: string; message: string }> {
    const response = await this.client.get<{ installed: boolean; version: string; message: string }>('/install/status')
    return response.data
  }

  async updateSingBox(version?: string, beta?: boolean): Promise<InstallTask> {
    const response = await this.client.post<InstallTask>('/update', {
      version,
      beta,
    })
    return response.data
  }

  // Dashboard APIs
  async downloadDashboard(
    targetDir?: string,
    downloadURL?: string,
    proxy?: string
  ): Promise<{ task_id: string; message: string }> {
    const response = await this.client.post<{ task_id: string; message: string }>('/dashboard/download', {
      target_dir: targetDir,
      download_url: downloadURL,
      proxy: proxy,
    })
    return response.data
  }

  async getDashboardTask(taskId: string): Promise<DashboardTask> {
    const response = await this.client.get<DashboardTask>(`/dashboard/task/${taskId}`)
    return response.data
  }

  async getDashboardStatus(): Promise<{ installed: boolean; path: string }> {
    const response = await this.client.get<{ installed: boolean; path: string }>('/dashboard/status')
    return response.data
  }

  async uploadDashboard(
    file: File,
    targetDir?: string,
    folderName?: string
  ): Promise<{ task_id: string; message: string }> {
    const formData = new FormData()
    formData.append('file', file)
    if (targetDir) {
      formData.append('target_dir', targetDir)
    }
    if (folderName) {
      formData.append('folder_name', folderName)
    }

    const response = await this.client.post<{ task_id: string; message: string }>(
      '/dashboard/upload',
      formData,
      {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
        timeout: 60000, // 1 minute timeout for upload
      }
    )
    return response.data
  }

  // Service Control APIs
  async getServiceStatus(): Promise<ServiceStatus> {
    const response = await this.client.get<ServiceStatus>('/service/status')
    return response.data
  }

  async startService(): Promise<void> {
    await this.client.post('/service/start')
  }

  async stopService(): Promise<void> {
    await this.client.post('/service/stop')
  }

  async restartService(): Promise<void> {
    await this.client.post('/service/restart')
  }

  // Configuration APIs
  async getConfig(): Promise<SingBoxConfig> {
    const response = await this.client.get<SingBoxConfig>('/config')
    return response.data
  }

  async validateConfig(config: SingBoxConfig): Promise<{ valid: boolean; error?: string }> {
    const response = await this.client.post<{ valid: boolean; error?: string }>(
      '/config/validate',
      config
    )
    return response.data
  }

  async getBackupConfig(): Promise<SingBoxConfig> {
    const response = await this.client.get<SingBoxConfig>('/config/backup')
    return response.data
  }

  async rollbackConfig(): Promise<void> {
    await this.client.post('/config/rollback')
  }

  // Node Parsing APIs
  async parseNodes(subscription: string): Promise<Outbound[]> {
    const response = await this.client.post<{
      message: string
      node_count: number
      nodes: Outbound[]
    }>('/nodes/parse', {
      subscription: subscription,
    })
    return response.data.nodes
  }

  // Outbound APIs
  async getOutbounds(): Promise<Outbound[]> {
    const response = await this.client.get<Outbound[]>('/outbounds')
    return response.data
  }

  async addOutbound(outbound: Outbound): Promise<void> {
    await this.client.post('/outbounds', outbound)
  }

  async addOutboundsBatch(outbounds: Outbound[]): Promise<{
    total: number
    added: number
    skipped: number
    results: Array<{ tag: string; added: boolean; reason?: string }>
  }> {
    const response = await this.client.post('/outbounds/batch', { outbounds })
    return response.data
  }

  async getOutboundGroups(): Promise<Outbound[]> {
    const response = await this.client.get<Outbound[]>('/outbounds/groups')
    return response.data
  }

  async getOutboundByTag(tag: string): Promise<Outbound> {
    const response = await this.client.get<Outbound>(`/outbounds/${tag}`)
    return response.data
  }

  async updateOutbound(tag: string, outbound: Outbound): Promise<void> {
    await this.client.put(`/outbounds/${tag}`, outbound)
  }

  async deleteOutbound(tag: string): Promise<void> {
    await this.client.delete(`/outbounds/${tag}`)
  }

  async updateOutboundMembers(tag: string, members: string[]): Promise<void> {
    await this.client.put(`/outbounds/${tag}/members`, { members })
  }

  // DNS APIs
  async getDNS(): Promise<DNS> {
    const response = await this.client.get<DNS>('/dns')
    return response.data
  }

  async updateDNS(dns: DNS): Promise<void> {
    await this.client.put('/dns', dns)
  }

  async getDNSServers(): Promise<DNSServer[]> {
    const response = await this.client.get<DNSServer[]>('/dns/servers')
    return response.data
  }

  async addDNSServer(server: DNSServer): Promise<void> {
    await this.client.post('/dns/servers', server)
  }

  async getDNSServerByTag(tag: string): Promise<DNSServer> {
    const response = await this.client.get<DNSServer>(`/dns/servers/${tag}`)
    return response.data
  }

  async updateDNSServer(tag: string, server: DNSServer): Promise<void> {
    await this.client.put(`/dns/servers/${tag}`, server)
  }

  async deleteDNSServer(tag: string): Promise<void> {
    await this.client.delete(`/dns/servers/${tag}`)
  }

  async getDNSHosts(): Promise<Record<string, string | string[]>> {
    const response = await this.client.get<Record<string, string | string[]>>('/dns/hosts')
    return response.data
  }

  async updateDNSHosts(hosts: Record<string, string | string[]>): Promise<void> {
    await this.client.put('/dns/hosts', hosts)
  }

  async getDNSRules(): Promise<DNSRule[]> {
    const response = await this.client.get<DNSRule[]>('/dns/rules')
    return response.data
  }

  async addDNSRule(rule: DNSRule): Promise<void> {
    await this.client.post('/dns/rules', rule)
  }

  async updateDNSRule(index: number, rule: DNSRule): Promise<void> {
    await this.client.put(`/dns/rules/${index}`, rule)
  }

  async deleteDNSRule(index: number): Promise<void> {
    await this.client.delete(`/dns/rules/${index}`)
  }

  // Inbound APIs
  async getInbounds(): Promise<{inbounds:Inbound[]}> {
    const response = await this.client.get<{inbounds:Inbound[]}>('/inbounds')
    return response.data
  }

  async addInbound(inbound: Inbound): Promise<void> {
    await this.client.post('/inbounds', inbound)
  }

  async getInboundByTag(tag: string): Promise<Inbound> {
    const response = await this.client.get<Inbound>(`/inbounds/${tag}`)
    return response.data
  }

  async updateInbound(tag: string, inbound: Inbound): Promise<void> {
    await this.client.put(`/inbounds/${tag}`, inbound)
  }

  async deleteInbound(tag: string): Promise<void> {
    await this.client.delete(`/inbounds/${tag}`)
  }

  // Route APIs
  async getRouteRules(): Promise<RouteRule[]> {
    const response = await this.client.get<RouteRule[]>('/route/rules')
    return response.data
  }

  async addRouteRule(rule: RouteRule): Promise<void> {
    await this.client.post('/route/rules', rule)
  }

  async updateRouteRule(index: number, rule: RouteRule): Promise<void> {
    await this.client.put(`/route/rules/${index}`, rule)
  }

  async deleteRouteRule(index: number): Promise<void> {
    await this.client.delete(`/route/rules/${index}`)
  }

  async getRuleSets(): Promise<{ rule_sets: RuleSet[] }> {
    const response = await this.client.get<{ rule_sets: RuleSet[] }>('/route/rule-sets')
    return response.data
  }

  async addRuleSet(ruleSet: RuleSet): Promise<void> {
    await this.client.post('/route/rule-sets', ruleSet)
  }

  async getRuleSetByTag(tag: string): Promise<RuleSet> {
    const response = await this.client.get<RuleSet>(`/route/rule-sets/${tag}`)
    return response.data
  }

  async updateRuleSet(tag: string, ruleSet: RuleSet): Promise<void> {
    await this.client.put(`/route/rule-sets/${tag}`, ruleSet)
  }

  async deleteRuleSet(tag: string): Promise<void> {
    await this.client.delete(`/route/rule-sets/${tag}`)
  }

  async getRouteFinal(): Promise<{ final: string }> {
    const response = await this.client.get<{ final: string }>('/route/final')
    return response.data
  }

  async updateRouteFinal(final: string): Promise<void> {
    await this.client.put('/route/final', { final })
  }

  // Log Configuration APIs
  async getLog(): Promise<LogConfig> {
    const response = await this.client.get<LogConfig>('/log')
    return response.data
  }

  async updateLog(log: LogConfig): Promise<void> {
    await this.client.put('/log', log)
  }

  // Experimental Configuration APIs
  async getClashAPI(): Promise<ClashAPI> {
    const response = await this.client.get<ClashAPI>('/experimental/clash-api')
    return response.data
  }

  async updateClashAPI(clashAPI: ClashAPI): Promise<void> {
    await this.client.put('/experimental/clash-api', clashAPI)
  }

  async getCacheFile(): Promise<CacheFile> {
    const response = await this.client.get<CacheFile>('/experimental/cache-file')
    return response.data
  }

  async updateCacheFile(cacheFile: CacheFile): Promise<void> {
    await this.client.put('/experimental/cache-file', cacheFile)
  }

  // Subscription APIs
  async getSubscriptions(): Promise<Subscription[]> {
    const response = await this.client.get<Subscription[]>('/subscriptions')
    return response.data
  }

  async addSubscription(subscription: Omit<Subscription, 'id'>): Promise<void> {
    await this.client.post('/subscriptions', subscription)
  }

  async getSubscriptionByID(id: string): Promise<Subscription> {
    const response = await this.client.get<Subscription>(`/subscriptions/${id}`)
    return response.data
  }

  async updateSubscription(id: string, subscription: Partial<Subscription>): Promise<void> {
    await this.client.put(`/subscriptions/${id}`, subscription)
  }

  async deleteSubscription(id: string): Promise<void> {
    await this.client.delete(`/subscriptions/${id}`)
  }

  async updateSubscriptionContent(id: string): Promise<{ count: number }> {
    const response = await this.client.post<{ count: number }>(`/subscriptions/${id}/update`)
    return response.data
  }

  // Template APIs
  async getDefaultRuleSets(): Promise<DefaultRuleSet[]> {
    const response = await this.client.get<DefaultRuleSet[]>('/templates/rule-sets')
    return response.data
  }
}

export const apiService = new ApiService()
