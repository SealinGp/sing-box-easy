import type { InjectionKey } from 'vue'
import type { useNodeRulesPage } from './useNodeRulesPage'
export const nodeRulesPageKey: InjectionKey<ReturnType<typeof useNodeRulesPage>> = Symbol('node-rules-page')
