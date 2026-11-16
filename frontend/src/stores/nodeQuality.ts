import { defineStore } from 'pinia'
import { ref } from 'vue'
import { settingsService } from '../services'
import { DEFAULT_NODE_QUALITY, type NodeQualityRules } from '../utils/nodeQuality'

export const useNodeQualityStore = defineStore('nodeQuality', () => {
  const rules = ref<NodeQualityRules>({ ...DEFAULT_NODE_QUALITY })
  const ready = ref(false)
  const failed = ref(false)
  async function load() {
    failed.value = false
    try {
      const { data } = await settingsService.getSettings()
      rules.value = data.node_quality ?? { ...DEFAULT_NODE_QUALITY }
      ready.value = true
    } catch {
      ready.value = false
      failed.value = true
    }
  }
  async function save(value: NodeQualityRules) {
    const { data } = await settingsService.updateSettings({ node_quality: value })
    rules.value = data.node_quality
    ready.value = true
    failed.value = false
  }
  return { rules, ready, failed, load, save }
})
