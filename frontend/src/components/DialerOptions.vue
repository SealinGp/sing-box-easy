<script setup lang="ts">
import { computed } from 'vue'
import type { DialerOptions } from '../types/shared'
import Input from './Input.vue'
import Select from './Select.vue'
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
        Dial Options
        <span class="text-xs text-gray-500 dark:text-gray-400 ml-2">(Optional)</span>
      </h4>
    </div>

    <!-- Basic Dial Options -->
    <div class="grid grid-cols-2 gap-4">
      <div>
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
          Detour
          <span class="text-xs text-gray-500 ml-1">(Outbound tag)</span>
        </label>
        <Select
          v-if="detourOptions.length > 0"
          v-model="detour"
          :options="detourOptions"
          :searchable="true"
          :clearable="true"
          placeholder="Select an outbound to route through"
          search-placeholder="Type to filter outbounds..."
          no-options-text="No matching outbounds found"
        />
        <div v-else class="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-gray-50 dark:bg-gray-800">
          <p class="text-sm text-gray-500 dark:text-gray-400">No other outbounds available</p>
        </div>
        <p class="mt-1 text-xs text-gray-500">Route through another outbound</p>
      </div>

      <div>
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
          Bind Interface
        </label>
        <Input
          v-model="bindInterface"
          placeholder="e.g., eth0, en0"
        />
        <p class="mt-1 text-xs text-gray-500">Bind to specific network interface</p>
      </div>
    </div>

    <!-- IP Bind Addresses -->
    <div class="grid grid-cols-2 gap-4">
      <div>
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
          IPv4 Bind Address
        </label>
        <Input
          v-model="inet4BindAddress"
          placeholder="e.g., 192.168.1.100"
        />
        <p class="mt-1 text-xs text-gray-500">Local IPv4 address to bind</p>
      </div>

      <div>
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
          IPv6 Bind Address
        </label>
        <Input
          v-model="inet6BindAddress"
          placeholder="e.g., ::1"
        />
        <p class="mt-1 text-xs text-gray-500">Local IPv6 address to bind</p>
      </div>
    </div>

    <!-- Domain Strategy -->
    <div>
      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
        Domain Strategy
      </label>
      <select class="select" v-model="domainStrategy">
        <option value="">Default</option>
        <option value="prefer_ipv4">Prefer IPv4</option>
        <option value="prefer_ipv6">Prefer IPv6</option>
        <option value="ipv4_only">IPv4 Only</option>
        <option value="ipv6_only">IPv6 Only</option>
      </select>
      <p class="mt-1 text-xs text-gray-500">DNS resolution strategy for domains</p>
    </div>

    <!-- Advanced Options (Collapsible) -->
    <details v-if="showAdvanced" class="border-t border-gray-200 dark:border-gray-700 pt-4">
      <summary class="cursor-pointer text-sm font-medium text-gray-700 dark:text-gray-300 hover:text-gray-900 dark:hover:text-gray-100">
        Advanced Options
      </summary>

      <div class="mt-4 space-y-4">
        <!-- Connection Options -->
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Connect Timeout
            </label>
            <Input
              v-model="connectTimeout"
              placeholder="e.g., 5s, 30s"
            />
            <p class="mt-1 text-xs text-gray-500">Connection timeout duration</p>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Fallback Delay
            </label>
            <Input
              v-model="fallbackDelay"
              placeholder="e.g., 300ms"
            />
            <p class="mt-1 text-xs text-gray-500">Delay before fallback</p>
          </div>
        </div>

        <!-- Platform-specific Options -->
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Protect Path
              <span class="text-xs text-gray-500 ml-1">(Android)</span>
            </label>
            <Input
              v-model="protectPath"
              placeholder="e.g., /dev/protect"
            />
            <p class="mt-1 text-xs text-gray-500">Android VPN protect path</p>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Routing Mark
              <span class="text-xs text-gray-500 ml-1">(Linux)</span>
            </label>
            <Input
              v-model="routingMark"
              type="number"
              placeholder="e.g., 255"
            />
            <p class="mt-1 text-xs text-gray-500">SO_MARK socket option</p>
          </div>
        </div>

        <!-- TCP/UDP Options -->
        <div class="space-y-3">
          <label class="flex items-center gap-2">
            <input
              type="checkbox"
              v-model="reuseAddr"
              class="w-4 h-4 text-violet-600 border-gray-300 rounded focus:ring-violet-500"
            />
            <span class="text-sm text-gray-700 dark:text-gray-300">
              Reuse Address
              <span class="text-xs text-gray-500 ml-1">(SO_REUSEADDR)</span>
            </span>
          </label>

          <label class="flex items-center gap-2">
            <input
              type="checkbox"
              v-model="tcpFastOpen"
              class="w-4 h-4 text-violet-600 border-gray-300 rounded focus:ring-violet-500"
            />
            <span class="text-sm text-gray-700 dark:text-gray-300">
              TCP Fast Open
              <span class="text-xs text-gray-500 ml-1">(TFO)</span>
            </span>
          </label>

          <label class="flex items-center gap-2">
            <input
              type="checkbox"
              v-model="tcpMultiPath"
              class="w-4 h-4 text-violet-600 border-gray-300 rounded focus:ring-violet-500"
            />
            <span class="text-sm text-gray-700 dark:text-gray-300">
              TCP Multi-Path
              <span class="text-xs text-gray-500 ml-1">(MPTCP)</span>
            </span>
          </label>
        </div>

        <!-- UDP Fragment -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            UDP Fragment
          </label>
          <select class="select" v-model="udpFragment">
            <option :value="undefined">Default</option>
            <option :value="true">Enabled</option>
            <option :value="false">Disabled</option>
          </select>
          <p class="mt-1 text-xs text-gray-500">Control UDP packet fragmentation</p>
        </div>
      </div>
    </details>
  </div>
</template>