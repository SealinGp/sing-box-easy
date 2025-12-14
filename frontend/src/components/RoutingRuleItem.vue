<script setup lang="ts">
import type { RouteRule } from '../types/api'

interface Props {
  rule: RouteRule
  index: number
}

interface Emits {
  (e: 'edit', index: number, rule: RouteRule): void
  (e: 'delete', index: number): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

function handleEdit() {
  emit('edit', props.index, props.rule)
}

function handleDelete() {
  emit('delete', props.index)
}
</script>

<template>
  <div class="border border-gray-200 dark:border-gray-700 rounded-lg p-4">
    <div class="flex justify-between items-start">
      <div class="flex-1">
        <div class="grid grid-cols-2 gap-4 text-sm">
          <div v-if="rule.action">
            <span class="font-medium text-gray-700 dark:text-gray-300">Action:</span>
            <span class="ml-2 text-gray-900 dark:text-gray-100">{{ rule.action }}</span>
          </div>
          <div v-if="rule.outbound">
            <span class="font-medium text-gray-700 dark:text-gray-300">Outbound:</span>
            <span class="ml-2 text-gray-900 dark:text-gray-100">{{ rule.outbound }}</span>
          </div>
          <div v-if="rule.inbound?.length">
            <span class="font-medium text-gray-700 dark:text-gray-300">Inbound:</span>
            <span class="ml-2 text-gray-900 dark:text-gray-100">{{ rule.inbound.join(', ') }}</span>
          </div>
          <div v-if="rule.protocol && rule.protocol?.length">
            <span class="font-medium text-gray-700 dark:text-gray-300">Protocol:</span>
            <span class="ml-2 text-gray-900 dark:text-gray-100">{{ rule.protocol.join(', ') }}</span>
          </div>
          <div v-if="rule.network?.length">
            <span class="font-medium text-gray-700 dark:text-gray-300">Network:</span>
            <span class="ml-2 text-gray-900 dark:text-gray-100">{{ rule.network.join(', ') }}</span>
          </div>
          <div v-if="rule.domain?.length">
            <span class="font-medium text-gray-700 dark:text-gray-300">Domain:</span>
            <span class="ml-2 text-gray-900 dark:text-gray-100">{{ rule.domain.join(', ') }}</span>
          </div>
          <div v-if="rule.domain_suffix?.length">
            <span class="font-medium text-gray-700 dark:text-gray-300">Domain Suffix:</span>
            <span class="ml-2 text-gray-900 dark:text-gray-100">{{ rule.domain_suffix.join(', ') }}</span>
          </div>
          <div v-if="rule.geosite && (Array.isArray(rule.geosite) ? rule.geosite.length : rule.geosite)">
            <span class="font-medium text-gray-700 dark:text-gray-300">GeoSite:</span>
            <span class="ml-2 text-gray-900 dark:text-gray-100">
              {{ Array.isArray(rule.geosite) ? rule.geosite.join(', ') : rule.geosite }}
            </span>
          </div>
          <div v-if="rule.geoip && (Array.isArray(rule.geoip) ? rule.geoip.length : rule.geoip)">
            <span class="font-medium text-gray-700 dark:text-gray-300">GeoIP:</span>
            <span class="ml-2 text-gray-900 dark:text-gray-100">
              {{ Array.isArray(rule.geoip) ? rule.geoip.join(', ') : rule.geoip }}
            </span>
          </div>
          <div v-if="rule.rule_set && (Array.isArray(rule.rule_set) ? rule.rule_set.length : rule.rule_set)">
            <span class="font-medium text-gray-700 dark:text-gray-300">Rule Set:</span>
            <span class="ml-2 text-gray-900 dark:text-gray-100">
              {{ Array.isArray(rule.rule_set) ? rule.rule_set.join(', ') : rule.rule_set }}
            </span>
          </div>
          <div v-if="rule.port?.length">
            <span class="font-medium text-gray-700 dark:text-gray-300">Port:</span>
            <span class="ml-2 text-gray-900 dark:text-gray-100">{{ rule.port.join(', ') }}</span>
          </div>
          <div v-if="rule.sniffer">
            <span class="font-medium text-gray-700 dark:text-gray-300">Sniffer:</span>
            <span class="ml-2 text-gray-900 dark:text-gray-100">
              {{ Array.isArray(rule.sniffer) ? rule.sniffer.join(', ') : rule.sniffer }}
            </span>
          </div>
          <div v-if="rule.timeout">
            <span class="font-medium text-gray-700 dark:text-gray-300">Timeout:</span>
            <span class="ml-2 text-gray-900 dark:text-gray-100">{{ rule.timeout }}</span>
          </div>
        </div>
      </div>
      <div class="flex space-x-2 ml-4">
        <button
          @click="handleEdit"
          class="text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300"
        >
          Edit
        </button>
        <button
          @click="handleDelete"
          class="text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300"
        >
          Delete
        </button>
      </div>
    </div>
  </div>
</template>
