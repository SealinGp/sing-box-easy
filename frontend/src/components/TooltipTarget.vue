<script setup lang="ts">
import { ref } from 'vue'
import Tooltip from './Tooltip.vue'
defineOptions({ inheritAttrs: false })
withDefaults(defineProps<{ as?: string; text?: string }>(), { as: 'span' })
const anchor = ref<HTMLElement | SVGElement>()
</script>
<template>
  <component :is="as" ref="anchor" v-bind="$attrs">
    <slot />
    <!-- Reset SVG namespace before teleporting an HTML bubble to body. The
         hidden bridge contributes no geometry to the diagram's group. -->
    <foreignObject v-if="as === 'g'" style="display: none">
      <Tooltip :target="text ? anchor : null"><template #content>{{ text }}</template></Tooltip>
    </foreignObject>
    <Tooltip v-else :target="text ? anchor : null"><template #content>{{ text }}</template></Tooltip>
  </component>
</template>
