<script setup lang="ts">
import { defineAsyncComponent, h, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Button from './Button.vue'
import Alert from './Alert.vue'
import Skeleton from './Skeleton.vue'
import { ContextPanel } from '../volt'
const props = defineProps<{ kind: 'dns' | 'inbounds' | 'routes' | 'ruleSets' }>()
const emit = defineEmits<{ applied: [] }>()
const { t } = useI18n()
const visible = ref(false)
const loadingComponent = { render: () => h('div', { class: 'space-y-3', role: 'status' }, [h(Skeleton, { class: 'h-16 w-full' }), h(Skeleton, { class: 'h-48 w-full' })]) }
const editors = {
  dns: defineAsyncComponent({ loader: () => import('./presets/DNSPresets.vue'), loadingComponent }),
  inbounds: defineAsyncComponent({ loader: () => import('./presets/InboundPresets.vue'), loadingComponent }),
  routes: defineAsyncComponent({ loader: () => import('./presets/RoutePresets.vue'), loadingComponent }),
  ruleSets: defineAsyncComponent({ loader: () => import('./presets/RuleSetPresets.vue'), loadingComponent }),
}
</script>
<template>
  <Button variant="secondary" @click="visible = true">{{ t('workspace.presets.button') }}</Button>
  <ContextPanel v-model:visible="visible" :header="t(`workspace.presets.${props.kind}`)" class="w-full max-w-3xl">
    <div v-if="visible" class="space-y-4">
      <Alert type="warning">{{ t(`workspace.presets.${props.kind}Impact`) }}</Alert>
      <p v-if="kind === 'dns' || kind === 'routes'" class="text-sm text-text-secondary">
        {{ t('workspace.presets.dependencies') }}
        <RouterLink to="/dashboard/route/rule-sets" class="ml-1 text-primary hover:underline">{{ t('route.ruleSets.title') }}</RouterLink>
        <span aria-hidden="true"> · </span><RouterLink to="/dashboard/outbounds/list" class="text-primary hover:underline">{{ t('nav.outbounds') }}</RouterLink>
      </p>
      <component :is="editors[kind]" @applied="emit('applied')" />
    </div>
  </ContextPanel>
</template>
