<script setup lang="ts">
/**
 * DNS diagnostics: probe a domain, and see the routing logic it travelled
 * through. The two halves are deliberately on one page — a probe result is
 * most legible when the rule ladder next to it highlights the rung that fired.
 */
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import DnsProbePanel from '../../components/DnsProbePanel.vue'
import DnsRuleFlow from '../../components/DnsRuleFlow.vue'
import Card from '../../components/Card.vue'
import { dnsService } from '../../services'
import { useNotify } from '../../composables/useNotify'
import type { DnsProbeResult } from '../../types/dnsprobe'

const { t } = useI18n()
const notify = useNotify()

const dns = ref<any | null>(null)
const loading = ref(true)
const probe = ref<DnsProbeResult | null>(null)

onMounted(async () => {
  try {
    const response = await dnsService.getDNS()
    dns.value = response.data
  } catch (err) {
    notify.apiError(err, t('dnsFlow.loadFailed'))
  } finally {
    loading.value = false
  }
})

const onResult = (result: DnsProbeResult | null) => {
  probe.value = result
}
</script>

<template>
  <!--
    Both panels are <Card>, not a hand-rolled `bg-white dark:bg-gray-800 shadow`
    block: that spelling paints an opaque white slab on the glass background and
    uses Tailwind's raw `shadow` rather than the `--shadow-surface` tier.
  -->
  <div class="grid grid-cols-1 @container xl:grid-cols-2 gap-3 items-start">
    <Card>
      <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">
        {{ $t('dnsProbe.title') }}
      </h3>
      <p class="text-xs text-gray-500 dark:text-gray-400 mb-3">{{ $t('dnsProbe.desc') }}</p>
      <DnsProbePanel @result="onResult" />
    </Card>

    <Card>
      <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">
        {{ $t('dnsFlow.title') }}
      </h3>
      <p class="text-xs text-gray-500 dark:text-gray-400 mb-3">{{ $t('dnsFlow.desc') }}</p>

      <div v-if="loading" class="flex items-center justify-center py-6">
        <div class="animate-spin rounded-pill h-6 w-6 border-b-2 border-primary-600"></div>
      </div>
      <DnsRuleFlow v-else :dns="dns" :probe="probe" />
    </Card>
  </div>
</template>
