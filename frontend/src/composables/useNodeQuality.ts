import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useNodeQualityStore } from '../stores/nodeQuality'
import { rateNode } from '../utils/nodeQuality'
import type { NodeDelaySample } from '../types/runtime'

export function useNodeQuality(history: () => NodeDelaySample[] | undefined) {
  const { t } = useI18n()
  const store = useNodeQualityStore()
  const rating = computed(() => rateNode(store.ready ? history() : [], store.rules))
  const description = computed(() => {
    if (!store.ready) return t(store.failed ? 'proxies.quality.unavailable' : 'common.loading')
    if (!rating.value.count) return t('proxies.quality.unknown')
    return t(`proxies.quality.${rating.value.level}`)
  })
  return { store, rating, description }
}
