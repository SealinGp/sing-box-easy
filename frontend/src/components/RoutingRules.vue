<script setup lang="ts">
import ModulePresets from './ModulePresets.vue'
import { ArrowsUpDownIcon } from '@heroicons/vue/24/outline'
import Card from './Card.vue'
import { ContextPanel, Select } from '../volt'
import Button from './Button.vue'
import RoutingRuleItem from './RoutingRuleItem.vue'
import List from './List.vue'
import RouteRuleMatchers from './RouteRuleMatchers.vue'
import SchemaFieldsEditor from './SchemaFieldsEditor.vue'
import RuleFlowPreview from './RuleFlowPreview.vue'
import FieldHelp from './FieldHelp.vue'
import { ALL_MATCHER_KEYS } from '../schemas/routeRuleMatcherFields'
import { useRoutingRules } from '../composables/useRoutingRules'

const { loading, submitting, formError, rules, coreCapabilities, editingRule, matchersKey,
  activeRule, currentAction, actionOptions, actionIsDefaulted, actionFields, actionRecord,
  flowOutcome, flowContinues, dialogVisible, reorder, openAddDialog, startEditRule,
  submitDeleteRule, submitUpdateRule, submitAddRule, changeAction, fetchRouteRules } = useRoutingRules()
</script>

