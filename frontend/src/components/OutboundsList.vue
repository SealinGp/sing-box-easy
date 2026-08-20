<script setup lang="ts">
import { ref, onMounted, watch, computed } from "vue";
import { useI18n } from "vue-i18n";
import type { Outbound } from "../types/api";
import Button from "./Button.vue";
import Input from "./Input.vue";
import Badge from "./Badge.vue";
import Alert from "./Alert.vue";
import Table from "./Table.vue";
import Textarea from "./Textarea.vue";
import SchemaFieldsEditor from "./SchemaFieldsEditor.vue";
import Modal from "./Modal.vue";
import { Select } from "../volt";
import {
  PlusIcon,
  PencilIcon,
  TrashIcon,
  ArrowDownTrayIcon,
} from "@heroicons/vue/24/outline";
import { nodesService } from "../services";
import { useToast } from "primevue/usetoast";
import { useOutboundsStore } from "../stores/outbounds";
import {
  OUTBOUND_GROUP_TYPES,
  OUTBOUND_TYPES_NEEDING_SERVER,
  OUTBOUND_TYPE_NAMES,
  OUTBOUND_TYPE_NOTES,
  applyTypeDefaults,
  pruneForType,
  resolveOutboundFields,
  type OutboundTypeName,
} from "../schemas/outboundFields";
import { isDeprecatedIn, isRetired } from "../schemas/optionSchema";
import { useSingBoxVersion } from "../composables/useSingBoxVersion";
import { storeToRefs } from "pinia";

const outboundsStore = useOutboundsStore();
// Use storeToRefs to maintain reactivity when destructuring
const { outbounds, loading, managedTags } = storeToRefs(outboundsStore);

const localLoading = ref(false);
const toast = useToast();
const { t, te } = useI18n();

// The installed binary, for gating retired types — see useSingBoxVersion.
const { singBoxVersion } = useSingBoxVersion();

// Modal state
const showModal = ref(false);
const isEditMode = ref(false);
const editingTag = ref<string>("");
const DEFAULT_TYPE: OutboundTypeName = "direct";

function blankOutbound(): Record<string, unknown> {
  return applyTypeDefaults({ tag: "" }, DEFAULT_TYPE);
}

const currentOutbound = ref<Record<string, unknown>>(blankOutbound());

/**
 * The form model is an open record because an outbound's fields depend on its
 * type, and the declared union cannot narrow on `type`.
 */
const currentType = computed(() =>
  typeof currentOutbound.value.type === "string"
    ? (currentOutbound.value.type as OutboundTypeName)
    : undefined,
);

/**
 * Switching type prunes rather than wipes.
 *
 * The old watcher reset the model to `{ type, tag }`, discarding every dialer
 * option the operator had already filled in — fields shared by both types that
 * did not need clearing.
 */
function changeType(next: unknown) {
  if (typeof next !== "string") return;
  currentOutbound.value = pruneForType(currentOutbound.value, next as OutboundTypeName);
}

/**
 * Whether the INSTALLED sing-box has retired the selected type.
 *
 * `dns` and `wireguard` both parse and then fail — one at config decode, one at
 * outbound init — so `sing-box check` passing means nothing here.
 */
/**
 * Whether the outbound being edited is owned by the node-rules engine.
 *
 * BuildGroupOutbounds replaces any outbound whose tag matches a Filter or Group
 * name rather than merging into it, so an edit here is discarded on the next
 * apply — and for a Group the rebuilt selector carries only its members, taking
 * url/interval/tolerance with it. Saying so beats letting the work evaporate.
 */
const managedWarning = computed(() => {
  if (!isEditMode.value) return "";
  return managedTags.value.includes(editingTag.value)
    ? t("outbounds.form.managedByRules")
    : "";
});

const currentTypeWarning = computed(() => {
  const type = currentType.value;
  if (!type) return "";
  const note = OUTBOUND_TYPE_NOTES[type];
  if (!note) return "";
  if (isRetired(note, singBoxVersion.value)) {
    return t("schema.field.typeRetired", {
      removed: note.removed,
      version: singBoxVersion.value,
    });
  }
  if (isDeprecatedIn(note, singBoxVersion.value)) {
    return t("schema.field.deprecated", { since: note.since });
  }
  return "";
});

// Delete confirmation. deleteOutbound() prefers the array index when
// supplied, falling back to the tag — the confirm-modal path uses the tag
// and skips the index entirely.
const showDeleteConfirm = ref(false);
const deletingOutbound = ref<Outbound | null>(null);

