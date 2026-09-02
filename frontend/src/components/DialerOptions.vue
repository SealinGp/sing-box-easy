<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { DialerOptions } from '../types/shared'
import Input from './Input.vue'
import { Select } from '../volt'
import { useOutboundsStore } from '../stores/outbounds'
import { storeToRefs } from 'pinia'


interface Props {
  modelValue: any
  showAdvanced?: boolean
  currentTag?: string // To exclude current outbound from detour options
}

const props = withDefaults(defineProps<Props>(), {
  showAdvanced: false
})

const emit = defineEmits<{
  'update:modelValue': [value: any]
}>()

const { t } = useI18n()

// Domain strategy choices — same order as the previous native <option> list.
const domainStrategyOptions = computed(() => [
  { label: t('dialer.domainStrategy.options.default'), value: '' },
  { label: t('dialer.domainStrategy.options.preferIpv4'), value: 'prefer_ipv4' },
  { label: t('dialer.domainStrategy.options.preferIpv6'), value: 'prefer_ipv6' },
  { label: t('dialer.domainStrategy.options.ipv4Only'), value: 'ipv4_only' },
  { label: t('dialer.domainStrategy.options.ipv6Only'), value: 'ipv6_only' },
])

// `undefined` means "leave the field out of the config" — same as before.
const udpFragmentOptions = computed(() => [
  { label: t('dialer.udpFragment.options.default'), value: undefined },
  { label: t('dialer.udpFragment.options.enabled'), value: true },
  { label: t('dialer.udpFragment.options.disabled'), value: false },
])

// Get outbounds from store
const outboundsStore = useOutboundsStore()
const { outbounds } = storeToRefs(outboundsStore)

// Generate detour options from outbounds (excluding current if editing)
const detourOptions = computed(() => {
  return outbounds.value
    .filter(ob => ob.tag !== props.currentTag) // Exclude current outbound to prevent self-reference
    .map(ob => ({
      label: `${ob.tag} (${ob.type})`,
      value: ob.tag
    }))
})

// Helper to update nested fields
const updateField = (field: keyof DialerOptions, value: any) => {
  emit('update:modelValue', {
    ...props.modelValue,
    [field]: value || undefined
  })
}

// Computed properties for individual fields
const detour = computed({
  get: () => props.modelValue?.detour || '',
  set: (val) => updateField('detour', val)
})

const bindInterface = computed({
  get: () => props.modelValue?.bind_interface || '',
  set: (val) => updateField('bind_interface', val)
})

const inet4BindAddress = computed({
  get: () => props.modelValue?.inet4_bind_address || '',
  set: (val) => updateField('inet4_bind_address', val)
})

const inet6BindAddress = computed({
  get: () => props.modelValue?.inet6_bind_address || '',
  set: (val) => updateField('inet6_bind_address', val)
})

const protectPath = computed({
  get: () => props.modelValue?.protect_path || '',
  set: (val) => updateField('protect_path', val)
})

const routingMark = computed({
  get: () => props.modelValue?.routing_mark || '',
  set: (val) => updateField('routing_mark', val ? parseInt(val) : undefined)
})

const reuseAddr = computed({
  get: () => props.modelValue?.reuse_addr || false,
  set: (val) => updateField('reuse_addr', val)
})

const connectTimeout = computed({
  get: () => props.modelValue?.connect_timeout || '',
  set: (val) => updateField('connect_timeout', val)
})

const tcpFastOpen = computed({
  get: () => props.modelValue?.tcp_fast_open || false,
  set: (val) => updateField('tcp_fast_open', val)
})

const tcpMultiPath = computed({
  get: () => props.modelValue?.tcp_multi_path || false,
  set: (val) => updateField('tcp_multi_path', val)
})

const udpFragment = computed({
  get: () => props.modelValue?.udp_fragment,
  set: (val) => updateField('udp_fragment', val === '' ? undefined : val)
})

const domainStrategy = computed({
  get: () => props.modelValue?.domain_strategy || '',
  set: (val) => updateField('domain_strategy', val)
})

const fallbackDelay = computed({
  get: () => props.modelValue?.fallback_delay || '',
  set: (val) => updateField('fallback_delay', val)
})
</script>

