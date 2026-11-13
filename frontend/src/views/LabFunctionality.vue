<script setup lang="ts">
import { useI18n } from 'vue-i18n'
const { t } = useI18n()

import { ref } from 'vue'
import TrafficPolicyPreview from '../lab/TrafficPolicyPreview.vue'
import DNSGroupPreview from '../lab/DNSGroupPreview.vue'
import type { PreviewDNSGroup } from '../lab/dnsGroupPreview'
const dnsGroups = ref<PreviewDNSGroup[]>([])
const trafficPolicy = ref<InstanceType<typeof TrafficPolicyPreview>>()
const returningToPolicy = ref(false)
function managePolicyGroups() {
  returningToPolicy.value = true
  showDNSGroups()
}
async function groupSaved(id: string) {
  if (!returningToPolicy.value) return
  returningToPolicy.value = false
  await trafficPolicy.value?.useGroup(id)
  await trafficPolicy.value?.returnToDNS()
}
function groupCancelled() {
  if (!returningToPolicy.value) return
  returningToPolicy.value = false
  void trafficPolicy.value?.returnToDNS()
}

function showDNSGroups() {
  document.getElementById('dns-groups')?.scrollIntoView({ block: 'start', behavior: 'instant' })
}



</script>
<template>
  <div class="space-y-6">
    <div><h1 class="text-xl font-semibold">{{ t('lab.functionality') }}</h1><p class="mt-2 text-sm text-text-secondary">{{ t('lab.tryDnsGroupsAndLinkedTrafficPoliciesSampleDataOnly') }}</p></div>
    <nav :aria-label="t('lab.functionalitySections')" class="flex gap-4 text-sm"><a href="#dns-groups" class="text-primary hover:underline">{{ t('lab.dnsGroups') }}</a><a href="#traffic-policy" class="text-primary hover:underline">{{ t('lab.trafficPolicies') }}</a></nav>
        <p v-if="returningToPolicy" class="rounded-control border border-border bg-bg-tertiary p-3 text-sm" role="status">{{ t('lab.chooseOrCreateADnsGroupForYourTrafficPolicy') }} <button type="button" class="ml-2 font-semibold underline" @click="groupCancelled">{{ t('lab.returnToPolicy') }}</button></p>
        <DNSGroupPreview v-model="dnsGroups" @use="returningToPolicy = false; trafficPolicy?.useGroup($event)" @saved="groupSaved" @cancel="groupCancelled" />

        <TrafficPolicyPreview ref="trafficPolicy" :groups="dnsGroups" @manage-groups="managePolicyGroups" @group-created="dnsGroups = [...dnsGroups, $event]" />

  </div>
</template>
