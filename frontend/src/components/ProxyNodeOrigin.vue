<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { subscriptionService } from '../services'
import { nodeNameWithoutSubscription, nodeSubscriptionId } from '../utils/nodeSubscription'
import { FOCUS_PAGES, focusLink } from '../utils/focusTarget'

const props = defineProps<{ name: string }>()
const id = computed(() => nodeSubscriptionId(props.name))
const subscriptionName = ref('')

watch(id, async (value, _, onCleanup) => {
  let active = true
  onCleanup(() => { active = false })
  subscriptionName.value = ''
  if (!value) return
  try {
    const { data } = await subscriptionService.getSubscriptionByID(value)
    if (active) subscriptionName.value = data.name
  } catch {
    // Deleted or unavailable subscriptions retain their ownership ID.
  }
}, { immediate: true })
</script>

<template>
  <span class="break-words">
    {{ nodeNameWithoutSubscription(name) }}<template v-if="id"> - <RouterLink
      :to="focusLink(FOCUS_PAGES.subscriptions, id)"
      class="rounded-sm underline decoration-dotted underline-offset-2 hover:decoration-solid focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary-500"
      @click.stop
      @keydown.enter.stop
      @keydown.space.stop
    >{{ subscriptionName || id }}</RouterLink></template>
  </span>
</template>
