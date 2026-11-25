<script setup lang="ts">
import IconButton from './IconButton.vue'
import Button from './Button.vue'
import Card from './Card.vue'
import List from './List.vue'
import DNSRuleItem from './DNSRuleItem.vue'
import { ContextPanel, Select } from '../volt'
import DNSRuleConditions from './DNSRuleConditions.vue'
import SchemaFieldsEditor from './SchemaFieldsEditor.vue'
import RuleFlowPreview from './RuleFlowPreview.vue'
import DNSParallelGroupDialog from './DNSParallelGroupDialog.vue'
import { PlusIcon, ArrowsUpDownIcon, BoltIcon } from '@heroicons/vue/24/outline'
import { useDNSRules } from '../composables/useDNSRules'

const { loading, dnsRules, showRuleModal, conditionsKey, isEditMode, currentRule, currentAction,
 actionFields, conditionKeys, flowOutcome, flowContinues, changeAction, actionTypes, ruleSetOptions,
 openAddRuleModal, openEditRuleModal, closeRuleModal, handleSaveRule, showDeleteConfirm,
 deletingIndex, openDeleteConfirm, closeDeleteConfirm, handleDeleteRule, showParallelModal,
 canBuildParallelGroup, handleCreateParallelGroup, orphanRespondIndices, reorder } = useDNSRules()
</script>

