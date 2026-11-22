import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useNodeQualityStore } from '../stores/nodeQuality'
import { rateNode } from '../utils/nodeQuality'
import type { NodeDelaySample } from '../types/runtime'

export function useNodeQuality(history: () => NodeDelaySample[] | undefined) {
  const { t, n } = useI18n()
  const store = useNodeQualityStore()
  const rating = computed(() => rateNode(store.ready ? history() : [], store.rules))
  const summary = computed(() => t('proxies.quality.passRate', {
    percent: n(rating.value.percent / 100, { style: 'percent', maximumFractionDigits: 1 }),
    passed: rating.value.passed, count: rating.value.count,
  }))
  const description = computed(() => {
    if (!store.ready) return t(store.failed ? 'proxies.quality.unavailable' : 'common.loading')
    if (!rating.value.count) return t('proxies.quality.unknown')
    return `${t(`proxies.quality.${rating.value.level}`)} · ${summary.value}`
  })
  return { store, rating, description, summary }
}
