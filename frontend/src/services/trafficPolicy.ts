import { apiService } from './api'
import type { BasicResponse } from '../types/api'
import type { PreviewDNSGroup } from '../lab/dnsGroupPreview'
export type ManagedGroup = PreviewDNSGroup
export interface ManagedPolicy {
  id: string
  name: string
  group_id: string
  dns_server?: string
  timeout?: number
  fallback?: 'strict' | 'general'
  outbound: string
  mode: 'or' | 'and'
  conditions: { kind: string; values: string[] }[]
  route_index: number
  dns_index: number
  prefix?: string
  route_members?: Record<string, unknown>[]
  dns_members?: Record<string, unknown>[]
  status?: 'synced' | 'modified' | 'missing' | 'missing_dependency'
}
export interface ManagedSnapshot {
  recovery?: boolean
  revision: string
  groups: ManagedGroup[]
  policies: ManagedPolicy[]
  servers: string[]
  outbounds: string[]
  rule_sets: string[]
  route_rules: Record<string, unknown>[]
  dns_rules: Record<string, unknown>[]
}
export interface ManagedCommand {
  action: string
  id?: string
  replacement?: string
  group?: ManagedGroup
  policy?: ManagedPolicy
  revision?: string
}
export interface ManagedPlan {
  revision: string
  command: ManagedCommand
  before: Record<string, unknown>
  after: Record<string, unknown>
  affected: string[]
}
export const trafficPolicyService = {
  async list() {
    return (
      await apiService.get<BasicResponse<ManagedSnapshot>>('/traffic-policies')
    ).data.data
  },
  async plan(command: ManagedCommand) {
    return (
      await apiService.post<BasicResponse<ManagedPlan>>(
        '/traffic-policies/plan',
        command
      )
    ).data.data
  },
  async commit(command: ManagedCommand) {
    await apiService.post('/traffic-policies/commit', command)
  },
  async impact(config: unknown) {
    return (
      await apiService.post<
        BasicResponse<{ revision: string; policies: ManagedPolicy[] }>
      >('/traffic-policies/impact', config)
    ).data.data
  },
}
