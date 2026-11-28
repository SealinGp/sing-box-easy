<script setup lang="ts">
import { computed, provide, reactive, ref, shallowReactive, watch } from 'vue'
import { routeLocationKey, type RouteLocationNormalizedLoaded } from 'vue-router'
import { workspaceEditKey, workspaceKey } from '../composables/workspaceContext'
const props = defineProps<{ route: RouteLocationNormalizedLoaded; active: boolean }>()
const emit = defineEmits<{ dirty: [value: boolean] }>()
// Each page sees its own route, including nested RouterViews and query links.
// A hidden DNS editor must not react to the active Connections URL.
const pageRoute = shallowReactive({ ...props.route })
watch(() => props.route, route => Object.assign(pageRoute, route))
provide(routeLocationKey, pageRoute)
const dirty = reactive(new Map<symbol, boolean>())
const inlineDirty = ref(false)
const editVersion = ref(0)
provide(workspaceKey, { active: computed(() => props.active), dirty, inlineDirty, editVersion })
function markEdited(target?: Element | null) {
  if (target?.closest('fieldset, form')) {
    inlineDirty.value = true
    editVersion.value++
  }
}
provide(workspaceEditKey, markEdited)
function edited(event: Event) { markEdited(event.target instanceof Element ? event.target : null) }
watch(() => inlineDirty.value || [...dirty.values()].some(Boolean), value => emit('dirty', value), { flush: 'sync' })
</script>
<template>
  <section @input.capture="edited" @change.capture="edited" v-show="active" :inert="!active" class="workspace-pane" :class="{ 'workspace-pane-fill': pageRoute.path === '/dashboard/config' }">
    <RouterView :route="pageRoute" />
  </section>
</template>
