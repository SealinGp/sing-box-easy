<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import Card from './Card.vue'
import { Select } from '../volt'
import { routeService, outboundService, dnsService } from '../services'
import { useToast } from 'primevue'

const toast = useToast()
const { t } = useI18n()

// Local state
const loading = ref(false)
const finalRoute = ref<string>('')
const autoDetectInterface = ref<boolean>(false)
const defaultDomainResolver = ref<string>('')

// Outbound + DNS option sources for the selects.
const outboundTags = ref<{ tag: string; type: string }[]>([])
const dnsServerTags = ref<string[]>([])

const GROUP_TYPES = ['selector', 'urltest']

// Final-outbound options: groups first (the common pick), then individual nodes.
// Grouped so the dropdown is navigable even with many nodes.
const finalOptions = computed(() => {
  const groups = outboundTags.value.filter((o) => GROUP_TYPES.includes(o.type))
  const nodes = outboundTags.value.filter((o) => !GROUP_TYPES.includes(o.type))
  const sections: { label: string; items: { label: string; value: string }[] }[] = []
  if (groups.length) {
    sections.push({
      label: t('route.finalPolicy.groupsHeading'),
      items: groups.map((o) => ({ label: o.tag, value: o.tag })),
    })
  }
  if (nodes.length) {
    sections.push({
      label: t('route.finalPolicy.nodesHeading'),
      items: nodes.map((o) => ({ label: o.tag, value: o.tag })),
    })
  }
  return sections
})

// Domain-resolver options: a "none" entry plus every DNS server tag.
const resolverOptions = computed(() => [
  { label: t('route.finalPolicy.none'), value: '' },
  ...dnsServerTags.value.map((tag) => ({ label: tag, value: tag })),
])

const fetchAll = async () => {
  loading.value = true
  try {
    const [finalRes, outboundsRes, dnsRes] = await Promise.all([
      routeService.getRouteFinal(),
      outboundService.getOutbounds(),
      dnsService.getDNSServers(),
    ])
    finalRoute.value = finalRes.data.final || ''
    autoDetectInterface.value = !!finalRes.data.auto_detect_interface
    defaultDomainResolver.value = finalRes.data.default_domain_resolver || ''

    outboundTags.value = (outboundsRes.data.outbounds || []).map((o) => ({
      tag: o.tag,
      type: (o.type as string) || '',
    }))
    dnsServerTags.value = (dnsRes.data.servers || [])
      .map((s) => s.tag)
      .filter((tag): tag is string => !!tag)
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: err.message || t('route.finalPolicy.toast.fetchFailed'),
      life: 3000,
    })
  } finally {
    loading.value = false
  }
}

// Patch a subset of route settings and report via toast.
const save = async (payload: {
  final?: string
  auto_detect_interface?: boolean
  default_domain_resolver?: string
}) => {
  loading.value = true
  try {
    await routeService.updateRouteSettings(payload)
    toast.add({
      severity: 'success',
      summary: t('common.success'),
      detail: t('route.finalPolicy.toast.updated'),
      life: 3000,
    })
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: err.message || t('route.finalPolicy.toast.updateFailed'),
      life: 3000,
    })
    // Re-sync from server so the UI reflects the actual persisted state.
    await fetchAll()
  } finally {
    loading.value = false
  }
}

const onFinalChange = () => save({ final: finalRoute.value })
const onAutoDetectChange = () => save({ auto_detect_interface: autoDetectInterface.value })
const onResolverChange = () => save({ default_domain_resolver: defaultDomainResolver.value })

onMounted(() => {
  fetchAll()
})
</script>

<template>
  <div class="space-y-4 pt-2">
    <Card>
      <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">
        {{ $t('route.finalPolicy.title') }}
      </h3>
      <p class="text-gray-600 dark:text-gray-400 mb-4">
        {{ $t('route.finalPolicy.description') }}
      </p>

      <div v-if="loading" class="text-center py-4">
        <div class="inline-block animate-spin rounded-pill h-6 w-6 border-b-2 border-primary-600"></div>
      </div>

      <div v-else class="space-y-4">
        <!-- Final outbound -->
        <div class="flex items-center gap-4">
          <label class="text-sm font-medium text-gray-700 dark:text-gray-300 w-44 shrink-0">
            {{ $t('route.finalPolicy.label') }}
          </label>
          <Select
            v-model="finalRoute"
            :options="finalOptions"
            optionLabel="label"
            optionValue="value"
            optionGroupLabel="label"
            optionGroupChildren="items"
            :filter="true"
            :placeholder="$t('route.finalPolicy.placeholder')"
            class="w-72"
            :disabled="loading"
            @change="onFinalChange"
          />
        </div>

        <!-- Auto-detect interface -->
        <div class="flex items-center gap-4">
          <label class="text-sm font-medium text-gray-700 dark:text-gray-300 w-44 shrink-0">
            {{ $t('route.finalPolicy.autoDetectInterface') }}
          </label>
          <div class="flex items-center gap-3">
            <input
              v-model="autoDetectInterface"
              type="checkbox"
              class="toggle toggle-primary"
              :disabled="loading"
              @change="onAutoDetectChange"
            />
            <span class="text-xs text-gray-400">{{ $t('route.finalPolicy.autoDetectInterfaceHint') }}</span>
          </div>
        </div>

        <!-- Default domain resolver -->
        <div class="flex items-center gap-4">
          <label class="text-sm font-medium text-gray-700 dark:text-gray-300 w-44 shrink-0">
            {{ $t('route.finalPolicy.defaultDomainResolver') }}
          </label>
          <div class="flex items-center gap-3">
            <Select
              v-model="defaultDomainResolver"
              :options="resolverOptions"
              optionLabel="label"
              optionValue="value"
              :placeholder="$t('route.finalPolicy.none')"
              class="w-72"
              :disabled="loading"
              @change="onResolverChange"
            />
            <span class="text-xs text-gray-400 hidden sm:inline">{{ $t('route.finalPolicy.defaultDomainResolverHint') }}</span>
          </div>
        </div>
      </div>
    </Card>
  </div>
</template>
