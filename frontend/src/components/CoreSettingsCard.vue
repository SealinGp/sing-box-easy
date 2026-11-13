<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { serviceControlService } from '../services'
import { setSingBoxVersion } from '../composables/useSingBoxVersion'
import { useConfirm } from '../composables/useConfirm'
import { ApiError, Code, type InstallTask } from '../types/api'
import Card from './Card.vue'
import Button from './Button.vue'
import Input from './Input.vue'
import Alert from './Alert.vue'
import Skeleton from './Skeleton.vue'
import StatusIndicator from './StatusIndicator.vue'

const { t } = useI18n()
const { confirm } = useConfirm()
const status = ref<{ installed: boolean; version: string } | null>(null)
const checking = ref(true)
const version = ref('')
const beta = ref(false)
const task = ref<InstallTask | null>(null)
const error = ref('')
const busy = ref(false)
const taskKey = 'sbe-core-install-task'
const pendingTask = ref<string | null>(null)
let timer: ReturnType<typeof setTimeout> | undefined
let disposed = false
const taskTone = computed(() => task.value?.status === 'failed' ? 'danger' : task.value?.status === 'completed' ? 'success' : 'info')

function rememberTask(id: string | null) {
  pendingTask.value = id
  try { if (id) localStorage.setItem(taskKey, id); else localStorage.removeItem(taskKey) } catch { /* Storage is optional. */ }
}
async function refresh() {
  checking.value = true
  try {
    const { data } = await serviceControlService.getInstallStatus()
    if (disposed) return
    status.value = data
    setSingBoxVersion(data.installed ? data.version : undefined)
    if (!version.value && data.installed) version.value = data.version
  } catch (err) { if (!disposed) error.value = err instanceof Error ? err.message : t('workspace.core.statusFailed') }
  finally { checking.value = false }
}
async function poll(id: string) {
  try {
    const { data } = await serviceControlService.getInstallTask(id)
    if (disposed) return
    task.value = data
    if (data.status === 'completed' || data.status === 'failed') {
      busy.value = false
      rememberTask(null)
      if (data.status === 'completed') await refresh()
      return
    }
    timer = setTimeout(() => void poll(id), 2000)
  } catch (err) {
    if (disposed) return
    busy.value = false
    error.value = err instanceof Error ? err.message : t('workspace.core.taskFailed')
    if (err instanceof ApiError && err.code === Code.NotFound) rememberTask(null)
    // Keep the task ID so refreshing or returning to Settings can reconnect.
  }
}
async function install(update = false) {
  if (busy.value || pendingTask.value || !status.value) return
  if (!(await confirm({ message: t('workspace.core.confirm'), confirmLabel: t(update ? 'workspace.core.update' : 'workspace.core.reinstall') }))) return
  if (disposed) return
  busy.value = true
  error.value = ''
  task.value = null
  try {
    const target = version.value.trim() || undefined
    const { data } = await (update
      ? serviceControlService.updateSingBox(target, beta.value)
      : serviceControlService.installSingBox(target, beta.value))
    rememberTask(data.task_id)
    if (disposed) return
    task.value = { id: data.task_id, status: 'running', message: data.message }
    await poll(data.task_id)
  } catch (err) {
    if (disposed) return
    busy.value = false
    error.value = err instanceof Error ? err.message : t('workspace.core.failed')
  }
}
onMounted(async () => {
  await refresh()
  let id: string | null = null
  try { id = localStorage.getItem(taskKey) } catch { /* Storage is optional. */ }
  if (id && !disposed) { pendingTask.value = id; busy.value = true; await poll(id) }
})
onBeforeUnmount(() => { disposed = true; clearTimeout(timer) })
</script>

<template>
  <Card id="sing-box-core" :title="t('workspace.core.title')" class="space-y-4 scroll-mt-6">
    <p class="text-sm text-text-secondary">{{ t('workspace.core.description') }}</p>
    <div class="min-h-6" :aria-busy="checking">
      <Skeleton v-if="checking" class="h-5 w-40" :aria-label="t('workspace.core.checking')" />
      <div v-else-if="status" class="flex items-center justify-between gap-3">
        <StatusIndicator :variant="status.installed ? 'success' : 'secondary'">{{ t(status.installed ? 'workspace.core.installed' : 'workspace.core.missing') }}</StatusIndicator>
        <span v-if="status.installed" class="font-mono text-sm text-text-primary">{{ status.version }}</span>
      </div>
    </div>
    <Input v-model="version" :label="t('workspace.core.version')" :hint="t('workspace.core.versionHint')" placeholder="1.14.0" :disabled="busy || checking" />
    <label class="flex items-center gap-2 text-sm text-text-primary"><input v-model="beta" type="checkbox" :disabled="busy || checking" class="h-4 w-4 accent-primary" />{{ t('workspace.core.beta') }}</label>
    <Alert v-if="error" type="error" closable @close="error = ''">{{ error }}</Alert>
    <div v-if="task || busy" class="min-h-24 space-y-2 rounded-control border border-border bg-bg-tertiary p-3" :aria-busy="busy">
      <StatusIndicator :variant="taskTone" :busy="busy">{{ t(busy ? 'workspace.core.progress' : task?.status === 'completed' ? 'workspace.core.done' : task?.status === 'failed' ? 'workspace.core.failed' : 'workspace.core.taskFailed') }}</StatusIndicator>
      <pre role="log" aria-live="polite" class="max-h-40 whitespace-pre-wrap break-words font-mono text-xs text-text-secondary">{{ task?.error || task?.message || t('common.loading') }}</pre>
    </div>
    <div class="flex flex-wrap gap-2">
      <Button :loading="busy" :disabled="checking || !status || !!pendingTask" @click="install()">{{ t(status?.installed ? 'workspace.core.reinstall' : 'workspace.core.install') }}</Button>
      <Button v-if="status?.installed" variant="secondary" :disabled="busy || checking || !!pendingTask" @click="install(true)">{{ t('workspace.core.update') }}</Button>
      <Button v-if="pendingTask && !busy" variant="secondary" @click="busy = true; error = ''; poll(pendingTask)">{{ t('workspace.core.reconnect') }}</Button>
      <Button variant="ghost" :disabled="busy || checking" @click="refresh">{{ t('workspace.core.refresh') }}</Button>
    </div>
  </Card>
</template>