<template>
  <div class="space-y-4 pt-2">
    <Card>
    <header class="section-toolbar">
      <h1>{{ $t('dns.rules.heading') }}</h1>
      <div class="section-toolbar-actions">
      <!-- Reorder is a MODE, and a BATCHED one: turning it on swaps the index
           column for a drag handle and hides edit/delete, and nothing is
           written until Save — a five-row cleanup is one config write, not
           five. -->
      <template v-if="reorder.enabled.value">
        <Button @click="reorder.cancel()" variant="secondary" :disabled="reorder.saving.value">
          {{ $t('dns.rules.reorder.cancel') }}
        </Button>
        <Button @click="reorder.save()" variant="primary" :disabled="reorder.saving.value">
          <ArrowsUpDownIcon class="h-4 w-4" />
          {{ reorder.dirty.value ? $t('dns.rules.reorder.save') : $t('dns.rules.reorder.done') }}
        </Button>
      </template>

      <Button
        v-else-if="dnsRules.length > 1"
        @click="reorder.start()"
        variant="secondary"
      >
        <ArrowsUpDownIcon class="h-4 w-4" />
        {{ $t('dns.rules.reorder.start') }}
      </Button>

      <!-- Only with a core that has evaluate/respond: on an older one this
           would build a config that fails `sing-box check`. -->
      <Button
        v-if="canBuildParallelGroup && !reorder.enabled.value"
        @click="showParallelModal = true"
        variant="secondary"
      >
        <BoltIcon class="h-4 w-4" />
        {{ $t('dns.rules.parallel.add') }}
      </Button>

      <Button @click="openAddRuleModal" variant="primary">
        <PlusIcon class="h-4 w-4" />
        {{ $t('dns.rules.add') }}
      </Button>
      </div>
    </header>

    <p v-if="reorder.enabled.value" class="mb-2 text-xs text-gray-500 dark:text-gray-400">
      {{ $t('dns.rules.reorder.hint') }}
      <span v-if="reorder.dirty.value" class="text-amber-600 dark:text-amber-400 font-medium">
        {{ $t('dns.rules.reorder.dirtyHint') }}
      </span>
    </p>

    <List :loading="loading && !dnsRules.length" :empty="!dnsRules.length" transition>
      <template #empty>
        <p class="mb-3">{{ $t('dns.rules.empty') }}</p>
        <Button @click="openAddRuleModal">{{ $t('dns.rules.addFirst') }}</Button>
      </template>
      <DNSRuleItem v-for="(rule, index) in dnsRules" :key="reorder.keyAt(index)"
        :rule="rule" :index="index" :reorderable="reorder.enabled.value"
        :row-attrs="reorder.rowAttrs(index)" :handle-attrs="reorder.handleAttrs(index)"
        :orphan="orphanRespondIndices.has(index)"
        @edit="openEditRuleModal(index, rule)" @delete="openDeleteConfirm(index)" />
    </List>

    </Card>

    <!-- Add/Edit DNS Rule Modal -->
    <ContextPanel
      :visible="showRuleModal"
      @update:visible="(v: boolean) => { if (!v) closeRuleModal() }"
      :header="isEditMode ? $t('dns.rules.modal.edit') : $t('dns.rules.modal.add')"
      modal
      class="w-full max-w-2xl"
    >
      <div class="space-y-4">
        <!--
          The rule restated as sing-box will run it. The form shows which fields
          exist; this says what they DO. Same component the route rule dialog
          uses — only the outcome phrase and the condition list differ.
        -->
        <RuleFlowPreview
          :rule="currentRule"
          :condition-keys="conditionKeys"
          label-prefix="dns.rules.form.fields"
          :outcome="flowOutcome"
          :continues-matching="flowContinues"
          :catch-all="!flowContinues"
        />

        <!--
          WHEN / THEN, in that order.

          The dialog used to lead with the action. sing-box tests the conditions
          first and only then applies the action, so reading top-to-bottom now
          follows the query through the rule.
        -->
        <section class="space-y-3">
          <div class="flex items-baseline gap-2">
            <span
              class="inline-flex items-center justify-center h-5 min-w-5 px-1.5 rounded-pill bg-primary-100 dark:bg-primary-900/40 text-[11px] font-semibold text-primary-700 dark:text-primary-300"
              >1</span
            >
            <h4 class="text-sm font-semibold text-gray-900 dark:text-gray-100">
              {{ $t('dns.rules.form.whenHeading') }}
            </h4>
            <span class="text-xs text-gray-500 dark:text-gray-400">
              {{ $t('dns.rules.form.whenHint') }}
            </span>
          </div>

          <!-- Remounted per modal open (:key) so the collapse state is decided
               from the rule as loaded, not from the previous edit. -->
          <DNSRuleConditions
            :key="conditionsKey"
            v-model:rule-set="currentRule.rule_set"
            v-model:domain="currentRule.domain"
            v-model:domain-suffix="currentRule.domain_suffix"
            v-model:domain-keyword="currentRule.domain_keyword"
            v-model:geosite="currentRule.geosite"
            v-model:query-type="currentRule.query_type"
            :rule-set-options="ruleSetOptions"
          />
        </section>

        <section class="space-y-3 border-t border-gray-200 dark:border-gray-700 pt-4">
          <div class="flex items-baseline gap-2">
            <span
              class="inline-flex items-center justify-center h-5 min-w-5 px-1.5 rounded-pill bg-primary-100 dark:bg-primary-900/40 text-[11px] font-semibold text-primary-700 dark:text-primary-300"
              >2</span
            >
            <h4 class="text-sm font-semibold text-gray-900 dark:text-gray-100">
              {{ $t('dns.rules.form.thenHeading') }}
            </h4>
          </div>

          <!--
            The action select stays outside the schema editor, exactly as the
            type select does in DNSServers.vue: it is the discriminator that
            CHOOSES the field set, so it cannot be one of the fields.
          -->
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('dns.rules.form.action') }}</label>
            <Select
              class="w-full"
              optionLabel="label"
              optionValue="value"
              :modelValue="currentAction"
              :options="actionTypes"
              @update:modelValue="changeAction"
            />
          </div>

          <!--
            Everything the action owns, from the generated inventory. Replaces
            three hand-written v-if blocks that covered 3 of 15 fields and left
            `route-options` with an empty form body.

            :key remounts on action change so the previous action's added-field
            state does not leak — same reason conditionsKey exists above.
          -->
          <SchemaFieldsEditor
            :key="currentAction"
            v-model="currentRule"
            :fields="actionFields"
          />
        </section>
      </div>

      <template #footer>
        <Button @click="closeRuleModal" variant="secondary">{{ $t('common.cancel') }}</Button>
        <Button @click="handleSaveRule" variant="primary" :disabled="loading">
          {{ isEditMode ? $t('common.update') : $t('common.add') }}
        </Button>
      </template>
    </ContextPanel>

    <DNSParallelGroupDialog
      v-model:visible="showParallelModal"
      :existing-rules="dnsRules as any"
      :rule-set-options="ruleSetOptions"
      :saving="loading"
      @create="handleCreateParallelGroup"
    />

    <!-- Delete Confirmation Modal -->
    <ContextPanel
      :visible="showDeleteConfirm"
      @update:visible="(v: boolean) => { if (!v) closeDeleteConfirm() }"
      :header="$t('dns.rules.del.title')"
      modal
      class="w-full max-w-md"
    >
      <p class="text-gray-700 dark:text-gray-300">
        {{ $t('dns.rules.del.confirm', { index: deletingIndex + 1 }) }}
      </p>

      <template #footer>
        <Button @click="closeDeleteConfirm" variant="secondary">{{ $t('common.cancel') }}</Button>
        <IconButton action="delete" @click="handleDeleteRule" :disabled="loading" />
      </template>
    </ContextPanel>
  </div>
</template>