<template>
  <div class="space-y-4">
    <!-- Dial Fields Section Header -->
    <div class="border-t border-gray-200 dark:border-gray-700 pt-4">
      <h4 class="text-sm font-medium text-gray-900 dark:text-gray-100 mb-3">
        {{ $t('dialer.title') }}
        <span class="text-xs text-gray-500 dark:text-gray-400 ml-2">{{ $t('dialer.optional') }}</span>
      </h4>
    </div>

    <!-- Basic Dial Options -->
    <div class="grid grid-cols-2 gap-4">
      <div>
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
          {{ $t('dialer.detour.label') }}
          <span class="text-xs text-gray-500 ml-1">{{ $t('dialer.detour.hint') }}</span>
        </label>
        <Select class="w-full" optionLabel="label" optionValue="value"
          v-if="detourOptions.length > 0"
          v-model="detour"
          :options="detourOptions"
          :filter="true"
          :showClear="true"
          :placeholder="$t('dialer.detour.placeholder')"
          :filterPlaceholder="$t('dialer.detour.searchPlaceholder')"
          :emptyFilterMessage="$t('dialer.detour.noOptions')"
        />
        <div v-else class="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-control bg-gray-50 dark:bg-gray-800">
          <p class="text-sm text-gray-500 dark:text-gray-400">{{ $t('dialer.detour.none') }}</p>
        </div>
        <p class="mt-1 text-xs text-gray-500">{{ $t('dialer.detour.help') }}</p>
      </div>

      <div>
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
          {{ $t('dialer.bindInterface.label') }}
        </label>
        <Input
          v-model="bindInterface"
          :placeholder="$t('dialer.bindInterface.placeholder')"
        />
        <p class="mt-1 text-xs text-gray-500">{{ $t('dialer.bindInterface.help') }}</p>
      </div>
    </div>

    <!-- IP Bind Addresses -->
    <div class="grid grid-cols-2 gap-4">
      <div>
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
          {{ $t('dialer.inet4BindAddress.label') }}
        </label>
        <Input
          v-model="inet4BindAddress"
          :placeholder="$t('dialer.inet4BindAddress.placeholder')"
        />
        <p class="mt-1 text-xs text-gray-500">{{ $t('dialer.inet4BindAddress.help') }}</p>
      </div>

      <div>
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
          {{ $t('dialer.inet6BindAddress.label') }}
        </label>
        <Input
          v-model="inet6BindAddress"
          :placeholder="$t('dialer.inet6BindAddress.placeholder')"
        />
        <p class="mt-1 text-xs text-gray-500">{{ $t('dialer.inet6BindAddress.help') }}</p>
      </div>
    </div>

    <!-- Domain Strategy -->
    <div>
      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
        {{ $t('dialer.domainStrategy.label') }}
      </label>
      <!-- The "default" entry's value is empty, which PrimeVue reads as "nothing
           selected" — so its label doubles as the placeholder. -->
      <Select
        class="w-full"
        v-model="domainStrategy"
        :options="domainStrategyOptions"
        optionLabel="label"
        optionValue="value"
        filter
        :filterPlaceholder="$t('common.search')"
        :emptyFilterMessage="$t('common.noMatch')"
        :placeholder="$t('dialer.domainStrategy.options.default')"
      />
      <p class="mt-1 text-xs text-gray-500">{{ $t('dialer.domainStrategy.help') }}</p>
    </div>

    <!-- Advanced Options (Collapsible) -->
    <details v-if="showAdvanced" class="border-t border-gray-200 dark:border-gray-700 pt-4">
      <summary class="cursor-pointer text-sm font-medium text-gray-700 dark:text-gray-300 hover:text-gray-900 dark:hover:text-gray-100">
        {{ $t('dialer.advanced') }}
      </summary>

      <div class="mt-4 space-y-4">
        <!-- Connection Options -->
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              {{ $t('dialer.connectTimeout.label') }}
            </label>
            <Input
              v-model="connectTimeout"
              :placeholder="$t('dialer.connectTimeout.placeholder')"
            />
            <p class="mt-1 text-xs text-gray-500">{{ $t('dialer.connectTimeout.help') }}</p>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              {{ $t('dialer.fallbackDelay.label') }}
            </label>
            <Input
              v-model="fallbackDelay"
              :placeholder="$t('dialer.fallbackDelay.placeholder')"
            />
            <p class="mt-1 text-xs text-gray-500">{{ $t('dialer.fallbackDelay.help') }}</p>
          </div>
        </div>

        <!-- Platform-specific Options -->
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              {{ $t('dialer.protectPath.label') }}
              <span class="text-xs text-gray-500 ml-1">{{ $t('dialer.protectPath.hint') }}</span>
            </label>
            <Input
              v-model="protectPath"
              :placeholder="$t('dialer.protectPath.placeholder')"
            />
            <p class="mt-1 text-xs text-gray-500">{{ $t('dialer.protectPath.help') }}</p>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              {{ $t('dialer.routingMark.label') }}
              <span class="text-xs text-gray-500 ml-1">{{ $t('dialer.routingMark.hint') }}</span>
            </label>
            <Input
              v-model="routingMark"
              type="number"
              :placeholder="$t('dialer.routingMark.placeholder')"
            />
            <p class="mt-1 text-xs text-gray-500">{{ $t('dialer.routingMark.help') }}</p>
          </div>
        </div>

        <!-- TCP/UDP Options -->
        <div class="space-y-3">
          <label class="flex items-center gap-2">
            <input
              type="checkbox"
              v-model="reuseAddr"
              class="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500"
            />
            <span class="text-sm text-gray-700 dark:text-gray-300">
              {{ $t('dialer.reuseAddr.label') }}
              <span class="text-xs text-gray-500 ml-1">{{ $t('dialer.reuseAddr.hint') }}</span>
            </span>
          </label>

          <label class="flex items-center gap-2">
            <input
              type="checkbox"
              v-model="tcpFastOpen"
              class="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500"
            />
            <span class="text-sm text-gray-700 dark:text-gray-300">
              {{ $t('dialer.tcpFastOpen.label') }}
              <span class="text-xs text-gray-500 ml-1">{{ $t('dialer.tcpFastOpen.hint') }}</span>
            </span>
          </label>

          <label class="flex items-center gap-2">
            <input
              type="checkbox"
              v-model="tcpMultiPath"
              class="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500"
            />
            <span class="text-sm text-gray-700 dark:text-gray-300">
              {{ $t('dialer.tcpMultiPath.label') }}
              <span class="text-xs text-gray-500 ml-1">{{ $t('dialer.tcpMultiPath.hint') }}</span>
            </span>
          </label>
        </div>

        <!-- UDP Fragment -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            {{ $t('dialer.udpFragment.label') }}
          </label>
          <Select
            class="w-full"
            v-model="udpFragment"
            :options="udpFragmentOptions"
            optionLabel="label"
            optionValue="value"
            :placeholder="$t('dialer.udpFragment.options.default')"
          />
          <p class="mt-1 text-xs text-gray-500">{{ $t('dialer.udpFragment.help') }}</p>
        </div>
      </div>
    </details>
  </div>
</template>