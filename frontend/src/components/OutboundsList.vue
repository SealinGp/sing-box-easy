<script setup lang="ts">
import { ref, onMounted, watch, computed } from "vue";
import { useI18n } from "vue-i18n";
import {
  TransitionRoot,
  TransitionChild,
  Dialog,
  DialogPanel,
  DialogTitle,
} from "@headlessui/vue";
import type { Outbound } from "../types/api";
import type { OutboundType } from "../types/outbound";
import Button from "./Button.vue";
import Input from "./Input.vue";
import Badge from "./Badge.vue";
import Textarea from "./Textarea.vue";
import DialerOptions from "./DialerOptions.vue";
import { Chips } from "../volt";
import {
  PlusIcon,
  PencilIcon,
  TrashIcon,
  XMarkIcon,
  ArrowDownTrayIcon,
} from "@heroicons/vue/24/outline";
import { nodesService } from "../services";
import { useToast } from "primevue/usetoast";
import { useOutboundsStore } from "../stores/outbounds";
import { storeToRefs } from "pinia";

const outboundsStore = useOutboundsStore();
// Use storeToRefs to maintain reactivity when destructuring
const { outbounds, loading } = storeToRefs(outboundsStore);

const localLoading = ref(false);
const toast = useToast();
const { t } = useI18n();

