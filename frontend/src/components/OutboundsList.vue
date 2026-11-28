<script setup lang="ts">
import IconButton from './IconButton.vue'
import SubscriptionSource from './SubscriptionSource.vue'
import { nodeNameWithoutSubscription } from '../utils/nodeSubscription'
import Button from './Button.vue'
import Input from './Input.vue'
import Badge from './Badge.vue'
import Alert from './Alert.vue'
import Table from './Table.vue'
import Textarea from './Textarea.vue'
import SchemaFieldsEditor from './SchemaFieldsEditor.vue'
import { Select, ContextPanel } from '../volt'
import { PlusIcon, ArrowDownTrayIcon } from '@heroicons/vue/24/outline'
import { resolveOutboundFields } from '../schemas/outboundFields'
import { FOCUS_PAGES, focusLink } from '../utils/focusTarget'
import { useOutboundsPage } from '../composables/useOutboundsPage'
import { useOutboundImport } from '../composables/useOutboundImport'
const {
  localLoading,
  showModal,
  isEditMode,
  editingTag,
  currentOutbound,
  currentType,
  managedWarning,
  currentTypeWarning,
  showDeleteConfirm,
  deletingOutbound,
  deletingOutboundIndex,
  selectedOutbounds,
  toggleOutboundSelection,
  toggleSelectAllOutbounds,
  handleBatchDelete,
  outboundTypes,
  getOutboundTypeLabel,
  getOutboundBadgeVariant,
  openAddModal,
  openEditModal,
  closeModal,
  handleSave,
  closeDeleteConfirm,
  handleDelete,
  changeType,
  outbounds,
  loading
} = useOutboundsPage()
const { showImportModal, importInput, parsing, parsedNodes, selectedNodes, importing, openImportModal, closeImportModal, parseSubscription, toggleNode, toggleSelectAll, handleImport, resetImportFlow } = useOutboundImport()
</script>

