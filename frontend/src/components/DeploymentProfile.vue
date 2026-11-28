<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Button from './Button.vue'
import Input from './Input.vue'
import SaveFeedback from './SaveFeedback.vue'
import { ContextPanel } from '../volt'
import { configService, serviceControlService } from '../services'
import { apiService } from '../services/api'
import { trafficPolicyService, type ManagedPolicy } from '../services/trafficPolicy'
import type { BasicResponse, SingBoxConfig } from '../types/api'
import { apiErrorMessage } from '../utils/apiErrorMessage'

const emit = defineEmits<{ applied: [] }>()
const { t } = useI18n()
const visible = ref(false)
const role = ref<'exit' | 'client'>('exit')
const port = ref('0')
const subscription = ref('')
const lan = ref(false)
const lanAddress = ref('')
const busy = ref(false)
const error = ref('')
const saved = ref(false)
const complete = ref(false)
const plan = ref<{ config: SingBoxConfig; revision: string; policies: ManagedPolicy[]; nodes: number; port: number; oldInbounds: number; oldOutbounds: number }>()
function open() { if (busy.value) return; visible.value = true; plan.value = undefined; saved.value = false; complete.value = false; error.value = '' }
async function preview() {
  if (busy.value) return
  error.value = ''
  if (role.value === 'exit' && (!/^\d{1,5}$/.test(port.value) || Number(port.value) > 65535)) { error.value = t('deploymentProfile.invalidPort'); return }
  if (role.value === 'client' && !subscription.value.trim()) { error.value = t('deploymentProfile.subscriptionRequired'); return }
  if (role.value === 'client' && lan.value && !lanAddress.value.trim()) { error.value = t('deploymentProfile.lanAddress'); return }
  busy.value = true
  try {
    const generated = await apiService.post<BasicResponse<{ config: SingBoxConfig; node_count: number; port: number }>>('/config/profile', {
        role: role.value, port: role.value === 'exit' ? Number(port.value) : 0,
        subscription: role.value === 'client' ? subscription.value.trim() : '',
        lan_address: role.value === 'client' && lan.value ? lanAddress.value.trim() : '',
      })
    const before = await trafficPolicyService.impact({})
    const current = await configService.getConfig()
    const result = generated.data.data
    const config = { ...current.data, ...result.config, log: current.data.log ?? result.config.log }
    const impact = await trafficPolicyService.impact(config)
    if (before.revision !== impact.revision) throw new Error(t('deploymentProfile.changed'))
    await configService.validateConfig(config)
    plan.value = { config, ...impact, nodes: result.node_count, port: result.port, oldInbounds: current.data.inbounds?.length ?? 0, oldOutbounds: current.data.outbounds?.length ?? 0 }
  } catch (err) { error.value = apiErrorMessage(err, t('deploymentProfile.failed')) }
  finally { busy.value = false }
}
async function apply() {
  if (!plan.value || busy.value) return
  busy.value = true; error.value = ''
  try {
    if (!saved.value) { await configService.saveConfig(plan.value.config, plan.value.revision); saved.value = true; emit('applied') }
    await serviceControlService.restartService()
    complete.value = true
  } catch (err) { error.value = apiErrorMessage(err, t('deploymentProfile.failed')) }
  finally { busy.value = false }
}
</script>

<template>
  <Button variant="secondary" :disabled="busy" @click="open">{{ t('deploymentProfile.title') }}</Button>
  <ContextPanel :visible="visible" :header="t('deploymentProfile.title')" class="max-w-2xl"
    :closable="!busy" :dismissable="!busy" :close-on-escape="!busy" @update:visible="value => { if (!busy) visible = value }">
    <fieldset v-if="!plan" :disabled="busy" :inert="busy" class="min-w-0 space-y-4">
      <div class="flex flex-wrap gap-3">
        <label v-for="value in (['exit', 'client'] as const)" :key="value" class="flex min-h-10 cursor-pointer items-center gap-2 rounded-control border border-border px-3">
          <input v-model="role" type="radio" :value="value" name="deployment-role" />{{ t(`deploymentProfile.${value}`) }}
        </label>
      </div>
      <template v-if="role === 'exit'">
        <p class="text-sm text-text-secondary">{{ t('deploymentProfile.exitSummary') }}</p>
        <label class="block text-sm">{{ t('deploymentProfile.port') }}<Input v-model="port" inputmode="numeric" class="mt-1" /></label>
      </template>
      <template v-else>
        <label class="block text-sm">{{ t('deploymentProfile.subscription') }}<Input v-model="subscription" class="mt-1" placeholder="https://…" /></label>
        <p class="text-xs text-text-secondary">{{ t('deploymentProfile.importHint') }}</p>
        <label class="flex min-h-9 items-center gap-2 text-sm"><input v-model="lan" type="checkbox" />{{ t('deploymentProfile.lan') }}</label>
        <label v-if="lan" class="block text-sm">{{ t('deploymentProfile.lanAddress') }}<Input v-model="lanAddress" class="mt-1" placeholder="192.168.1.2" /></label>
        <p class="text-sm text-text-secondary">{{ t('deploymentProfile.clientSummary') }}</p>
      </template>
    </fieldset>
    <div v-else class="space-y-3 text-sm">
      <p>{{ role === 'exit' ? t('deploymentProfile.exitPlan', { port: plan.port }) : t('deploymentProfile.clientPlan', { count: plan.nodes }) }}</p>
      <p class="rounded-control border border-amber-400/50 p-3 text-amber-700 dark:text-amber-300">{{ t('deploymentProfile.replace', { inbounds: plan.oldInbounds, outbounds: plan.oldOutbounds }) }}</p>
      <p v-if="role === 'client'">{{ t('deploymentProfile.routingImpact') }}</p>
      <p v-if="lan && role === 'client'">{{ t('deploymentProfile.gatewayHint') }}</p>
      <div v-if="plan.policies.length"><strong>{{ t('managed.rawImpact') }}</strong><p v-for="policy in plan.policies" :key="policy.id">{{ policy.name }}</p></div>
      <details><summary class="cursor-pointer py-2">{{ t('deploymentProfile.details') }}</summary><pre class="max-h-64 overflow-auto whitespace-pre-wrap break-all rounded-control bg-bg-secondary p-2 text-xs">{{ JSON.stringify(plan.config, null, 2) }}</pre></details>
      <p v-if="saved" role="status">{{ t(complete ? 'deploymentProfile.done' : 'deploymentProfile.saved') }}</p>
    </div>
    <SaveFeedback :error="error" class="mt-3" />
    <RouterLink v-if="error" to="/dashboard/settings" class="mt-2 inline-block text-sm text-primary-600 underline">{{ t('deploymentProfile.installCore') }}</RouterLink>
    <template #footer>
      <Button v-if="plan && !saved" variant="secondary" :disabled="busy" @click="plan = undefined; error = ''">{{ t('deploymentProfile.back') }}</Button>
      <Button v-if="!plan" :loading="busy" @click="preview">{{ t('deploymentProfile.preview') }}</Button>
      <Button v-else-if="!complete" :loading="busy" @click="apply">{{ t(saved ? 'deploymentProfile.retry' : 'deploymentProfile.apply') }}</Button>
      <Button v-else @click="visible = false">{{ t('common.close') }}</Button>
    </template>
  </ContextPanel>
</template>