// Modal state
const showModal = ref(false);
const isEditMode = ref(false);
const editingTag = ref<string>("");
const currentOutbound = ref<any>({
  type: "direct",
  tag: "",
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

const outboundTypes = computed(() => [
  { value: "direct", label: t('outbounds.types.direct') },
  { value: "block", label: t('outbounds.types.block') },
  { value: "socks", label: t('outbounds.types.socks') },
  { value: "http", label: t('outbounds.types.http') },
  { value: "shadowsocks", label: t('outbounds.types.shadowsocks') },
  { value: "vmess", label: t('outbounds.types.vmess') },
  { value: "trojan", label: t('outbounds.types.trojan') },
  { value: "vless", label: t('outbounds.types.vless') },
  { value: "hysteria", label: t('outbounds.types.hysteria') },
  { value: "hysteria2", label: t('outbounds.types.hysteria2') },
  { value: "tuic", label: t('outbounds.types.tuic') },
  { value: "wireguard", label: t('outbounds.types.wireguard') },
  { value: "ssh", label: t('outbounds.types.ssh') },
  { value: "tor", label: t('outbounds.types.tor') },
  { value: "shadowtls", label: t('outbounds.types.shadowtls') },
  { value: "selector", label: t('outbounds.types.selector') },
  { value: "urltest", label: t('outbounds.types.urltest') },
]);

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

const isProxyType = (type: OutboundType) => {
  const nonProxyTypes = ["direct", "block", "dns", "selector", "urltest"];
  return nonProxyTypes.indexOf(type) === -1;
};

const isGroupType = (type: OutboundType) => {
  const groupTypes = ["selector", "urltest"];
  return groupTypes.indexOf(type) !== -1;
};

const needsServer = computed(() => isProxyType(currentOutbound.value.type));
const needsPassword = computed(() => {
  const types = ["shadowsocks", "trojan", "hysteria2", "tuic"];
  return types.indexOf(currentOutbound.value.type) !== -1;
});
const needsUUID = computed(() => {
  const types = ["vmess", "vless", "tuic"];
  return types.indexOf(currentOutbound.value.type) !== -1;
});
const needsMethod = computed(
  () => currentOutbound.value.type === "shadowsocks"
);
const needsOutbounds = computed(() => isGroupType(currentOutbound.value.type));

const openAddModal = () => {
  isEditMode.value = false;
  currentOutbound.value = {
    type: "direct",
    tag: "",
  };
  showModal.value = true;
};

const openEditModal = (outbound: Outbound) => {
  isEditMode.value = true;
  editingTag.value = outbound.tag;
  // sing-box accepts scalar OR array for `outbounds` on the wire; coerce to
  // array so the Chips v-model sees the shape it expects and the save path
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
  currentOutbound.value = {
    type: "direct",
    tag: "",
  };
};

const handleSave = async () => {
  // Validation
  if (!currentOutbound.value.tag?.trim()) {
    toast.add({
      severity: "error",
      summary: t('outbounds.validation.title'),
      detail: t('outbounds.validation.tagRequired'),
      life: 3000,
    });
    return;
  }

  if (!currentOutbound.value.type) {
    toast.add({
      severity: "error",
      summary: t('outbounds.validation.title'),
      detail: t('outbounds.validation.typeRequired'),
      life: 3000,
    });
    return;
  }

  // Validate proxy-specific fields
  if (needsServer.value) {
    if (!currentOutbound.value.server?.trim()) {
      toast.add({
        severity: "error",
        summary: t('outbounds.validation.title'),
        detail: t('outbounds.validation.serverRequired'),
        life: 3000,
      });
      return;
    }
    if (!currentOutbound.value.server_port) {
      toast.add({
        severity: "error",
        summary: t('outbounds.validation.title'),
        detail: t('outbounds.validation.portRequired'),
        life: 3000,
      });
      return;
    }
  }

  if (needsPassword.value && !currentOutbound.value.password?.trim()) {
    toast.add({
      severity: "error",
      summary: t('outbounds.validation.title'),
      detail: t('outbounds.validation.passwordRequired'),
      life: 3000,
    });
    return;
  }

  if (needsUUID.value && !currentOutbound.value.uuid?.trim()) {
    toast.add({
      severity: "error",
      summary: t('outbounds.validation.title'),
      detail: t('outbounds.validation.uuidRequired'),
      life: 3000,
    });
    return;
  }

  if (needsMethod.value && !currentOutbound.value.method?.trim()) {
    toast.add({
      severity: "error",
      summary: t('outbounds.validation.title'),
      detail: t('outbounds.validation.methodRequired'),
      life: 3000,
    });
    return;
  }

  if (
    needsOutbounds.value &&
    (!currentOutbound.value.outbounds ||
      currentOutbound.value.outbounds.length === 0)
  ) {
    toast.add({
      severity: "error",
      summary: t('outbounds.validation.title'),
      detail: t('outbounds.validation.outboundsRequired'),
      life: 3000,
    });
    return;
  }

  localLoading.value = true;
  try {
    if (isEditMode.value) {
      await outboundsStore.updateOutbound(
        editingTag.value,
        currentOutbound.value
      );
      toast.add({
        severity: "success",
        summary: t('common.success'),
        detail: t('outbounds.toast.updatedOk'),
        life: 3000,
      });
    } else {
      await outboundsStore.addOutbound(currentOutbound.value);
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
    <div class="flex justify-end items-center mb-4">
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
      class="bg-white dark:bg-slate-800 rounded-lg shadow dark:shadow-xl dark:shadow-slate-700/50 overflow-hidden"
    >
      <div
        v-if="loading && outbounds.length === 0"
        class="flex items-center justify-center py-12"
      >
        <div
          class="animate-spin rounded-full h-8 w-8 border-b-2 border-violet-600"
        ></div>
      </div>

      <div v-else-if="outbounds.length === 0" class="text-center py-12">
        <p class="text-gray-500 dark:text-gray-500 mb-4">
          {{ $t('outbounds.empty') }}
        </p>
        <Button @click="openAddModal" variant="primary" size="sm">
          <PlusIcon class="h-4 w-4 mr-2" />
          {{ $t('outbounds.addFirst') }}
        </Button>
      </div>

      <div v-else class="overflow-x-auto">
        <table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
          <thead class="bg-gray-50 dark:bg-gray-700">
            <tr>
              <th
                class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider"
              >
                <input
                  type="checkbox"
                  :checked="selectedOutbounds.size === outbounds.length && outbounds.length > 0"
                  :indeterminate="selectedOutbounds.size > 0 && selectedOutbounds.size < outbounds.length"
                  class="w-4 h-4 text-violet-600 border-gray-300 rounded focus:ring-violet-500"
                  @change="toggleSelectAllOutbounds"
                />
              </th>
              <th
                class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider"
              >
                #
              </th>
              <th
                class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider"
              >
                {{ $t('outbounds.table.tag') }}
              </th>
              <th
                class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider"
              >
                {{ $t('outbounds.table.type') }}
              </th>
              <th
                class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider"
              >
                {{ $t('outbounds.table.server') }}
              </th>
              <th
                class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider"
              >
                {{ $t('outbounds.table.port') }}
              </th>
              <th
                class="px-4 py-2 text-right text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider"
              >
                {{ $t('outbounds.table.actions') }}
              </th>
            </tr>
          </thead>
          <tbody
            class="bg-white dark:bg-slate-800 divide-y divide-gray-200 dark:divide-gray-700"
          >
            <tr
              v-for="(outbound, i) in outbounds"
              :key="outbound.tag"
              class="hover:bg-gray-50 dark:hover:bg-gray-700"
            >
              <td class="px-4 py-4 whitespace-nowrap">
                <input
                  type="checkbox"
                  :checked="selectedOutbounds.has(outbound.tag)"
                  class="w-4 h-4 text-violet-600 border-gray-300 rounded focus:ring-violet-500"
                  @change="toggleOutboundSelection(outbound.tag)"
                />
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <div
                  class="text-sm font-medium text-gray-900 dark:text-gray-100"
                >
                  {{ i + 1 }}
                </div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <div
                  class="text-sm font-medium text-gray-900 dark:text-gray-100"
                >
                  {{ outbound.tag || i }}
                </div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <Badge :variant="getOutboundBadgeVariant(outbound.type)">
                  {{ getOutboundTypeLabel(outbound.type) }}
                </Badge>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="text-sm text-gray-900 dark:text-gray-100">
                  {{ (outbound as any).server || "-" }}
                </div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="text-sm text-gray-900 dark:text-gray-100">
                  {{ (outbound as any).server_port || "-" }}
                </div>
              </td>
              <td
                class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium"
              >
                <div class="outbound-table-actions flex items-center justify-end gap-2">
                  <Button
                    @click="openEditModal(outbound)"
                    variant="ghost"
                    size="sm"
                    action
                  >
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
          </tbody>
        </table>
      </div>
    </div>

    <!-- Add/Edit Modal -->
    <TransitionRoot appear :show="showModal" as="template">
      <Dialog as="div" @close="closeModal" class="relative z-50">
        <TransitionChild
          as="template"
          enter="duration-300 ease-out"
          enter-from="opacity-0"
          enter-to="opacity-100"
          leave="duration-200 ease-in"
          leave-from="opacity-100"
          leave-to="opacity-0"
        >
          <div class="fixed inset-0 bg-black/50 dark:bg-black/70" />
        </TransitionChild>

        <div class="fixed inset-0 overflow-y-auto">
          <div
            class="flex min-h-full items-center justify-center p-4 text-center"
          >
            <TransitionChild
              as="template"
              enter="duration-300 ease-out"
              enter-from="opacity-0 scale-95"
              enter-to="opacity-100 scale-100"
              leave="duration-200 ease-in"
              leave-from="opacity-100 scale-100"
              leave-to="opacity-0 scale-95"
            >
              <DialogPanel
                class="w-full max-w-3xl transform overflow-hidden rounded-lg bg-white dark:bg-slate-800 text-left align-middle shadow-xl transition-all flex flex-col max-h-[90vh]"
              >
                <div
                  class="flex items-center justify-between p-6 pb-4 border-b border-gray-200 dark:border-gray-700"
                >
                  <DialogTitle
                    as="h3"
                    class="text-lg font-semibold text-gray-900 dark:text-gray-100"
                  >
                    {{ isEditMode ? $t('outbounds.modal.edit') : $t('outbounds.modal.add') }}
                  </DialogTitle>
                  <button
                    type="button"
                    class="text-gray-400 hover:text-gray-500 transition-colors"
                    @click="closeModal"
                  >
                    <XMarkIcon class="h-6 w-6" />
                  </button>
                </div>

                <div class="space-y-4 overflow-y-auto flex-1 p-6 pt-4">
                  <div class="grid grid-cols-2 gap-4">
                    <div>
                      <label
                        class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
                        >{{ $t('outbounds.form.tag') }}</label
                      >
                      <Input
                        v-model="currentOutbound.tag"
                        :placeholder="$t('outbounds.form.tagPlaceholder')"
                        :disabled="isEditMode"
                      />
                      <p class="mt-1 text-xs text-gray-500">
                        {{ $t('outbounds.form.tagHelp') }}
                      </p>
                    </div>

                    <div>
                      <label
                        class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
                        >{{ $t('outbounds.form.type') }}</label
                      >
                      <select
                        class="select"
                        v-model="currentOutbound.type"
                        :disabled="isEditMode"
                      >
                        <option disabled selected>{{ $t('outbounds.form.typePlaceholder') }}</option>
                        <option
                          v-for="outboundType in outboundTypes"
                          :value="outboundType.value"
                        >
                          {{ outboundType.label }}
                        </option>
                      </select>
                    </div>
                  </div>

                  <!-- Proxy Server Fields -->
                  <div v-if="needsServer" class="grid grid-cols-2 gap-4">
                    <div class="col-span-1">
                      <label
                        class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
                        >{{ $t('outbounds.form.server') }}</label
                      >
                      <Input
                        v-model="currentOutbound.server"
                        :placeholder="$t('outbounds.form.serverPlaceholder')"
                      />
                    </div>

                    <div class="col-span-1">
                      <label
                        class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
                        >{{ $t('outbounds.form.port') }}</label
                      >
                      <Input
                        v-model.number="currentOutbound.server_port"
                        type="number"
                        :placeholder="$t('outbounds.form.portPlaceholder')"
                      />
                    </div>
                  </div>

                  <!-- Method (Shadowsocks) -->
                  <div v-if="needsMethod">
                    <label
                      class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
                      >{{ $t('outbounds.form.method') }}</label
                    >
                    <Input
                      v-model="currentOutbound.method"
                      :placeholder="$t('outbounds.form.methodPlaceholder')"
                    />
                  </div>

                  <!-- Password -->
                  <div v-if="needsPassword">
                    <label
                      class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
                      >{{ $t('outbounds.form.password') }}</label
                    >
                    <Input
                      v-model="currentOutbound.password"
                      type="password"
                      :placeholder="$t('outbounds.form.passwordPlaceholder')"
                    />
                  </div>

                  <!-- UUID (VMess, VLESS, TUIC) -->
                  <div v-if="needsUUID">
                    <label
                      class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
                      >{{ $t('outbounds.form.uuid') }}</label
                    >
                    <Input
                      v-model="currentOutbound.uuid"
                      :placeholder="$t('outbounds.form.uuidPlaceholder')"
                    />
                  </div>

                  <!-- Group Outbounds (Selector, URLTest) -->
                  <!-- Uses Chips (string[]) so the v-model shape matches the
                       Selector/URLTest schema. sing-box also accepts a scalar
                       string on the wire; openEditModal() coerces to array. -->
                  <div v-if="needsOutbounds">
                    <label
                      class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
                      >{{ $t('outbounds.form.outbounds') }}</label
                    >
                    <Chips
                      v-model="currentOutbound.outbounds"
                      :placeholder="$t('outbounds.form.outboundsPlaceholder')"
                      class="w-full"
                    />
                    <p class="mt-1 text-xs text-gray-500">
                      {{ $t('outbounds.form.outboundsHelp') }}
                    </p>
                  </div>

                  <!-- URLTest specific -->
                  <div
                    v-if="currentOutbound.type === 'urltest'"
                    class="grid grid-cols-2 gap-4"
                  >
                    <div>
                      <label
                        class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
                        >{{ $t('outbounds.form.testUrl') }}</label
                      >
                      <Input
                        v-model="currentOutbound.url"
                        :placeholder="$t('outbounds.form.testUrlPlaceholder')"
                      />
                    </div>
                    <div>
                      <label
                        class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
                        >{{ $t('outbounds.form.interval') }}</label
                      >
                      <Input
                        v-model="currentOutbound.interval"
                        :placeholder="$t('outbounds.form.intervalPlaceholder')"
                      />
                    </div>
                  </div>

                  <!-- DialerOptions for all outbound types except block and dns -->
                  <DialerOptions
                    v-if="
                      currentOutbound.type !== 'block' &&
                      currentOutbound.type !== 'dns'
                    "
                    v-model="currentOutbound"
                    :current-tag="isEditMode ? editingTag : currentOutbound.tag"
                    :show-advanced="true"
                  />
                </div>

                <div
                  class="flex justify-end gap-3 p-6 pt-4 border-t border-gray-200 dark:border-gray-700"
                >
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
                </div>
              </DialogPanel>
            </TransitionChild>
          </div>
        </div>
      </Dialog>
    </TransitionRoot>

    <!-- Import Modal -->
    <TransitionRoot appear :show="showImportModal" as="template">
      <Dialog as="div" @close="closeImportModal" class="relative z-50">
        <TransitionChild
          as="template"
          enter="duration-300 ease-out"
          enter-from="opacity-0"
          enter-to="opacity-100"
          leave="duration-200 ease-in"
          leave-from="opacity-100"
          leave-to="opacity-0"
        >
          <div class="fixed inset-0 bg-black/50 dark:bg-black/70" />
        </TransitionChild>

        <div class="fixed inset-0 overflow-y-auto">
          <div
            class="flex min-h-full items-center justify-center p-4 text-center"
          >
            <TransitionChild
              as="template"
              enter="duration-300 ease-out"
              enter-from="opacity-0 scale-95"
              enter-to="opacity-100 scale-100"
              leave="duration-200 ease-in"
              leave-from="opacity-100 scale-100"
              leave-to="opacity-0 scale-95"
            >
              <DialogPanel
                class="w-full max-w-2xl transform overflow-hidden rounded-lg bg-white dark:bg-slate-800 p-6 text-left align-middle shadow-xl transition-all"
              >
                <div class="flex items-center justify-between mb-4">
                  <DialogTitle
                    as="h3"
                    class="text-lg font-semibold text-gray-900 dark:text-gray-100"
                  >
                    {{ $t('outbounds.importModal.title') }}
                  </DialogTitle>
                  <button
                    type="button"
                    class="text-gray-400 hover:text-gray-500 transition-colors"
                    @click="closeImportModal"
                  >
                    <XMarkIcon class="h-6 w-6" />
                  </button>
                </div>

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
                      class="max-h-80 overflow-y-auto border border-gray-200 dark:border-gray-700 rounded-lg"
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
                          class="w-4 h-4 text-violet-600 border-gray-300 rounded focus:ring-violet-500"
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
              </DialogPanel>
            </TransitionChild>
          </div>
        </div>
      </Dialog>
    </TransitionRoot>

    <!-- Delete Confirmation Modal -->
    <TransitionRoot appear :show="showDeleteConfirm" as="template">
      <Dialog as="div" @close="closeDeleteConfirm" class="relative z-50">
        <TransitionChild
          as="template"
          enter="duration-300 ease-out"
          enter-from="opacity-0"
          enter-to="opacity-100"
          leave="duration-200 ease-in"
          leave-from="opacity-100"
          leave-to="opacity-0"
        >
          <div class="fixed inset-0 bg-black/50 dark:bg-black/70" />
        </TransitionChild>

        <div class="fixed inset-0 overflow-y-auto">
          <div
            class="flex min-h-full items-center justify-center p-4 text-center"
          >
            <TransitionChild
              as="template"
              enter="duration-300 ease-out"
              enter-from="opacity-0 scale-95"
              enter-to="opacity-100 scale-100"
              leave="duration-200 ease-in"
              leave-from="opacity-100 scale-100"
              leave-to="opacity-0 scale-95"
            >
              <DialogPanel
                class="w-full max-w-md transform overflow-hidden rounded-lg bg-white dark:bg-slate-800 p-6 text-left align-middle shadow-xl transition-all"
              >
                <div class="flex items-center justify-between mb-4">
                  <DialogTitle
                    as="h3"
                    class="text-lg font-semibold text-gray-900 dark:text-gray-100"
                  >
                    {{ $t('outbounds.del.title') }}
                  </DialogTitle>
                  <button
                    type="button"
                    class="text-gray-400 hover:text-gray-500 transition-colors"
                    @click="closeDeleteConfirm"
                  >
                    <XMarkIcon class="h-6 w-6" />
                  </button>
                </div>

                <p class="text-gray-700 dark:text-gray-300">
                  {{ $t('outbounds.del.confirm', { tag: deletingOutbound?.tag }) }}
                </p>

                <div class="mt-6 flex justify-end gap-3">
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
                </div>
              </DialogPanel>
            </TransitionChild>
          </div>
        </div>
      </Dialog>
    </TransitionRoot>
  </div>
</template>
