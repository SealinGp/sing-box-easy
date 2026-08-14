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
  <div class="grid grid-cols-1 @container xl:grid-cols-2 gap-6 items-start">
    <section class="bg-white dark:bg-gray-800 rounded-surface shadow p-5">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-1">
        {{ $t('dnsProbe.title') }}
      </h3>
      <p class="text-sm text-gray-500 dark:text-gray-400 mb-4">{{ $t('dnsProbe.desc') }}</p>
      <DnsProbePanel @result="onResult" />
    </section>

    <section class="bg-white dark:bg-gray-800 rounded-surface shadow p-5">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-1">
        {{ $t('dnsFlow.title') }}
      </h3>
      <p class="text-sm text-gray-500 dark:text-gray-400 mb-4">{{ $t('dnsFlow.desc') }}</p>

      <div v-if="loading" class="flex items-center justify-center py-6">
        <div class="animate-spin rounded-pill h-8 w-8 border-b-2 border-primary-600"></div>
      </div>
      <DnsRuleFlow v-else :dns="dns" :probe="probe" />
    </section>
  </div>
</template>
