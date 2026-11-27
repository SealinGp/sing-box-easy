import { apiService } from './api'
import type { BasicResponse } from '../types/api'

export interface OrphanPlan {
  revision: string
  subscriptions: { id: string; nodes: string[] }[]
  node_count: number
  updated_groups: string[]
  removed_groups: string[]
  blockers: string[]
}
export interface OrphanCleanupResult {
  removed_nodes: number
  backup_path: string
  restart_required: boolean
}
export async function detectSubscriptionOrphans() {
  return (await apiService.get<BasicResponse<OrphanPlan>>('/subscription-orphans')).data.data
}
export async function cleanupSubscriptionOrphans(revision: string) {
  return (await apiService.post<BasicResponse<OrphanCleanupResult>>('/subscription-orphans/cleanup', { revision })).data.data
}
