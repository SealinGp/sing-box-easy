<script setup lang="ts">
import FieldHelp from './FieldHelp.vue'
import SaveFeedback from './SaveFeedback.vue'
import Button from './Button.vue'
import { useInlineSave } from '../composables/useInlineSave'
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import Card from './Card.vue'
import { Select } from '../volt'
import { FILTER_THRESHOLD } from '../utils/selectFilter'
import { routeService, outboundService, dnsService } from '../services'
import { useToast } from 'primevue'

const toast = useToast()
const { t } = useI18n()

// Local state
const loading = ref(false)
const loadError = ref('')
const { saving, saveError, savedMessage, persist } = useInlineSave()
let persisted = { final: '', auto_detect_interface: false, default_domain_resolver: '' }
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
  loadError.value = ''
  try {
    const [finalRes, outboundsRes, dnsRes] = await Promise.all([
      routeService.getRouteFinal(),
      outboundService.getOutbounds(),
      dnsService.getDNSServers(),
    ])
    finalRoute.value = finalRes.data.final || ''
    autoDetectInterface.value = !!finalRes.data.auto_detect_interface
    defaultDomainResolver.value = finalRes.data.default_domain_resolver || ''
    persisted = { final: finalRoute.value, auto_detect_interface: autoDetectInterface.value, default_domain_resolver: defaultDomainResolver.value }

    outboundTags.value = (outboundsRes.data.outbounds || []).map((o) => ({
      tag: o.tag,
      type: (o.type as string) || '',
    }))
    dnsServerTags.value = (dnsRes.data.servers || [])
      .map((s) => s.tag)
      .filter((tag): tag is string => !!tag)
  } catch (err: any) {
    loadError.value = err.message || t('route.finalPolicy.toast.fetchFailed')
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
  if (saving.value || loading.value || loadError.value) return
  const ok = await persist(() => routeService.updateRouteSettings(payload),
    t('route.finalPolicy.toast.updated'), t('route.finalPolicy.toast.updateFailed'))
  if (ok) persisted = { ...persisted, ...payload }
  else {
    finalRoute.value = persisted.final
    autoDetectInterface.value = persisted.auto_detect_interface
    defaultDomainResolver.value = persisted.default_domain_resolver
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

      <div v-else-if="loadError" class="space-y-3">
        <p role="alert" class="text-sm text-danger">{{ loadError }}</p>
        <Button variant="secondary" @click="fetchAll">{{ $t('common.refresh') }}</Button>
      </div>
      <div v-else class="space-y-4" :aria-busy="saving">
        <!-- Final outbound -->
        <div class="flex flex-wrap items-center gap-3">
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
            :filterPlaceholder="$t('common.search')"
            :emptyFilterMessage="$t('common.noMatch')"
            :placeholder="$t('route.finalPolicy.placeholder')"
            class="w-full sm:w-72 min-w-0"
            :disabled="saving"
            @change="onFinalChange"
          />
        </div>

        <!-- Auto-detect interface -->
        <div class="flex flex-wrap items-center gap-3">
          <div class="flex items-center gap-1"><label class="text-sm font-medium text-gray-700 dark:text-gray-300 w-44 shrink-0">
            {{ $t('route.finalPolicy.autoDetectInterface') }}
          </label><FieldHelp :label="$t('route.finalPolicy.autoDetectInterface')" :text="$t('route.finalPolicy.autoDetectInterfaceHint')" /></div>
          <div class="flex items-center gap-3">
            <input
              v-model="autoDetectInterface"
              type="checkbox"
              :aria-label="$t('route.finalPolicy.autoDetectInterface')"
              class="toggle toggle-primary"
              :disabled="saving"
              @change="onAutoDetectChange"
            />

          </div>
        </div>

        <!-- Default domain resolver -->
        <div class="flex flex-wrap items-center gap-3">
          <div class="flex items-center gap-1"><label class="text-sm font-medium text-gray-700 dark:text-gray-300 w-44 shrink-0">
            {{ $t('route.finalPolicy.defaultDomainResolver') }}
          </label><FieldHelp :label="$t('route.finalPolicy.defaultDomainResolver')" :text="$t('route.finalPolicy.defaultDomainResolverHint')" /></div>
          <div class="flex items-center gap-3">
            <Select
              v-model="defaultDomainResolver"
              :options="resolverOptions"
              optionLabel="label"
              optionValue="value"
              :filter="resolverOptions.length >= FILTER_THRESHOLD"
              :filterPlaceholder="$t('common.search')"
              :emptyFilterMessage="$t('common.noMatch')"
              :placeholder="$t('route.finalPolicy.none')"
              class="w-full sm:w-72 min-w-0"
              :disabled="saving"
              @change="onResolverChange"
            />

          </div>
        </div>
      </div>
      <p v-if="saving" role="status" class="mt-3 text-sm text-text-secondary">{{ $t('common.saving') }}</p>
      <SaveFeedback class="mt-3" :error="saveError" :message="savedMessage" />
    </Card>
  </div>
</template>
