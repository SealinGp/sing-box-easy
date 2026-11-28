import { ref, onMounted, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Outbound } from '../types/api'
import { useToast } from 'primevue/usetoast'
import { useOutboundsStore } from '../stores/outbounds'
import { OUTBOUND_GROUP_TYPES, OUTBOUND_TYPES_NEEDING_SERVER, OUTBOUND_TYPE_NAMES, OUTBOUND_TYPE_NOTES, applyTypeDefaults, pruneForType, type OutboundTypeName } from '../schemas/outboundFields'
import { isDeprecatedIn, isRetired } from '../schemas/optionSchema'
import { useSingBoxVersion } from './useSingBoxVersion'
import { storeToRefs } from 'pinia'

export function useOutboundsPage() {
const outboundsStore = useOutboundsStore();
// Use storeToRefs to maintain reactivity when destructuring
const { outbounds, loading, managedTags } = storeToRefs(outboundsStore);

const localLoading = ref(false);
const saveError = ref('');
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
const deletingOutboundIndex = ref(-1);

// Import modal state


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
  if (localLoading.value) return;
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
  if (localLoading.value) return;
  isEditMode.value = false;
  saveError.value = '';
  currentOutbound.value = blankOutbound();
  showModal.value = true;
};

const openEditModal = (outbound: Outbound) => {
  if (localLoading.value) return;
  isEditMode.value = true;
  saveError.value = '';
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
  if (localLoading.value) return;
  saveError.value = '';
  const error = validateOutbound(currentOutbound.value);
  if (error) {
    saveError.value = t(error);
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
    saveError.value = err.message || t('outbounds.toast.saveFailed');
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
  if (localLoading.value) return;
  if (!deletingOutbound.value) return;
  localLoading.value = true;
  await deleteOutbound(deletingOutboundIndex.value, deletingOutbound.value);
};

// Import functions. Reset state immutably (new Set + empty array).
// Watch for modal close to reset form state
watch(showModal, (newValue, _previous, onCleanup) => {
  if (!newValue) {
    const timer = setTimeout(() => {
      isEditMode.value = false;
      editingTag.value = "";
      currentOutbound.value = {
        type: "direct",
        tag: "",
      };
    }, 300);
    onCleanup(() => clearTimeout(timer));
  }
});

watch(showDeleteConfirm, (newValue, _previous, onCleanup) => {
  if (!newValue) {
    const timer = setTimeout(() => {
      deletingOutbound.value = null;
    }, 300);
    onCleanup(() => clearTimeout(timer));
  }
});

onMounted(() => {
  outboundsStore.fetchOutbounds();
});

return {
  saveError,
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
}
}
