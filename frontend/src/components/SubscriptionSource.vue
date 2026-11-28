<script setup lang="ts">
import { computed } from 'vue'
import { nodeSubscriptionId } from '../utils/nodeSubscription'
import { useSubscriptionNames } from '../composables/useSubscriptionNames'
import { FOCUS_PAGES, focusLink } from '../utils/focusTarget'
const props = defineProps<{ name: string }>()
const id = computed(() => nodeSubscriptionId(props.name))
const names = useSubscriptionNames()
</script>
<template>
  <RouterLink v-if="id" :to="focusLink(FOCUS_PAGES.subscriptions, id)" :title="names[id] || id"
    class="block min-w-0 truncate rounded-sm text-primary-600 underline decoration-dotted underline-offset-2 focus-visible:outline-2 focus-visible:outline-primary-500 dark:text-primary-400"
    @click.stop @mousedown.stop>{{ names[id] || id }}</RouterLink>
  <span v-else class="text-text-secondary">—</span>
</template>
