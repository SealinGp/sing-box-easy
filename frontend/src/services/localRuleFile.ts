import { apiService } from './api'
import type { BasicResponse } from '../types/api'
export interface RuleSource { version: number; rules: Record<string, unknown>[] }
export interface LocalRuleFile { tag: string; path: string; directory: string; content: RuleSource; revision: string }
export async function readLocalRuleFile(tag: string) {
  return (await apiService.get<BasicResponse<LocalRuleFile>>('/route/local-rule-file?tag=' + encodeURIComponent(tag))).data.data
}
export async function saveLocalRuleFile(tag: string, existing: boolean, revision: string, content: RuleSource) {
  return (await apiService.put<BasicResponse<LocalRuleFile>>('/route/local-rule-file', { tag, existing, revision, content })).data.data
}