<template>
  <div class="space-y-4 pt-2">
    <Card>
      <div class="flex justify-between items-center mb-4">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">
          {{ $t('route.rules.title') }}
        </h3>
        <div class="flex flex-wrap items-center gap-2">
          <ModulePresets kind="routes" @applied="fetchRouteRules" />
          <!-- Reorder is a MODE, and a BATCHED one: turning it on swaps every
               row's edit/delete cluster for a drag handle, and nothing is
               written until Save — a five-row cleanup is one config write, not
               five. -->
          <template v-if="reorder.enabled.value">
            <button
              @click="reorder.cancel()"
              :disabled="reorder.saving.value"
              class="px-3 py-1.5 text-sm font-medium rounded-control border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors disabled:opacity-50"
            >
              {{ $t('route.rules.reorder.cancel') }}
            </button>
            <button
              @click="reorder.save()"
              :disabled="reorder.saving.value"
              class="px-3 py-1.5 text-sm font-medium bg-primary-600 text-white rounded-control hover:bg-primary-700 transition-colors inline-flex items-center gap-1.5 disabled:opacity-50"
            >
              <ArrowsUpDownIcon class="w-4 h-4" />
              {{ reorder.dirty.value ? $t('route.rules.reorder.save') : $t('route.rules.reorder.done') }}
            </button>
          </template>

          <button
            v-else
            @click="reorder.start()"
            :disabled="rules.length < 2"
            class="px-3 py-1.5 text-sm font-medium rounded-control border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors inline-flex items-center gap-1.5 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <ArrowsUpDownIcon class="w-4 h-4" />
            {{ $t('route.rules.reorder.start') }}
          </button>

          <button
            @click="openAddDialog"
            class="px-3 py-1.5 text-sm font-medium bg-primary-600 text-white rounded-control hover:bg-primary-700 transition-colors"
          >
            {{ $t('route.rules.add') }}
          </button>
        </div>
      </div>

      <p v-if="reorder.enabled.value" class="mb-3 text-xs text-gray-500 dark:text-gray-400">
        {{ $t('route.rules.reorder.hint') }}
        <span v-if="reorder.dirty.value" class="text-amber-600 dark:text-amber-400 font-medium">
          {{ $t('route.rules.reorder.dirtyHint') }}
        </span>
      </p>

      <div class="mb-3 flex items-center gap-1 text-xs text-text-secondary">
        <span>{{ $t('route.rules.priority.title') }}</span>
        <FieldHelp :label="$t('route.rules.priority.title')" :text="$t('route.rules.priority.hint')" />
      </div>

      <List :loading="loading" :empty="rules.length === 0" transition>
        <template #empty>{{ $t('route.rules.empty') }}</template>

        <!-- Keyed by rule identity, not by index — index keys would make every
             reorder a text patch, which cannot animate. `rowAttrs` carries the
             drag handlers; it is empty off reorder mode. -->
        <RoutingRuleItem
          v-for="(rule, index) in rules"
          :key="reorder.keyAt(index)"
          :rule="rule"
          :index="index"
          :reorderable="reorder.enabled.value"
          :row-attrs="reorder.rowAttrs(index)"
          :handle-attrs="reorder.handleAttrs(index)"
          @edit="startEditRule"
          @delete="submitDeleteRule"
        />
      </List>
    </Card>

    <!-- Add/Edit Rule Dialog -->
    <ContextPanel
      v-model:visible="dialogVisible"
      modal
      :closable="!submitting"
      :dismissable="!submitting"
      :header="editingRule ? $t('route.rules.modal.edit') : $t('route.rules.modal.add')"
      class="w-full max-w-2xl"
    >
      <div :key="editingRule ? `edit-${editingRule.index}` : 'add'" class="space-y-4" :inert="submitting" :aria-busy="submitting">
        <!--
          The rule restated as sing-box will run it. The form below shows which
          fields exist; this says what they DO. Kept at the top so it reads as a
          heading the editing confirms, and it updates live.
        -->
        <RuleFlowPreview
          :rule="(activeRule as Record<string, unknown>)"
          :condition-keys="ALL_MATCHER_KEYS"
          label-prefix="route.rules.fields"
          :outcome="flowOutcome"
          :continues-matching="flowContinues"
          :catch-all="!flowContinues"
        />

        <!--
          WHEN / THEN, in that order.

          The dialog used to lead with the action and put the conditions
          underneath, which is backwards from how the rule executes: sing-box
          tests the matchers first and only then applies the action. Reading the
          form top-to-bottom now follows the connection through it.
        -->
        <section class="space-y-3">
          <div class="flex items-baseline gap-2">
            <span
              class="inline-flex items-center justify-center h-5 min-w-5 px-1.5 rounded-pill bg-primary-100 dark:bg-primary-900/40 text-[11px] font-semibold text-primary-700 dark:text-primary-300"
              >1</span
            >
            <h4 class="text-sm font-semibold text-gray-900 dark:text-gray-100">
              {{ $t('route.rules.flow.whenHeading') }}
            </h4>
            <span class="text-xs text-gray-500 dark:text-gray-400">
              {{ $t('route.rules.flow.whenHint') }}
            </span>
          </div>

          <!-- Remounted per dialog open (:key) so the collapse state is decided
               from the rule as loaded, not from the previous edit. -->
          <!-- The key also changes when /core answers, so a dialog opened
               before the capabilities arrived re-derives its gated fields. -->
          <RouteRuleMatchers
            :key="`${matchersKey}-${coreCapabilities ? 'core' : 'nocore'}`"
            v-model="activeRule"
            :capabilities="coreCapabilities"
          />
        </section>

        <section class="space-y-3 border-t border-gray-200 dark:border-gray-700 pt-4">
          <div class="flex items-baseline gap-2">
            <span
              class="inline-flex items-center justify-center h-5 min-w-5 px-1.5 rounded-pill bg-primary-100 dark:bg-primary-900/40 text-[11px] font-semibold text-primary-700 dark:text-primary-300"
              >2</span
            >
            <h4 class="text-sm font-semibold text-gray-900 dark:text-gray-100">
              {{ $t('route.rules.flow.thenHeading') }}
            </h4>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('route.rules.fields.action') }}</label>
            <Select
              :modelValue="currentAction"
              @update:modelValue="changeAction"
              :options="actionOptions"
              optionLabel="label"
              optionValue="value"
              filter
              :filterPlaceholder="$t('common.search')"
              :emptyFilterMessage="$t('common.noMatch')"
              :placeholder="$t('route.rules.placeholders.action')"
              class="w-full"
            />
            <p v-if="actionIsDefaulted" class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ $t('route.rules.actionDefaultHint') }}
            </p>
          </div>

          <!--
            Everything the action owns, from the generated inventory. Replaces six
            hand-written v-if blocks that covered 9 of the 45 fields across the
            seven actions, offered no `direct` action at all, and left
            `route-options` unable to reach seven of its ten.

            :key remounts on action change so the previous action's added-field
            state does not leak — same reason matchersKey exists above.
          -->
          <SchemaFieldsEditor
            :key="currentAction"
            v-model="actionRecord"
            :fields="actionFields"
            :empty-hint="$t('route.rules.hints.hijackDns')"
          />
        </section>
      </div>

      <template #footer>
        <p v-if="formError" role="alert" class="w-full rounded-control border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900 dark:bg-red-950/40 dark:text-red-300">{{ formError }}</p>
        <Button
          :label="$t('common.cancel')"
          severity="secondary"
          :disabled="submitting"
          @click="dialogVisible = false"
        />
        <Button
          :label="editingRule ? $t('common.update') : $t('common.add')"
          :loading="submitting"
          :disabled="submitting"
          @click="editingRule ? submitUpdateRule() : submitAddRule()"
        />
      </template>
    </ContextPanel>
  </div>
</template>