// Import modal state
const showImportModal = ref(false);
const importInput = ref("");
const parsing = ref(false);
const parsedNodes = ref<Outbound[]>([]);
const selectedNodes = ref<Set<string>>(new Set());
const importing = ref(false);

// Batch selection state
const selectedOutbounds = ref<Set<string>>(new Set());

// Immutable Set updates: replace the ref's value with a new Set rather than
// mutating the existing one. Vue tracks Set mutations, but the project style
// guide requires immutable patterns.
const toggleOutboundSelection = (tag: string) => {
  const next = new Set(selectedOutbounds.value);
  if (next.has(tag)) next.delete(tag); else next.add(tag);
  selectedOutbounds.value = next;
};

const toggleSelectAllOutbounds = () => {
  if (selectedOutbounds.value.size === outbounds.value.length) {
    selectedOutbounds.value = new Set();
  } else {
    selectedOutbounds.value = new Set(outbounds.value.map((o) => o.tag));
  }
};

const handleBatchDelete = async () => {
  if (selectedOutbounds.value.size === 0) return;
  
  localLoading.value = true;
  try {
    const tags = Array.from(selectedOutbounds.value);
    await outboundsStore.deleteOutboundsBatch(tags);
    selectedOutbounds.value = new Set();
    toast.add({
      severity: "success",
      summary: t('common.success'),
      detail: t('outbounds.toast.batchDeletedOk', { count: tags.length }),
      life: 3000,
    });
  } catch (err: any) {
    toast.add({
      severity: "error",
      summary: t('common.error'),
      detail: err.message || t('outbounds.toast.batchDeleteFailed'),
      life: 3000,
    });
  } finally {
    localLoading.value = false;
  }
};

/**
 * Driven by the generated inventory rather than a hardcoded array.
 *
 * The old literal listed 17 types while the backend registry constructs 20 —
 * `anytls`, `dns` and `shadowsocksr` were registered and unofferable, the same
 * drift that hid `anytls` on the inbound side. Labels stay opt-in: a type with
 * no `outbounds.types.*` entry shows its own name.
 */
const outboundTypes = computed(() =>
  OUTBOUND_TYPE_NAMES.map((value) => ({
    value,
    label: te(`outbounds.types.${value}`) ? t(`outbounds.types.${value}`) : value,
  })),
);

const getOutboundTypeLabel = (type: string) => {
  return outboundTypes.value.find((ot) => ot.value === type)?.label || type;
};

const getOutboundBadgeVariant = (
  type: string
): "primary" | "success" | "warning" | "info" | "secondary" | "danger" => {
  if (type === "direct") return "success";
  if (type === "block") return "danger";
  if (type === "selector" || type === "urltest") return "warning";
  return "primary";
};

/*
 * isProxyType / isGroupType / needsServer / needsPassword / needsUUID /
 * needsMethod / needsOutbounds and groupOutboundOptions used to live here.
 *
 * All of them are now answered by the generated inventory: a type needs a
 * server if its option struct embeds ServerOptions, and it is a group if it has
 * an `outbounds` list. The member picker moved into SchemaFieldControl's
 * 'outbound-list' control, which keeps the same self-exclusion and the same
 * flagging of tags that no longer resolve.
 *
 * The predicates were also doing double duty as required-field gates, which was
 * wrong in two places: `hysteria` needs auth_str and both bandwidths and was
 * asked for none of them, and `tuic` needs a uuid AND a password but the
 * branches were independent.
 */

/**
 * Save-time validation. Returns an i18n key, or '' when valid.
 *
 * Only rules sing-box will actually reject, plus the two selector rules it
 * checks at START rather than at parse — `sing-box check` passes a group with
 * no members or a `default` naming a non-member, and then the service will not
 * come up.
 */
function validateOutbound(outbound: Record<string, unknown>): string {
  const tag = outbound.tag;
  if (typeof tag !== "string" || !tag.trim()) return "outbounds.validation.tagRequired";

  const type = outbound.type;
  if (typeof type !== "string" || !type) return "outbounds.validation.typeRequired";

  if (OUTBOUND_TYPES_NEEDING_SERVER.includes(type as OutboundTypeName)) {
    const server = outbound.server;
    if (typeof server !== "string" || !server.trim()) {
      return "outbounds.validation.serverRequired";
    }
  }

  if (OUTBOUND_GROUP_TYPES.includes(type as OutboundTypeName)) {
    const members = outbound.outbounds;
    if (!Array.isArray(members) || members.length === 0) {
      return "outbounds.validation.outboundsRequired";
    }
    // sing-box: "default outbound not found" — at start, not at check.
    const fallback = outbound.default;
    if (typeof fallback === "string" && fallback && !members.includes(fallback)) {
      return "outbounds.validation.defaultNotMember";
    }
  }

  return "";
}

