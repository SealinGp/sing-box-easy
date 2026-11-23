<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Button, Card, Skeleton, IconButton } from '../../components'
import Select from '../../volt/Select.vue'
import Popover from '../../volt/Popover.vue'
import DNSGroupForm from '../../lab/DNSGroupForm.vue'
import ManagedPolicyEditor from '../../components/traffic/ManagedPolicyEditor.vue'
import ManagedPolicyList from '../../components/traffic/ManagedPolicyList.vue'
import ManagedPlanPreview from '../../components/traffic/ManagedPlanPreview.vue'
import { useManagedTraffic } from '../../composables/useManagedTraffic'

const { t } = useI18n()
const {
  groupsPage, snapshot, loading, busy, error, feedbackPolicyId,
  feedback, policy, conditions, plan, panelOpen,
  panel, panelKind, groupDraft, targetGroup, targetPolicy,
  replacement, replacementSearch, replacementSelect, dnsSearch, dnsAnchor,
  dnsSelect, dnsChoice, dependencies, missingServers, serialized,
  load, cancelPolicy, previewGroup, clearPlan, openPolicy,
  openGroup, quickGroup, createReplacement, cancelGroup, deleteGroup,
  policyAction, preview, previewPolicy, commit, panelClosed,
} = useManagedTraffic()
const editor = ref<InstanceType<typeof ManagedPolicyEditor>>()
function createGroup(event: Event) {
  dnsAnchor.value = editor.value?.dnsAnchor
  dnsSelect.value = editor.value?.dnsSelect
  void quickGroup(event)
}
</script>
<template>
  <section class="space-y-5">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="text-xl font-semibold">
          {{ t(groupsPage ? 'lab.resolverGroups' : 'lab.trafficPolicies') }}
        </h1>
        <p class="mt-2 text-sm text-text-secondary">
          {{ t('managed.description') }}
        </p>
      </div>
      <div class="flex gap-2">
        <Button variant="ghost" :disabled="busy" @click="load">{{
          t('managed.refresh')
        }}</Button
        ><Button v-if="groupsPage" @click="openGroup($event)">{{
          t('lab.createDnsGroup')
        }}</Button
        ><Button v-else :disabled="busy || !!policy" @click="openPolicy()">{{
          t('lab.addTrafficPolicy')
        }}</Button>
      </div>
    </header>
    <p v-if="feedback && !feedbackPolicyId" role="status" class="text-sm text-success">
      {{ feedback }}
    </p>
    <p v-if="error && !policy && !panelOpen" role="alert" class="text-sm text-danger">{{ error }}</p>
    <div v-if="loading" class="space-y-3" aria-busy="true">
      <Skeleton v-for="n in 3" :key="n" class="h-24 w-full" />
    </div>
    <template v-else-if="snapshot">
      <Card v-if="snapshot.recovery"
        ><h2 class="font-semibold">{{ t('managed.recovery') }}</h2>
        <p class="my-3 text-sm">{{ t('managed.recoveryHelp') }}</p>
        <div class="flex flex-wrap gap-2">
          <Button
            :disabled="busy"
            @click="preview({ action: 'recover_accept' })"
            >{{ t('managed.recoverAccept') }}</Button
          ><Button
            variant="secondary"
            :disabled="busy"
            @click="preview({ action: 'recover_discard' })"
            >{{ t('managed.recoverDiscard') }}</Button
          >
        </div></Card
      >
      <template v-if="groupsPage">
        <Card v-if="!snapshot.groups.length"
          ><p>{{ t('managed.noGroups') }}</p></Card
        >
        <Card v-for="g in snapshot.groups" :key="g.id"
          ><div class="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h2 class="font-semibold">{{ g.name }}</h2>
              <p class="mt-2 text-sm text-text-secondary">
                {{ g.servers.join(' + ') }} ·
                {{ t('lab.seconds', { n: g.timeout }) }}
              </p>
              <p class="mt-2 text-sm">
                {{
                  t('managed.usedBy', {
                    n: snapshot.policies.filter((p) => p.group_id === g.id)
                      .length,
                  })
                }}
              </p>
              <p
                v-if="missingServers(g).length"
                role="status"
                class="mt-2 text-sm text-danger"
              >
                {{
                  t('managed.missingServers', {
                    names: missingServers(g).join(', '),
                  })
                }}
              </p>
            </div>
            <div class="flex gap-1">
              <IconButton
                action="edit"
                :label="t('lab.editNamed', { name: g.name })"
                @click="openGroup($event, g)"
              /><IconButton
                action="delete"
                :label="t('managed.deleteNamed', { name: g.name })"
                @click="deleteGroup($event, g)"
              />
            </div></div
        ></Card>
        <RouterLink
          class="inline-block text-sm text-primary underline"
          to="/dashboard/route/policies"
          >{{ t('lab.trafficPolicies') }} →</RouterLink
        >
      </template>
      <template v-else>
        <ManagedPolicyList :snapshot="snapshot" :editing-id="policy?.id" :busy="busy"
          :feedback="feedback" :feedback-policy-id="feedbackPolicyId"
          @edit="openPolicy" @review="policyAction">
          <template #create>
            <ManagedPolicyEditor v-if="policy && !policy.id" ref="editor" v-model="policy" v-model:conditions="conditions" v-model:dns-choice="dnsChoice"
            v-model:dns-search="dnsSearch" :snapshot="snapshot" :busy="busy" :error="error"
            :plan="panelOpen ? null : plan" @preview="previewPolicy" @cancel="cancelPolicy"
            @apply="commit" @back="clearPlan" @create-group="createGroup" />
          </template>
          <template #editor>
            <ManagedPolicyEditor v-if="policy?.id" ref="editor" v-model="policy" v-model:conditions="conditions" v-model:dns-choice="dnsChoice"
            v-model:dns-search="dnsSearch" :snapshot="snapshot" :busy="busy" :error="error"
            :plan="panelOpen ? null : plan" @preview="previewPolicy" @cancel="cancelPolicy"
            @apply="commit" @back="clearPlan" @create-group="createGroup" />
          </template>
        </ManagedPolicyList>
        <RouterLink
          class="inline-block text-sm text-primary underline"
          to="/dashboard/dns/groups"
          >{{ t('lab.resolverGroups') }} →</RouterLink
        >
      </template>
      <ManagedPlanPreview v-if="plan && !panelOpen && !policy" :plan="plan" :busy="busy"
        @apply="commit" @back="clearPlan" />
    </template>
    <Popover
      ref="panel"
      panel-class="w-[min(44rem,calc(100vw-2rem))]"
      @hide="panelClosed"
      ><div class="w-[min(42rem,calc(100vw-4rem))] space-y-4">
        <DNSGroupForm
          v-if="panelKind === 'group' && groupDraft && !plan"
          :key="groupDraft.id + groupDraft.name"
          :initial="groupDraft"
          :groups="snapshot?.groups ?? []"
          :servers="snapshot?.servers ?? []"
          :mode="groupDraft.id ? 'edit' : 'create'"
          live
          :busy="busy"
          @cancel="cancelGroup"
          @save="previewGroup"
        />
        <template v-if="panelKind === 'deleteGroup' && !plan"
          ><h2 class="font-semibold">
            {{ t('managed.deleteNamed', { name: targetGroup?.name }) }}
          </h2>
          <p class="text-sm">{{ t('managed.deleteExplanation') }}</p>
          <p>{{ t('managed.usedBy', { n: dependencies.length }) }}</p>
          <ul class="list-inside list-disc text-sm">
            <li v-for="p in dependencies" :key="p.id">
              {{ p.name }} · {{ t('managed.' + p.status) }}
            </li>
          </ul>
          <Select
            v-if="dependencies.length"
            ref="replacementSelect"
 v-model="replacement"
 @filter="replacementSearch=$event.value"
            :options="snapshot?.groups.filter((g) => g.id !== targetGroup?.id)"
            option-label="name"
            option-value="id"
            filter
            :aria-label="t('managed.replacement')"
            :placeholder="t('managed.replacement')"
            class="w-full"
            @change="clearPlan"
          ><template #footer><button type="button" class="w-full border-t border-border p-3 text-left text-primary" @click="createReplacement">+ {{ t('lab.createDnsGroup') }}</button></template></Select>
          <button v-if="dependencies.length" type="button" class="block text-sm text-primary underline" @click="createReplacement">{{ t('lab.createDnsGroup') }}</button>
          <Button
            :disabled="busy || (dependencies.length > 0 && !replacement)"
            @click="
              preview({
                action: 'delete_group',
                id: targetGroup?.id,
                replacement,
              })
            "
            >{{ t('managed.preview') }}</Button
          >
          <Button
            variant="ghost"
            :disabled="busy"
            @click="preview({ action: 'detach_group', id: targetGroup?.id })"
            >{{ t('managed.detachGroup') }}</Button
          >
        </template>
        <template v-if="panelKind === 'policyAction' && targetPolicy && !plan"
          ><h2 class="font-semibold">{{ targetPolicy.name }}</h2>
          <p class="text-sm">{{ t('managed.driftHelp') }}</p>
          <details class="text-xs">
            <summary>{{ t('managed.savedRules') }}</summary>
            <pre class="max-h-64 overflow-auto">{{
              serialized({
                route: targetPolicy.route_members,
                dns: targetPolicy.dns_members,
              })
            }}</pre>
          </details>
          <div class="flex flex-wrap gap-2">
            <Button
              v-if="targetPolicy.status === 'synced'"
              :disabled="busy"
              @click="preview({ action: 'delete_policy', id: targetPolicy.id })"
              >{{ t('managed.removeOwned') }}</Button
            ><Button
              v-if="targetPolicy.status === 'modified'"
              :disabled="busy"
              @click="preview({ action: 'accept_policy', id: targetPolicy.id })"
              >{{ t('managed.accept') }}</Button
            ><Button
              v-if="targetPolicy.status === 'missing'"
              :disabled="busy"
              @click="
                preview({ action: 'restore_policy', id: targetPolicy.id })
              "
              >{{ t('managed.restore') }}</Button
            ><Button
              variant="secondary"
              :disabled="busy"
              @click="preview({ action: 'detach_policy', id: targetPolicy.id })"
              >{{ t('managed.detach') }}</Button
            >
          </div></template
        >
        <p v-if="error" role="alert" class="text-sm text-danger">{{ error }}</p>
        <template v-if="plan"
          ><h2 class="font-semibold">{{ t('managed.proposed') }}</h2>
          <p class="text-sm">
            {{ t('managed.affected') }}:
            {{ plan.affected.join(', ') || t('managed.metadataOnly') }}
          </p>
          <p class="text-xs text-text-secondary">
            {{ t('managed.validation') }}
          </p>
          <pre
            v-if="plan.command.group"
            class="max-h-40 overflow-auto text-xs"
            >{{ serialized(plan.command.group) }}</pre
          >
          <details open>
            <summary class="text-sm">{{ t('managed.after') }}</summary>
            <pre class="max-h-64 overflow-auto text-xs">{{
              serialized(plan.after)
            }}</pre>
          </details>
          <details>
            <summary class="text-sm">{{ t('managed.before') }}</summary>
            <pre class="max-h-64 overflow-auto text-xs">{{
              serialized(plan.before)
            }}</pre>
          </details>
          <div class="flex justify-end gap-2">
            <Button variant="ghost" :disabled="busy" @click="plan = null">{{
              t('managed.back')
            }}</Button
            ><Button :disabled="busy" @click="commit">{{
              t('managed.apply')
            }}</Button>
          </div></template
        >
      </div></Popover
    >
  </section>
</template>
