<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Button from './Button.vue'
import Dialog from '../volt/Dialog.vue'
import { userService, serviceControlService } from '../services'
import { ensureDeployment, useDeployment } from '../composables/useDeployment'
import { detectSubscriptionOrphans, cleanupSubscriptionOrphans, type OrphanPlan, type OrphanCleanupResult } from '../services/subscriptionOrphans'

const props = defineProps<{ refreshKey: string }>()
const emit = defineEmits<{ cleaned: [] }>()
const { t } = useI18n()
const { authEnabled } = useDeployment()
const plan = ref<OrphanPlan>()
const result = ref<OrphanCleanupResult>()
const visible = ref(false)
const loading = ref(false)
const busy = ref(false)
const error = ref('')
const canClean = ref(false)
let request = 0
const message = (e: unknown) => e instanceof Error ? e.message : String(e)
async function check() {
  if (busy.value) return
  const current = ++request
  loading.value = true
  error.value = ''
  try {
    const value = await detectSubscriptionOrphans()
    if (current === request) plan.value = value
  } catch (e) {
    if (current === request) { plan.value = undefined; error.value = message(e) }
  } finally { if (current === request) loading.value = false }
}
async function clean() {
  if (!plan.value || busy.value || loading.value || plan.value.blockers.length) return
  busy.value = true
  error.value = ''
  try {
    result.value = await cleanupSubscriptionOrphans(plan.value.revision)
    visible.value = false
    emit('cleaned')
  } catch (e) {
    error.value = message(e)
    // A stale plan must be reviewed again before another submission.
    plan.value = undefined
  } finally {
    busy.value = false
  }
  if (!error.value) await check()
}
async function restart() {
  busy.value = true
  error.value = ''
  try {
    await serviceControlService.restartService()
    if (result.value) result.value.restart_required = false
  } catch (e) { error.value = message(e) }
  finally { busy.value = false }
}
watch(() => props.refreshKey, check, { immediate: true })
onMounted(async () => {
  try {
    await ensureDeployment()
    canClean.value = !authEnabled.value || (await userService.getMe()).role === 'admin'
  } catch { canClean.value = false }
})
</script>

<template>
  <section v-if="loading || error || plan?.node_count || result" class="mb-4 space-y-3 rounded-control border border-border bg-bg-secondary p-4">
    <p v-if="loading" role="status" class="text-sm text-text-secondary">{{ t('orphanSubscriptions.checking') }}</p>
    <div v-if="plan?.node_count" class="flex flex-wrap items-center justify-between gap-3">
      <p class="text-sm">{{ t('orphanSubscriptions.found', { subscriptions: plan.subscriptions.length, nodes: plan.node_count }) }}</p>
      <Button variant="secondary" size="sm" :disabled="busy || loading" @click="visible = true">{{ t('orphanSubscriptions.review') }}</Button>
    </div>
    <div v-if="result" role="status" class="space-y-2 text-sm">
      <p>{{ t('orphanSubscriptions.saved', { count: result.removed_nodes }) }}</p>
      <p class="break-all text-xs text-text-secondary">{{ t('orphanSubscriptions.backup') }}: {{ result.backup_path }}</p>
      <div v-if="result.restart_required" class="flex flex-wrap items-center gap-3">
        <span>{{ t('orphanSubscriptions.restartHint') }}</span>
        <Button size="sm" :loading="busy" @click="restart">{{ t('orphanSubscriptions.restart') }}</Button>
      </div>
      <p v-else>{{ t('orphanSubscriptions.applied') }}</p>
    </div>
    <div v-if="error && !visible" role="alert" class="space-y-2 text-sm text-red-600">
      <p>{{ error }}</p>
      <Button variant="ghost" size="sm" :disabled="busy" @click="check">{{ t('common.refresh') }}</Button>
    </div>
    <Dialog v-model:visible="visible" modal class="w-full max-w-2xl" :header="t('orphanSubscriptions.title')" :closable="!busy" :close-on-escape="!busy">
      <div v-if="plan" class="space-y-4">
        <p class="text-sm">{{ t('orphanSubscriptions.explanation') }}</p>
        <div class="max-h-[45vh] space-y-3 overflow-y-auto">
          <details v-for="subscription in plan.subscriptions" :key="subscription.id" class="rounded-control border border-border p-3">
            <summary class="cursor-pointer break-all text-sm">{{ subscription.id }} · {{ t('orphanSubscriptions.nodes', { count: subscription.nodes.length }) }}</summary>
            <ul class="mt-2 space-y-1 text-xs text-text-secondary"><li v-for="node in subscription.nodes" :key="node" class="break-all">{{ node }}</li></ul>
          </details>
          <div v-if="plan.updated_groups.length">
            <h3 class="text-sm font-medium">{{ t('orphanSubscriptions.groups') }}</h3>
            <ul class="text-xs"><li v-for="group in plan.updated_groups" :key="group" class="break-all">{{ group }}</li></ul>
          </div>
          <div v-if="plan.removed_groups.length">
            <h3 class="text-sm font-medium">{{ t('orphanSubscriptions.emptyGroups') }}</h3>
            <ul class="text-xs"><li v-for="group in plan.removed_groups" :key="group" class="break-all">{{ group }}</li></ul>
          </div>
          <div v-if="plan.blockers.length" role="alert" class="text-sm text-red-600">
            <p>{{ t('orphanSubscriptions.blocked') }}</p>
            <ul class="mt-2 text-xs"><li v-for="item in plan.blockers" :key="item" class="break-all">{{ item }}</li></ul>
          </div>
        </div>
        <p class="text-xs text-text-secondary">{{ t('orphanSubscriptions.consequence') }}</p>
        <p v-if="!canClean" class="text-sm">{{ t('orphanSubscriptions.adminOnly') }}</p>
      </div>
      <p v-if="error" role="alert" class="mt-3 text-sm text-red-600">{{ error }}</p>
      <template #footer>
        <Button variant="ghost" :disabled="busy" @click="visible = false">{{ t('common.cancel') }}</Button>
        <Button variant="secondary" :disabled="busy" :loading="loading" @click="check">{{ t('common.refresh') }}</Button>
        <Button v-if="canClean" :loading="busy" :disabled="loading || !plan?.node_count || !!plan?.blockers.length" @click="clean">{{ t('orphanSubscriptions.clean') }}</Button>
      </template>
    </Dialog>
  </section>
</template>
