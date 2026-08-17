import type { ApiService } from './api'
import type { BasicResponse, DNS, DNSServer, DNSRule } from '../types/api'
import type { DnsProbeRequest, DnsProbeResult } from '../types/dnsprobe'

export class DNSService {
  private api: ApiService

  constructor(api: ApiService) {
    this.api = api
  }

  async getDNS(): Promise<BasicResponse<DNS>> {
    const response = await this.api.get<BasicResponse<DNS>>('/dns')
    return response.data
  }

  async updateDNS(dns: DNS): Promise<BasicResponse<{ message: string }>> {
    const response = await this.api.put<BasicResponse<{ message: string }>>('/dns', dns)
    return response.data
  }

  async getDNSServers(): Promise<BasicResponse<{ servers: DNSServer[] }>> {
    const response = await this.api.get<BasicResponse<{ servers: DNSServer[] }>>('/dns/servers')
    return response.data
  }

  async addDNSServer(server: DNSServer): Promise<BasicResponse<{ message: string; tag: string }>> {
    const response = await this.api.post<BasicResponse<{ message: string; tag: string }>>('/dns/servers', server)
    return response.data
  }

  async getDNSServerByTag(tag: string): Promise<BasicResponse<DNSServer>> {
    const response = await this.api.get<BasicResponse<DNSServer>>(`/dns/servers/${tag}`)
    return response.data
  }

  async updateDNSServer(tag: string, server: DNSServer): Promise<BasicResponse<{ message: string; tag: string }>> {
    const response = await this.api.put<BasicResponse<{ message: string; tag: string }>>(`/dns/servers/${tag}`, server)
    return response.data
  }

  async deleteDNSServer(tag: string): Promise<BasicResponse<{ message: string; tag: string }>> {
    const response = await this.api.delete<BasicResponse<{ message: string; tag: string }>>(`/dns/servers/${tag}`)
    return response.data
  }

  async getDNSHosts(): Promise<BasicResponse<{ hosts: Record<string, string | string[]> }>> {
    const response = await this.api.get<BasicResponse<{ hosts: Record<string, string | string[]> }>>('/dns/hosts')
    return response.data
  }

  async updateDNSHosts(hosts: Record<string, string | string[]>): Promise<BasicResponse<{ message: string }>> {
    const response = await this.api.put<BasicResponse<{ message: string }>>('/dns/hosts', hosts)
    return response.data
  }

  async getDNSRules(): Promise<BasicResponse<{ rules: DNSRule[] }>> {
    const response = await this.api.get<BasicResponse<{ rules: DNSRule[] }>>('/dns/rules')
    return response.data
  }

  async addDNSRule(rule: DNSRule): Promise<BasicResponse<{ message: string }>> {
    const response = await this.api.post<BasicResponse<{ message: string }>>('/dns/rules', rule)
    return response.data
  }

  // `order` is a permutation of the CURRENT indices, in the order the rules
  // should end up in. The rule bodies never leave the server — which matters
  // here because a DNS rule is polymorphic, and re-uploading one would mean
  // reproducing sing-box's own decode.
  async reorderDNSRules(order: number[]): Promise<BasicResponse<{ message: string }>> {
    const response = await this.api.put<BasicResponse<{ message: string }>>('/dns/rules', { order })
    return response.data
  }

  async updateDNSRule(index: number, rule: DNSRule): Promise<BasicResponse<{ message: string; index: number }>> {
    const response = await this.api.put<BasicResponse<{ message: string; index: number }>>(`/dns/rules/${index}`, rule)
    return response.data
  }

  async deleteDNSRule(index: number): Promise<BasicResponse<{ message: string; index: number }>> {
    const response = await this.api.delete<BasicResponse<{ message: string; index: number }>>(`/dns/rules/${index}`)
    return response.data
  }

  async getDNSRuleSets(): Promise<BasicResponse<{ rule_sets: any[] }>> {
    const response = await this.api.get<BasicResponse<{ rule_sets: any[] }>>('/dns/rule-sets')
    return response.data
  }

  async addDNSRuleSet(ruleSet: any): Promise<BasicResponse<{ message: string; tag: string }>> {
    const response = await this.api.post<BasicResponse<{ message: string; tag: string }>>('/dns/rule-sets', ruleSet)
    return response.data
  }

  async updateDNSRuleSet(tag: string, ruleSet: any): Promise<BasicResponse<{ message: string; tag: string }>> {
    const response = await this.api.put<BasicResponse<{ message: string; tag: string }>>(`/dns/rule-sets/${tag}`, ruleSet)
    return response.data
  }

  async deleteDNSRuleSet(tag: string): Promise<BasicResponse<{ message: string; tag: string }>> {
    const response = await this.api.delete<BasicResponse<{ message: string; tag: string }>>(`/dns/rule-sets/${tag}`)
    return response.data
  }

  /**
   * Resolves a domain through sing-box and explains which rule handled it.
   *
   * POST because it performs live DNS queries; `compare_servers` additionally
   * queries every reachable configured upstream.
   */
  async probe(request: DnsProbeRequest): Promise<DnsProbeResult> {
    const response = await this.api.post<BasicResponse<DnsProbeResult>>('/dns/probe', request)
    return response.data.data
  }
}