const openAddModal = () => {
  isEditMode.value = false;
  currentOutbound.value = blankOutbound();
  showModal.value = true;
};

const openEditModal = (outbound: Outbound) => {
  isEditMode.value = true;
  editingTag.value = outbound.tag;
  // sing-box accepts scalar OR array for `outbounds` on the wire; coerce to
  // array so the MultiSelect v-model sees the shape it expects and the save path
  // round-trips correctly.
  const raw = outbound as Outbound & { outbounds?: string | string[] };
  currentOutbound.value = {
    ...outbound,
    outbounds:
      raw.outbounds === undefined || raw.outbounds === null
        ? undefined
        : Array.isArray(raw.outbounds)
          ? [...raw.outbounds]
          : [raw.outbounds],
  };
  showModal.value = true;
};

const closeModal = () => {
  showModal.value = false;
  isEditMode.value = false;
  editingTag.value = "";
  currentOutbound.value = blankOutbound();
};

const handleSave = async () => {
  const error = validateOutbound(currentOutbound.value);
  if (error) {
    toast.add({
      severity: "error",
      summary: t('outbounds.validation.title'),
      detail: t(error),
      life: 3000,
    });
    return;
  }

  localLoading.value = true;
  try {
    if (isEditMode.value) {
      await outboundsStore.updateOutbound(
        editingTag.value,
        currentOutbound.value as unknown as Outbound,
      );
      toast.add({
        severity: "success",
        summary: t('common.success'),
        detail: t('outbounds.toast.updatedOk'),
        life: 3000,
      });
    } else {
      await outboundsStore.addOutbound(currentOutbound.value as unknown as Outbound);
      toast.add({
        severity: "success",
        summary: t('common.success'),
        detail: t('outbounds.toast.addedOk'),
        life: 3000,
      });
    }
    closeModal();
  } catch (err: any) {
    toast.add({
      severity: "error",
      summary: t('common.error'),
      detail: err.message || t('outbounds.toast.saveFailed'),
      life: 3000,
    });
  } finally {
    localLoading.value = false;
  }
};

const deleteOutbound = async (outboundIndex: number, outbound: Outbound) => {
  localLoading.value = true;

  try {
    const val = outboundIndex > -1 ? outboundIndex : outbound.tag;
    await outboundsStore.deleteOutbound(val);
    toast.add({
      severity: "success",
      summary: t('common.success'),
      detail: t('outbounds.toast.deletedOk'),
      life: 3000,
    });
    closeDeleteConfirm();
  } catch (err: any) {
    toast.add({
      severity: "error",
      summary: t('common.error'),
      detail: err.message || t('outbounds.toast.deleteFailed'),
      life: 3000,
    });
  } finally {
    localLoading.value = false;
  }
};
const closeDeleteConfirm = () => {
  showDeleteConfirm.value = false;
  deletingOutbound.value = null;
};

const handleDelete = async () => {
  if (!deletingOutbound.value) return;
  localLoading.value = true;
  // -1 forces deleteOutbound to use outbound.tag — the confirm modal does
  // not know the array index of the row that opened it.
  await deleteOutbound(-1, deletingOutbound.value);
  closeDeleteConfirm();
};

// Import functions. Reset state immutably (new Set + empty array).
const resetImportFlow = () => {
  importInput.value = "";
  parsedNodes.value = [];
  selectedNodes.value = new Set();
};

const openImportModal = () => {
  resetImportFlow();
  showImportModal.value = true;
};

const closeImportModal = () => {
  showImportModal.value = false;
  resetImportFlow();
};

