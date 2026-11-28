<script setup lang="ts">
import { computed } from 'vue'
import type { TopologyExit } from '../types/routeTopology'
const props = defineProps<{ exits: TopologyExit[] }>()
const emit = defineEmits<{ inspect: [name: string, anchor: HTMLElement] }>()
const available = computed(() => props.exits.filter(exit => exit.kind === 'outbound'))
</script>

<template>
  <div v-if="available.length" class="mb-2 min-w-0 md:hidden">
    <p class="mb-1 text-xs text-text-secondary">{{ $t('routeFlow.columns.outbounds') }}</p>
    <div class="flex gap-2 overflow-x-auto pb-1">
      <button v-for="exit in available" :key="exit.id" type="button"
        class="min-h-10 shrink-0 rounded-control border border-border px-3 text-sm text-primary-600 focus-visible:outline-2 focus-visible:outline-primary-500 dark:text-primary-400"
        :aria-label="$t('connections.details.openProxyGroup', { name: exit.label })" aria-haspopup="dialog"
        @click.stop="emit('inspect', exit.id, $event.currentTarget as HTMLElement)">
        {{ exit.label }} <span aria-hidden="true">›</span>
      </button>
    </div>
  </div>
</template>
