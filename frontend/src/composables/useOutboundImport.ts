import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from 'primevue/usetoast'
import { nodesService } from '../services'
import { useOutboundsStore } from '../stores/outbounds'
import type { Outbound } from '../types/api'

export function useOutboundImport() {
const { t } = useI18n()
const toast = useToast()
const outboundsStore = useOutboundsStore()
const showImportModal = ref(false);
const importInput = ref("");
const parsing = ref(false);
const parsedNodes = ref<Outbound[]>([]);
const selectedNodes = ref<Set<string>>(new Set());
const importing = ref(false);
const resetImportFlow = () => {
  importInput.value = "";
  parsedNodes.value = [];
  selectedNodes.value = new Set();
};

const openImportModal = () => {
  if (parsing.value || importing.value) return;
  resetImportFlow();
  showImportModal.value = true;
};

const closeImportModal = () => {
  showImportModal.value = false;
  resetImportFlow();
};

const parseSubscription = async () => {
  if (parsing.value || importing.value) return;
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
  if (importing.value) return;
  const next = new Set(selectedNodes.value);
  if (next.has(tag)) next.delete(tag); else next.add(tag);
  selectedNodes.value = next;
};

const toggleSelectAll = () => {
  if (parsing.value || importing.value) return;
  if (selectedNodes.value.size === parsedNodes.value.length) {
    selectedNodes.value = new Set();
  } else {
    selectedNodes.value = new Set(parsedNodes.value.map((n) => n.tag));
  }
};

const handleImport = async () => {
  if (parsing.value || importing.value) return;
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


return { showImportModal, importInput, parsing, parsedNodes, selectedNodes, importing, openImportModal, closeImportModal, parseSubscription, toggleNode, toggleSelectAll, handleImport, resetImportFlow }
}