const parseSubscription = async () => {
  if (!importInput.value.trim()) {
    toast.add({
      severity: "error",
      summary: t('outbounds.validation.title'),
      detail: t('outbounds.toast.inputRequired'),
      life: 3000,
    });
    return;
  }

  parsing.value = true;
  parsedNodes.value = [];
  selectedNodes.value = new Set();

  try {
    const lines = importInput.value
      .split("\n")
      .map((line) => line.trim())
      .filter((line) => line.length > 0);

    const linesToParse = lines.join("\n");
    const { data } = await nodesService.parseNodes(linesToParse);
    parsedNodes.value = data.nodes;

    // Select all by default
    selectedNodes.value = new Set(data.nodes.map((n) => n.tag));

    if (data.nodes.length === 0) {
      toast.add({
        severity: "warn",
        summary: t('outbounds.toast.noNodesTitle'),
        detail: t('outbounds.toast.noNodesDetail'),
        life: 3000,
      });
    }
  } catch (err: any) {
    toast.add({
      severity: "error",
      summary: t('outbounds.errorTitle.parse'),
      detail: err.message || t('outbounds.toast.parseFailed'),
      life: 3000,
    });
  } finally {
    parsing.value = false;
  }
};

const toggleNode = (tag: string) => {
  const next = new Set(selectedNodes.value);
  if (next.has(tag)) next.delete(tag); else next.add(tag);
  selectedNodes.value = next;
};

const toggleSelectAll = () => {
  if (selectedNodes.value.size === parsedNodes.value.length) {
    selectedNodes.value = new Set();
  } else {
    selectedNodes.value = new Set(parsedNodes.value.map((n) => n.tag));
  }
};

const handleImport = async () => {
  importing.value = true;

  try {
    const outboundsToAdd: Outbound[] = [];

    parsedNodes.value.forEach((node) => {
      if (selectedNodes.value.has(node.tag)) {
        outboundsToAdd.push(node);
      }
    });

    if (outboundsToAdd.length > 0) {
      const { data } = await outboundsStore.addOutboundsBatch(outboundsToAdd);
      toast.add({
        severity: "success",
        summary: t('common.success'),
        detail: t('outbounds.toast.importedOk', { added: data.added, skipped: data.skipped }),
        life: 3000,
      });
      closeImportModal();
    }
  } catch (err: any) {
    toast.add({
      severity: "error",
      summary: t('outbounds.errorTitle.import'),
      detail: err.message || t('outbounds.toast.importFailed'),
      life: 3000,
    });
  } finally {
    importing.value = false;
  }
};

// Watch for modal close to reset form state
watch(showModal, (newValue) => {
  if (!newValue) {
    setTimeout(() => {
      isEditMode.value = false;
      editingTag.value = "";
      currentOutbound.value = {
        type: "direct",
        tag: "",
      };
    }, 300);
  }
});

watch(showDeleteConfirm, (newValue) => {
  if (!newValue) {
    setTimeout(() => {
      deletingOutbound.value = null;
    }, 300);
  }
});

// Reset fields when type changes (only in add mode, not edit mode)
watch(
  () => currentOutbound.value.type,
  (newType, oldType) => {
    // Only reset if we're in add mode and the type actually changed
    // Skip the initial undefined -> value change
    if (!isEditMode.value && oldType !== undefined && newType !== oldType) {
      const tag = currentOutbound.value.tag;
      const type = currentOutbound.value.type;
      currentOutbound.value = { type, tag };
    }
  }
);

onMounted(() => {
  outboundsStore.fetchOutbounds();
});
</script>

