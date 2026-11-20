<script setup lang="ts">
import { computed } from 'vue'
import RoutingConditionNode from './RoutingConditionNode.vue'
import { routeConditionTree } from '../utils/routeRulePresentation'

const props = defineProps<{ rule: Record<string, unknown>; compact?: boolean }>()
const condition = computed(() => routeConditionTree(props.rule))
</script>

<template>
  <RoutingConditionNode :node="condition" :compact="compact" />
</template>
