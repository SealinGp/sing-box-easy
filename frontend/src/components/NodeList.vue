<script setup lang="ts">
import type { Outbound } from '../types/api'
import Badge from './Badge.vue'

export interface Props {
  nodes: Outbound[]
  maxHeight?: string
  selectable?: boolean
  selectedTags?: Set<string>
}

const props = withDefaults(defineProps<Props>(), {
  maxHeight: 'max-h-64',
  selectable: false,
})

const emit = defineEmits<{
  select: [tag: string]
}>()

// Helper to get server info from outbound (handles different types)
const getServerInfo = (node: Outbound): string | null => {
  // Type assertion to access server properties that exist on proxy outbounds
  const proxyNode = node as Outbound & { server?: string; server_port?: number }
  if (proxyNode.server) {
    return proxyNode.server_port
      ? `${proxyNode.server}:${proxyNode.server_port}`
      : proxyNode.server
  }
  return null
}

const handleClick = (tag: string) => {
  if (props.selectable) {
    emit('select', tag)
  }
}
</script>

<template>
  <div :class="[maxHeight, 'overflow-y-auto border border-gray-200 rounded-lg']">
    <div
      v-for="node in nodes"
      :key="node.tag"
      :class="[
        'flex items-center gap-3 p-3 hover:bg-gray-50 border-b border-gray-100 last:border-0',
        selectable ? 'cursor-pointer' : ''
      ]"
      @click="handleClick(node.tag)"
    >
      <input
        v-if="selectable && selectedTags"
        type="checkbox"
        :checked="selectedTags.has(node.tag)"
        class="w-4 h-4 text-violet-600 border-gray-300 rounded focus:ring-violet-500"
        @click.stop="handleClick(node.tag)"
      />
      <div class="flex-1 min-w-0">
        <div class="flex items-center gap-2">
          <p class="text-sm font-medium text-gray-900 truncate">
            {{ node.tag }}
          </p>
          <Badge variant="gray" size="sm">{{ node.type }}</Badge>
        </div>
        <p v-if="getServerInfo(node)" class="text-xs text-gray-500 truncate">
          {{ getServerInfo(node) }}
        </p>
      </div>
    </div>
    <div v-if="nodes.length === 0" class="p-4 text-center text-sm text-gray-500">
      No nodes available
    </div>
  </div>
</template>
