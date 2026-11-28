<script setup lang="ts">
import { computed, ref, useId, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Popover from '../volt/Popover.vue'
import Button from './Button.vue'
import ProxyGroupCard from './ProxyGroupCard.vue'
import SaveFeedback from './SaveFeedback.vue'
import SubscriptionSource from './SubscriptionSource.vue'
import ProxyNodeHistory from './ProxyNodeHistory.vue'
import type { Outbound } from '../types/api'
import { useRuntimeProxies } from '../composables/useRuntimeProxies'
import { useServiceStore } from '../stores/service'
import { useNodeQualityStore } from '../stores/nodeQuality'
import { displayName } from '../utils/proxyGroups'
import { apiErrorMessage } from '../utils/apiErrorMessage'

const props = defineProps<{ name: string; outbound?: Outbound; external?: boolean }>()
const { t } = useI18n()
const panel = ref<InstanceType<typeof Popover>>()
const panelId = useId()
const visible = ref(false)
const expanded = ref(true)
const actionError = ref('')
const message = ref('')
const inspectedName = ref(props.name)
const trail = ref<string[]>([])
let actionGeneration = 0
watch(() => props.name, name => { inspectedName.value = name; trail.value = []; expanded.value = true; actionGeneration++; actionError.value = ''; message.value = '' })
const service = useServiceStore()
const quality = useNodeQualityStore()
const running = computed(() => service.status?.status === 'running')
const enabled = computed(() => visible.value && running.value)
const { view, loading, error, testing, testingNodes, switching, load, select, testGroup, testNode } =
  useRuntimeProxies(enabled, () => t('proxies.loadFailed'))
const group = computed(() => view.value?.groups.find(item => item.name === inspectedName.value))
const parents = computed(() => view.value?.groups.filter(item => item.members.some(member => member.name === props.name)) ?? [])
const member = computed(() => parents.value.flatMap(item => item.members).find(item => item.name === props.name))
const server = computed(() => props.outbound && 'server' in props.outbound ? String(props.outbound.server) : '')
const port = computed(() => props.outbound && 'server_port' in props.outbound ? String(props.outbound.server_port) : '')
const switchingTo = computed(() => switching.value.split('\u0000')[1] ?? '')

function opened() {
  expanded.value = true
  inspectedName.value = props.name
  trail.value = []
  actionGeneration++
  actionError.value = ''; message.value = ''
  visible.value = true
  if (!quality.ready) void quality.load()
}
async function run(action: () => Promise<void>, fallback: string, success = '') {
  const generation = actionGeneration
  actionError.value = ''; message.value = ''
  try { await action(); if (generation === actionGeneration) message.value = success }
  catch (err) { if (generation === actionGeneration) actionError.value = apiErrorMessage(err, fallback) }
}
function choose(name: string) {
  if (!group.value?.switchable || switching.value || name === group.value.now) return
  void run(() => select(inspectedName.value, name), t('proxies.selectFailed'), t('connections.details.switchedTo', { name: displayName(name) }))
}
function browse(name: string) {
  trail.value = [...trail.value, inspectedName.value]
  inspectedName.value = name; expanded.value = true; actionGeneration++; actionError.value = ''; message.value = ''
}
function back() {
  const previous = trail.value.at(-1)
  if (!previous) return
  trail.value = trail.value.slice(0, -1)
  inspectedName.value = previous; expanded.value = true; actionGeneration++; actionError.value = ''; message.value = ''
}
let externalAnchor: HTMLElement | null = null
function showAt(anchor: HTMLElement) {
  if (visible.value && externalAnchor === anchor) { panel.value?.hide(); return }
  externalAnchor = anchor
  panel.value?.showAt(anchor)
}
defineExpose({ showAt })
</script>

<template>
  <span class="inline-flex max-w-full align-middle">
    <button v-if="!external" type="button" class="inline-flex min-h-8 max-w-full items-center rounded-control text-left text-primary-600 underline decoration-primary-300 underline-offset-2 hover:decoration-current focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary-500 dark:text-primary-400"
      :aria-label="t('connections.details.openProxyGroup', { name })" :title="t('connections.details.openProxyGroup', { name })"
      :aria-expanded="visible" :aria-controls="panelId" aria-haspopup="dialog" @mousedown.stop @click.stop="panel?.toggle($event)">
      {{ displayName(name) }}
    </button>
    <Popover ref="panel" :id="panelId" :aria-label="t('connections.details.openProxyGroup', { name })"
      panel-class="w-[36rem] max-h-[80dvh]" content-class="p-3" @show="opened" @hide="visible = false">
      <div class="mb-2 flex items-start justify-between gap-3">
        <p class="text-xs text-text-secondary">{{ t('connections.details.switchScope') }}</p>
        <Button size="sm" variant="ghost" @click="panel?.hide()">{{ t('common.close') }}</Button>
      </div>
      <button v-if="trail.length" type="button" class="mb-2 min-h-8 rounded-control px-2 text-sm text-primary-600 focus-visible:outline-2 focus-visible:outline-primary-500" @click="back">← {{ trail.at(-1) }}</button>
      <p v-if="!running" class="py-4 text-sm text-text-secondary" role="status">{{ t('connections.stopped') }}</p>
      <div v-else>
        <div v-if="error" class="mb-2 flex items-center gap-2">
          <p class="min-w-0 flex-1 break-words text-sm text-danger" role="alert">{{ error }}</p>
          <Button size="sm" :loading="loading" @click="load()">{{ t('proxies.retry') }}</Button>
        </div>
        <p v-if="loading && !view" class="min-h-32 py-4 text-sm text-text-secondary" role="status">{{ t('common.loading') }}</p>
        <ProxyGroupCard v-if="group" inspectable :group="group" :members="group.members" :open="expanded"
          :testing="testing.has(inspectedName)" :testing-nodes="testingNodes" :switching-to="switchingTo" :focused-node="inspectedName !== name ? name : ''"
          @toggle="expanded = !expanded" @select="choose" @inspect="browse" @automatic-changed="load(true)"
          @test="run(() => testGroup(inspectedName), t('proxies.testFailed'))"
          @test-node="node => run(() => testNode(node), t('proxies.testFailed'))" />
        <div v-else-if="view && (member || outbound)" class="space-y-3 rounded-surface border border-border p-3 text-sm">
          <div class="flex items-center gap-2"><strong class="min-w-0 flex-1 break-words">{{ displayName(name) }}</strong><ProxyNodeHistory v-if="member" :name="name" :history="member.history" /></div>
          <p class="text-text-secondary">{{ member?.type || outbound?.type }}</p>
          <dl v-if="server" class="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs">
            <dt class="text-text-secondary">{{ t('outbounds.table.server') }}</dt><dd class="break-all font-mono">{{ server }}</dd>
            <dt v-if="port" class="text-text-secondary">{{ t('outbounds.table.port') }}</dt><dd v-if="port" class="font-mono">{{ port }}</dd>
          </dl>
          <SubscriptionSource :name="name" />
          <Button v-if="member && !member.group" size="sm" :loading="testingNodes.has(name)" @click="run(() => testNode(name), t('proxies.testFailed'))">{{ t('proxies.testNode', { node: displayName(name) }) }} · {{ member.delay }} ms</Button>
          <div v-if="parents.length" class="space-y-1"><p class="text-xs text-text-secondary">{{ t('connections.details.memberOf') }}</p>
            <button v-for="parent in parents" :key="parent.name" type="button" class="mr-2 min-h-8 rounded-control border border-border px-2 text-primary-600 dark:text-primary-400" @click="browse(parent.name)">{{ parent.name }}</button>
          </div>
        </div>
        <p v-else-if="view && !loading" class="py-4 text-sm text-text-secondary" role="status">{{ t('connections.details.groupUnavailable') }}</p>
        <SaveFeedback class="mt-2" :error="actionError" :message="message" />
      </div>
    </Popover>
  </span>
</template>