<template>
  <div>
    <div class="flex justify-end items-center mb-2">
      <div class="flex gap-3">
        <Button
          v-if="selectedOutbounds.size > 0"
          @click="handleBatchDelete"
          variant="danger"
        >
          <TrashIcon class="h-5 w-5 mr-2" />
          {{ $t('outbounds.deleteSelected', { count: selectedOutbounds.size }) }}
        </Button>
        <Button @click="openImportModal" variant="secondary">
          <ArrowDownTrayIcon class="h-5 w-5 mr-2" />
          {{ $t('outbounds.import') }}
        </Button>
        <Button @click="openAddModal" variant="primary">
          <PlusIcon class="h-5 w-5 mr-2" />
          {{ $t('outbounds.add') }}
        </Button>
      </div>
    </div>

    <div
      class="bg-white dark:bg-slate-800 rounded-surface shadow-surface overflow-hidden"
    >
      <!--
        The select-all checkbox means this header needs markup, not just
        labels, so it uses #head rather than the `columns` prop.
      -->
      <Table :loading="loading && outbounds.length === 0" :empty="outbounds.length === 0">
        <template #empty>
          <p class="mb-3">{{ $t('outbounds.empty') }}</p>
          <Button @click="openAddModal" variant="primary" size="sm">
            <PlusIcon class="h-4 w-4 mr-1.5" />
            {{ $t('outbounds.addFirst') }}
          </Button>
        </template>

        <template #head>
          <th>
            <input
              type="checkbox"
              :checked="selectedOutbounds.size === outbounds.length && outbounds.length > 0"
              :indeterminate="selectedOutbounds.size > 0 && selectedOutbounds.size < outbounds.length"
              class="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500"
              @change="toggleSelectAllOutbounds"
            />
          </th>
          <th>#</th>
          <th>{{ $t('outbounds.table.tag') }}</th>
          <th>{{ $t('outbounds.table.type') }}</th>
          <th>{{ $t('outbounds.table.server') }}</th>
          <th>{{ $t('outbounds.table.port') }}</th>
          <th class="col-actions">{{ $t('outbounds.table.actions') }}</th>
        </template>

        <tr v-for="(outbound, i) in outbounds" :key="outbound.tag">
          <td>
            <input
              type="checkbox"
              :checked="selectedOutbounds.has(outbound.tag)"
              class="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500"
              @change="toggleOutboundSelection(outbound.tag)"
            />
          </td>
          <td class="font-medium text-gray-900 dark:text-gray-100">{{ i + 1 }}</td>
          <td class="font-medium text-gray-900 dark:text-gray-100">{{ outbound.tag || i }}</td>
          <td>
            <Badge :variant="getOutboundBadgeVariant(outbound.type)">
              {{ getOutboundTypeLabel(outbound.type) }}
            </Badge>
          </td>
          <td class="text-gray-900 dark:text-gray-100">{{ (outbound as any).server || "-" }}</td>
          <td class="text-gray-900 dark:text-gray-100">{{ (outbound as any).server_port || "-" }}</td>
          <td class="col-actions font-medium">
            <div class="flex items-center justify-end gap-1">
              <Button @click="openEditModal(outbound)" variant="ghost" size="sm" action>
                <PencilIcon class="h-4 w-4" />
              </Button>
              <Button
                @click="deleteOutbound(i, outbound)"
                variant="ghost"
                size="sm"
                action
                class="text-red-600 dark:text-red-400 hover:text-red-700 dark:hover:text-red-300"
              >
                <TrashIcon class="h-4 w-4" />
              </Button>
            </div>
          </td>
        </tr>
      </Table>
    </div>

    <!-- Add/Edit Modal -->
    <Modal
      :model-value="showModal"
      @update:model-value="(v) => { if (!v) closeModal() }"
      :title="isEditMode ? $t('outbounds.modal.edit') : $t('outbounds.modal.add')"
      size="wide"
      show-close
    >
      <div class="space-y-3">
        <Alert v-if="managedWarning" type="warning" :title="$t('outbounds.form.managedByRulesTitle')">
          {{ managedWarning }}
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
      </div>

      <template #footer>
        <Button @click="closeModal" variant="secondary"
          >{{ $t('common.cancel') }}</Button
        >
        <Button
          @click="handleSave"
          variant="primary"
          :disabled="localLoading"
        >
          {{ isEditMode ? $t('common.update') : $t('common.add') }}
        </Button>
      </template>
    </Modal>

    <!-- Import Modal -->
    <Modal
      :model-value="showImportModal"
      @update:model-value="(v) => { if (!v) closeImportModal() }"
      :title="$t('outbounds.importModal.title')"
      size="lg"
      show-close
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
            <Button variant="ghost" @click="resetImportFlow">
              {{ $t('outbounds.importModal.backToInput') }}
            </Button>
            <div class="flex gap-3">
              <Button variant="secondary" @click="closeImportModal">
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
    </Modal>

    <!-- Delete Confirmation Modal -->
    <Modal
      :model-value="showDeleteConfirm"
      @update:model-value="(v) => { if (!v) closeDeleteConfirm() }"
      :title="$t('outbounds.del.title')"
      size="sm"
      show-close
    >
      <p class="text-gray-700 dark:text-gray-300">
        {{ $t('outbounds.del.confirm', { tag: deletingOutbound?.tag }) }}
      </p>

      <template #footer>
        <Button @click="closeDeleteConfirm" variant="secondary"
          >{{ $t('common.cancel') }}</Button
        >
        <Button
          @click="handleDelete"
          variant="danger"
          :disabled="localLoading"
        >
          {{ $t('common.delete') }}
        </Button>
      </template>
    </Modal>
  </div>
</template>
