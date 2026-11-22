<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import Skeleton from './Skeleton.vue'
import { columnCount } from '../utils/proxyGroups'
const root = ref<HTMLElement | null>(null)
const width = ref(0)
const columns = computed(() => columnCount(width.value))
let observer: ResizeObserver | undefined
onMounted(() => {
  if (!root.value) return
  width.value = root.value.clientWidth
  observer = new ResizeObserver(([entry]) => { width.value = entry?.contentRect.width ?? 0 })
  observer.observe(root.value)
})
onBeforeUnmount(() => observer?.disconnect())
</script>
<template>
  <div ref="root" role="status" :aria-label="$t('common.loading')" :aria-busy="true" class="flex items-start gap-3">
    <div v-for="column in columns" :key="column" class="flex min-w-0 flex-1 basis-0 flex-col gap-3">
      <div v-for="card in 2" :key="card" class="rounded-surface border border-border bg-bg-secondary p-3">
        <div class="mb-3 flex items-center gap-2">
          <Skeleton class="h-4 w-28" /><Skeleton class="h-3 w-14" /><Skeleton class="ml-auto h-6 w-14" />
        </div>
        <Skeleton class="mb-3 h-3 w-44 max-w-full" />
        <div v-if="card === 1" class="grid grid-cols-[repeat(auto-fill,minmax(150px,1fr))] gap-1.5">
          <div v-for="node in 6" :key="node" class="space-y-3 rounded-control border border-border p-2">
            <Skeleton class="h-3 w-3/4" />
            <div class="flex justify-between gap-2"><Skeleton class="h-3 w-10" /><Skeleton class="h-4 w-12" /></div>
          </div>
        </div>
        <div v-else class="flex gap-1"><Skeleton v-for="node in 12" :key="node" class="h-3 w-3" /></div>
      </div>
    </div>
  </div>
</template>