<template>
  <div>
    <div class="mb-4 grid gap-4 sm:flex sm:flex-wrap sm:items-end sm:justify-between">
      <div>
        <h1 class="text-2xl font-semibold text-text-primary">{{ $t('outbounds.tabs.list') }}</h1>
        <p class="mt-1 text-xs text-text-secondary">{{ $t('outbounds.count', outbounds.length) }}</p>
      </div>
      <div class="flex min-h-9 min-w-0 flex-wrap items-center justify-end gap-2">
        <span v-if="selectedOutbounds.size > 0" class="text-xs text-text-secondary">{{ $t('outbounds.importModal.selected', { count: selectedOutbounds.size }) }}</span>
        <IconButton action="delete" v-if="selectedOutbounds.size > 0" :label="$t('outbounds.deleteSelected', { count: selectedOutbounds.size })" @click="handleBatchDelete" />
        <Button v-if="selectedOutbounds.size > 0" variant="ghost" class="sm:hidden" @click="selectedOutbounds.clear()">
          {{ $t('common.cancel') }}
        </Button>
        <div :class="selectedOutbounds.size > 0 ? 'hidden sm:flex' : 'flex'" class="items-center gap-2">
        <Button @click="openImportModal" variant="secondary">
          <ArrowDownTrayIcon class="h-5 w-5" />
          {{ $t('outbounds.import') }}
        </Button>
        <Button @click="openAddModal" variant="primary">
          <PlusIcon class="h-5 w-5" />
          {{ $t('outbounds.add') }}
        </Button>
        </div>
      </div>
    </div>

    <div
      class="table-surface"
    >
      <!--
        The select-all checkbox means this header needs markup, not just
        labels, so it uses #head rather than the `columns` prop.
      -->
      <Table table-class="table-fixed min-w-5xl" :loading="loading && outbounds.length === 0" :empty="outbounds.length === 0">
        <template #empty>
          <p class="mb-3">{{ $t('outbounds.empty') }}</p>
          <Button @click="openAddModal" variant="primary" size="sm">
            <PlusIcon class="h-4 w-4 mr-1.5" />
            {{ $t('outbounds.addFirst') }}
          </Button>
        </template>

        <template #head>
          <th class="col-select">
            <label class="table-checkbox">
            <input
              type="checkbox"
              :checked="selectedOutbounds.size === outbounds.length && outbounds.length > 0"
              :indeterminate="selectedOutbounds.size > 0 && selectedOutbounds.size < outbounds.length"
              :aria-label="$t('outbounds.selectAll')"
              class="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500"
              @change="toggleSelectAllOutbounds"
            />
            </label>
          </th>
          <th class="w-16">#</th>
          <th>{{ $t('outbounds.table.tag') }}</th>
          <th class="w-48">{{ $t('outbounds.table.subscription') }}</th>
          <th class="w-36">{{ $t('outbounds.table.type') }}</th>
          <th class="w-52">{{ $t('outbounds.table.server') }}</th>
          <th class="w-20">{{ $t('outbounds.table.port') }}</th>
          <th class="col-actions w-28">{{ $t('outbounds.table.actions') }}</th>
        </template>

        <tr v-for="(outbound, i) in outbounds" :key="outbound.tag" :data-selected="selectedOutbounds.has(outbound.tag)">
          <td class="col-select">
            <label class="table-checkbox">
            <input
              type="checkbox"
              :checked="selectedOutbounds.has(outbound.tag)"
              :aria-label="$t('outbounds.selectOutbound', {tag: outbound.tag})"
              class="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500"
              @change="toggleOutboundSelection(outbound.tag)"
            />
            </label>
          </td>
          <td class="font-medium text-gray-900 dark:text-gray-100">{{ i + 1 }}</td>
          <td class="font-medium text-gray-900 dark:text-gray-100"><span class="block truncate" :title="outbound.tag">{{ nodeNameWithoutSubscription(outbound.tag) || i }}</span></td>
          <td data-cell-tooltip="false"><SubscriptionSource :name="outbound.tag" /></td>
          <td>
            <Badge :variant="getOutboundBadgeVariant(outbound.type)">
              {{ getOutboundTypeLabel(outbound.type) }}
            </Badge>
          </td>
          <td class="text-gray-900 dark:text-gray-100"><span class="block truncate" :title="(outbound as any).server">{{ (outbound as any).server || "-" }}</span></td>
          <td class="text-gray-900 dark:text-gray-100">{{ (outbound as any).server_port || "-" }}</td>
          <td class="col-actions font-medium">
            <div class="flex items-center justify-end gap-1">
              <IconButton action="edit" @click="openEditModal(outbound)" :label="`${$t('common.edit')} ${outbound.tag}`" />
              <IconButton
                action="delete"
                @click="deletingOutbound = outbound; deletingOutboundIndex = i; showDeleteConfirm = true"
                :label="`${$t('common.delete')} ${outbound.tag}`"
              />
            </div>
          </td>
        </tr>
      </Table>
    </div>

    <!-- Add/Edit Modal -->
    <ContextPanel
      :visible="showModal"
      :closable="!(localLoading)" :dismissable="!(localLoading)" :close-on-escape="!(localLoading)"
      @update:visible="(v: boolean) => { if (!v && !localLoading) closeModal() }"
      :header="isEditMode ? $t('outbounds.modal.edit') : $t('outbounds.modal.add')"
      modal
      class="w-full max-w-3xl"
    >
      <fieldset :disabled="localLoading" :inert="localLoading" class="space-y-3 min-w-0">
        <Alert v-if="managedWarning" type="warning" :title="$t('outbounds.form.managedByRulesTitle')">
          {{ managedWarning }}
          <!-- The tag of a managed outbound is its Filter's or Group's name,
               so the link can land on the exact card to edit. -->
          <RouterLink
            :to="focusLink(FOCUS_PAGES.nodeRules, editingTag)"
            class="mt-1 block font-medium underline underline-offset-2 hover:no-underline"
          >
            {{ $t('outbounds.form.openNodeRule', { name: editingTag }) }}
          </RouterLink>
        </Alert>

        <!--
          Tag and type live on the outbound wrapper rather than in any type's
          option struct, so they are rendered here rather than coming from the
          schema. Both lock on edit: the tag is what groups and route rules
          reference, and changing the type would reinterpret every field.
        -->
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('outbounds.form.tag') }}</label>
            <Input
              v-model="(currentOutbound.tag as string)"
              :placeholder="$t('outbounds.form.tagPlaceholder')"
              :disabled="isEditMode"
            />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('outbounds.form.type') }}</label>
            <Select
              class="w-full"
              :modelValue="currentOutbound.type"
              :options="outboundTypes"
              optionLabel="label"
              optionValue="value"
              filter
              :filterPlaceholder="$t('common.search')"
              :emptyFilterMessage="$t('common.noMatch')"
              :placeholder="$t('outbounds.form.typePlaceholder')"
              :disabled="isEditMode"
              @update:modelValue="changeType"
            />
            <p v-if="currentTypeWarning" class="mt-1 text-xs text-amber-600 dark:text-amber-400">
              {{ currentTypeWarning }}
            </p>
          </div>
        </div>

        <!--
          :key remounts the editor on every type change, dropping the set of
          fields the operator had added. The old watcher wiped the whole model
          instead — including dialer options already filled in — so switching
          type cost work rather than just clearing what no longer applied.
        -->
        <SchemaFieldsEditor
          v-if="currentType"
          :key="currentType"
          v-model="currentOutbound"
          :fields="resolveOutboundFields(currentType)"
          :empty-hint="$t('outbounds.form.noOptionsHint')"
        />
      </fieldset>

      <template #footer>
        <Button @click="closeModal" :disabled="localLoading" variant="secondary"
          >{{ $t('common.cancel') }}</Button
        >
        <Button
          @click="handleSave"
          variant="primary"
          :disabled="localLoading"
          :loading="localLoading"
        >
          {{ isEditMode ? $t('common.update') : $t('common.add') }}
        </Button>
      </template>
    </ContextPanel>

    <!-- Import Modal -->
    <ContextPanel
      :visible="showImportModal"
      :closable="!(parsing || importing)" :dismissable="!(parsing || importing)" :close-on-escape="!(parsing || importing)"
      @update:visible="(v: boolean) => { if (!v && !parsing && !importing) closeImportModal() }"
      :header="$t('outbounds.importModal.title')"
      modal
      class="w-full max-w-2xl"
    >
      <div class="space-y-4">
        <!-- Input Section -->
        <div v-if="parsedNodes.length === 0">
          <p class="text-sm text-gray-600 dark:text-gray-400 mb-3">
            {{ $t('outbounds.importModal.instructions') }}
          </p>
          <Textarea
            v-model="importInput"
            :placeholder="$t('outbounds.importModal.inputPlaceholder')"
            :disabled="parsing"
            :rows="6"
            full-width
          />
          <div class="flex justify-end mt-3">
            <Button
              variant="primary"
              :loading="parsing"
              :disabled="parsing"
              @click="parseSubscription"
            >
              {{ parsing ? $t('outbounds.importModal.parsing') : $t('outbounds.importModal.parse') }}
            </Button>
          </div>
        </div>

        <!-- Parsed Nodes List -->
        <div v-else>
          <div class="flex items-center justify-between mb-3">
            <div>
              <h4
                class="text-sm font-semibold text-gray-900 dark:text-gray-100"
              >
                {{ $t('outbounds.importModal.parsedNodes') }}
                <Badge variant="primary" class="ml-2" size="sm">{{
                  parsedNodes.length
                }}</Badge>
              </h4>
              <p
                class="text-xs text-gray-600 dark:text-gray-400 mt-1"
              >
                {{ $t('outbounds.importModal.selected', { count: selectedNodes.size }) }}
              </p>
            </div>
            <Button
              variant="ghost"
              size="sm"
              @click="toggleSelectAll"
            >
              {{
                selectedNodes.size === parsedNodes.length
                  ? $t('outbounds.importModal.deselectAll')
                  : $t('outbounds.importModal.selectAll')
              }}
            </Button>
          </div>

          <!-- Nodes List -->
          <div
            class="max-h-80 overflow-y-auto border border-gray-200 dark:border-gray-700 rounded-surface"
          >
            <div
              v-for="node in parsedNodes"
              :key="node.tag"
              class="flex items-center gap-3 p-3 hover:bg-gray-50 dark:hover:bg-slate-700 border-b border-gray-100 dark:border-gray-800 last:border-0 cursor-pointer"
              @click="toggleNode(node.tag)"
            >
              <input
                type="checkbox"
                :checked="selectedNodes.has(node.tag)"
                class="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500"
                @click.stop="toggleNode(node.tag)"
              />
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2">
                  <p
                    class="text-sm font-medium text-gray-900 dark:text-gray-100 truncate"
                  >
                    {{ node.tag }}
                  </p>
                  <Badge variant="info" size="sm">{{
                    node.type
                  }}</Badge>
                </div>
                <p
                  class="text-xs text-gray-500 dark:text-gray-500 truncate"
                >
                  {{ (node as any).server || "-" }}:{{
                    (node as any).server_port || "-"
                  }}
                </p>
              </div>
            </div>
          </div>

          <div
            class="flex justify-between items-center mt-4 pt-4 border-t border-gray-200 dark:border-gray-700"
          >
            <Button variant="ghost" :disabled="importing" @click="resetImportFlow">
              {{ $t('outbounds.importModal.backToInput') }}
            </Button>
            <div class="flex gap-3">
              <Button variant="secondary" :disabled="parsing || importing" @click="closeImportModal">
                {{ $t('common.cancel') }}
              </Button>
              <Button
                variant="primary"
                :loading="importing"
                :disabled="importing || selectedNodes.size === 0"
                @click="handleImport"
              >
                {{
                  importing
                    ? $t('outbounds.importModal.importing')
                    : $t('outbounds.importModal.importNodes', { count: selectedNodes.size })
                }}
              </Button>
            </div>
          </div>
        </div>
      </div>
    </ContextPanel>

    <!-- Delete Confirmation Modal -->
    <ContextPanel
      :visible="showDeleteConfirm"
      :closable="!(localLoading)" :dismissable="!(localLoading)" :close-on-escape="!(localLoading)"
      @update:visible="(v: boolean) => { if (!v && !localLoading) closeDeleteConfirm() }"
      :header="$t('outbounds.del.title')"
      modal
      class="w-full max-w-md"
    >
      <p class="text-gray-700 dark:text-gray-300">
        {{ $t('outbounds.del.confirm', { tag: deletingOutbound?.tag }) }}
      </p>

      <template #footer>
        <Button @click="closeDeleteConfirm" :disabled="localLoading" variant="secondary">{{ $t('common.cancel') }}</Button>
        <IconButton action="delete" @click="handleDelete" :disabled="localLoading" />
      </template>
    </ContextPanel>
  </div>
</template>
