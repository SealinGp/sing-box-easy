<script setup lang="ts">
import { ref, onMounted, computed } from "vue";
import { useI18n } from "vue-i18n";
import { apiService } from "../../services/api";
import { SubscriptionService } from "../../services/subscription";
import { type Subscription } from "../../types/api";
import { useNotify } from "../../composables/useNotify";
import Button from "../../components/Button.vue";
import Modal from "../../components/Modal.vue";
import Input from "../../components/Input.vue";
import Badge from "../../components/Badge.vue";
import { parseDurationToHours, isValidDuration } from "../../plugins/dayjs";
import {
  PlusIcon,
  PencilIcon,
  TrashIcon,
  ArrowPathIcon,
  ServerIcon,
  CheckCircleIcon,
  XCircleIcon,
  ClockIcon,
  ClipboardDocumentIcon,
} from "@heroicons/vue/24/outline";

const subscriptionService = new SubscriptionService(apiService);
const { t } = useI18n();
const notify = useNotify();

interface FormData {
  name: string;
  url: string;
  auto_update: boolean;
  update_interval: string;
  fetch_mode: "" | "clean_dns" | "proxy";
  proxy_url: string;
}

const subscriptions = ref<Subscription[]>([]);
const isLoading = ref(false);
const isUpdating = ref<string[]>([]);
const showModal = ref(false);
const editingSubscription = ref<Subscription | null>(null);
const showDeleteConfirm = ref(false);
const deletingSubscriptionId = ref<string>("");

// Form data
const formData = ref<FormData>({
  name: "",
  url: "",
  auto_update: true,
  update_interval: "24h",
  fetch_mode: "",
  proxy_url: "",
});

// Form errors
const formErrors = ref<Record<string, string>>({});

// Computed properties
const isEditing = computed(() => !!editingSubscription.value);
const modalTitle = computed(() =>
  isEditing.value ? t("subscriptions.modal.edit") : t("subscriptions.modal.add")
);
const isFormValid = computed(() => {
  return formData.value.name.trim() !== "" && formData.value.url.trim() !== "";
});

// Methods
const loadSubscriptions = async () => {
  try {
    isLoading.value = true;
    const response = await subscriptionService.getSubscriptions();
    subscriptions.value = response.data.subscriptions || [];
  } catch (error) {
    notify.apiError(error, t("subscriptions.notify.loadError"));
  } finally {
    isLoading.value = false;
  }
};

const resetForm = () => {
  formData.value = {
    name: "",
    url: "",
    auto_update: true,
    update_interval: "24h",
    fetch_mode: "",
    proxy_url: "",
  } as FormData;
  formErrors.value = {};
  editingSubscription.value = null;
};

const openAddModal = () => {
  resetForm();
  showModal.value = true;
};

const openEditModal = (subscription: Subscription) => {
  formData.value = {
    name: subscription.name,
    url: subscription.url,
    auto_update: subscription.auto_update ?? false,
    update_interval: subscription.update_interval, // Already a string from backend (e.g., "24h")
    fetch_mode: subscription.fetch_mode ?? "",
    proxy_url: subscription.proxy_url ?? "",
  };
  editingSubscription.value = subscription;
  showModal.value = true;
};

const validateForm = () => {
  const errors: Record<string, string> = {};

  if (!formData.value.name.trim()) {
    errors.name = t("subscriptions.validation.nameRequired");
  }

  if (!formData.value.url.trim()) {
    errors.url = t("subscriptions.validation.urlRequired");
  } else {
    // The backend fetches this URL server-side. Reject non-http(s) schemes
    // (file://, gopher://, etc.) to avoid SSRF / local-file disclosure.
    try {
      const parsed = new URL(formData.value.url);
      if (!["http:", "https:"].includes(parsed.protocol)) {
        errors.url = t("subscriptions.validation.urlScheme");
      }
    } catch {
      errors.url = t("subscriptions.validation.urlInvalid");
    }
  }

  // Validate update_interval format if auto_update is enabled
  if (formData.value.auto_update) {
    if (!formData.value.update_interval) {
      errors.update_interval = t("subscriptions.validation.intervalRequired");
    } else if (!isValidDuration(formData.value.update_interval)) {
      errors.update_interval = t("subscriptions.validation.intervalInvalid");
    } else {
      const hours = parseDurationToHours(formData.value.update_interval);
      if (hours && hours < 1) {
        errors.update_interval = t("subscriptions.validation.intervalMin");
      }
    }
  }

  formErrors.value = errors;
  return Object.keys(errors).length === 0;
};

const saveSubscription = async () => {
  if (!validateForm()) {
    return;
  }

  try {
    const subscriptionData = {
      name: formData.value.name,
      url: formData.value.url,
      auto_update: formData.value.auto_update,
      update_interval: formData.value.update_interval, // Send as string
      fetch_mode: formData.value.fetch_mode,
      // proxy_url only matters in proxy mode; send empty otherwise.
      proxy_url:
        formData.value.fetch_mode === "proxy"
          ? formData.value.proxy_url.trim()
          : "",
    };

    let response;
    if (isEditing.value && editingSubscription.value) {
      response = await subscriptionService.updateSubscription(
        editingSubscription.value.id,
        subscriptionData
      );
    } else {
      response = await subscriptionService.addSubscription(subscriptionData);
    }

    notify.success(response.data.message || t("subscriptions.notify.savedOk"));
    showModal.value = false;
    resetForm();
    await loadSubscriptions();
  } catch (error) {
    notify.apiError(error, t("subscriptions.notify.saveError"));
  }
};

const copyUrl = async (url: string) => {
  try {
    if (navigator.clipboard?.writeText) {
      // Secure context (HTTPS / localhost)
      await navigator.clipboard.writeText(url);
    } else {
      // Fallback for plain-HTTP deployments where navigator.clipboard is undefined
      const ta = document.createElement("textarea");
      ta.value = url;
      ta.style.cssText =
        "position:fixed;top:0;left:0;opacity:0;pointer-events:none";
      document.body.appendChild(ta);
      ta.focus();
      ta.select();
      if (!document.execCommand("copy")) {
        throw new Error("execCommand copy failed");
      }
      document.body.removeChild(ta);
    }
    notify.success(t("subscriptions.notify.copiedOk"));
  } catch (error) {
    notify.apiError(error, t("subscriptions.notify.copyFailed"));
  }
};

const confirmDelete = (subscription: Subscription) => {
  deletingSubscriptionId.value = subscription.id;
  showDeleteConfirm.value = true;
};

const deleteSubscription = async () => {
  if (!deletingSubscriptionId.value) return;

  try {
    const response = await subscriptionService.deleteSubscription(
      deletingSubscriptionId.value
    );
    notify.success(
      response.data.message || t("subscriptions.notify.deletedOk")
    );
    showDeleteConfirm.value = false;
    deletingSubscriptionId.value = "";
    await loadSubscriptions();
  } catch (error) {
    notify.apiError(error, t("subscriptions.notify.deleteError"));
  }
};

const updateSubscription = async (subscription: Subscription) => {
  if (isUpdating.value.includes(subscription.id)) return;

  try {
    isUpdating.value.push(subscription.id);
    const response = await subscriptionService.updateSubscriptionContent(
      subscription.id
    );

    const { added, updated, deleted } = response.data;
    // Backend now returns a 3-way diff. Build a human-readable summary
    // and fall back to "no changes" when the subscription was already
    // in sync with the upstream feed.
    const parts: string[] = [];
    if (added > 0) parts.push(t("subscriptions.notify.added", { n: added }));
    if (updated > 0)
      parts.push(t("subscriptions.notify.updated", { n: updated }));
    if (deleted > 0)
      parts.push(t("subscriptions.notify.removed", { n: deleted }));
    const summary =
      parts.length > 0 ? parts.join(", ") : t("subscriptions.notify.noChanges");
    notify.success(
      t("subscriptions.notify.synced", { name: subscription.name, summary })
    );
    await loadSubscriptions();
  } catch (error) {
    notify.apiError(error, t("subscriptions.notify.updateError"));
  } finally {
    const index = isUpdating.value.indexOf(subscription.id);
    if (index > -1) {
      isUpdating.value.splice(index, 1);
    }
  }
};

const updateAllSubscriptions = async () => {
  for (const subscription of subscriptions.value) {
    await updateSubscription(subscription);
    // Small delay between requests to avoid overwhelming the server
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }
};

const formatDate = (dateString?: string) => {
  if (!dateString) return t("common.never");
  return new Date(dateString).toLocaleString();
};

const getIntervalHours = (interval: string | undefined) => {
  // Use dayjs parser for more flexible duration parsing
  const hours = parseDurationToHours(interval);
  return hours ?? 24; // Default to 24 hours if invalid or undefined
};

const getStatusBadge = (subscription: Subscription) => {
  if (!subscription.last_update) {
    return {
      type: "warning" as const,
      icon: ClockIcon,
      text: t("subscriptions.status.notUpdated"),
    };
  }

  // "Outdated" only has meaning for auto-updating subscriptions: it mirrors the
  // backend's own refresh rule (AutoUpdater.shouldUpdate), which considers a
  // subscription due once time-since-last-update >= its update_interval. When
  // auto-update is off there is no expected cadence, so we never flag outdated.
  const autoUpdate = subscription.auto_update ?? false;
  if (autoUpdate) {
    const lastUpdate = new Date(subscription.last_update);
    const hoursSinceUpdate =
      (Date.now() - lastUpdate.getTime()) / (1000 * 60 * 60);
    const intervalHours = getIntervalHours(subscription.update_interval); // handles undefined

    if (hoursSinceUpdate >= intervalHours) {
      return {
        type: "warning" as const,
        icon: ClockIcon,
        text: t("subscriptions.status.outdated"),
      };
    }
  }

  return {
    type: "success" as const,
    icon: CheckCircleIcon,
    text: t("subscriptions.status.updated"),
  };
};

// Lifecycle
onMounted(() => {
  loadSubscriptions();
});
</script>

<template>
  <div>
    <!-- Header: subtitle + actions (page title is owned by the Outbounds TabNav) -->
    <div class="flex items-center justify-between mb-4">
      <p class="text-gray-500 dark:text-gray-400">
        {{ $t("subscriptions.subtitle") }}
      </p>
      <div class="flex gap-3">
        <Button
          variant="secondary"
          :loading="isLoading && !isUpdating.length"
          @click="loadSubscriptions"
          :disabled="isLoading"
        >
          <ArrowPathIcon class="h-5 w-5" />
          {{ $t("subscriptions.refresh") }}
        </Button>
        <Button
          v-if="subscriptions.length > 0"
          variant="secondary"
          :loading="isUpdating.length === subscriptions.length"
          @click="updateAllSubscriptions"
          :disabled="isUpdating.length > 0"
        >
          <ArrowPathIcon class="h-5 w-5" />
          {{ $t("subscriptions.updateAll") }}
        </Button>
        <Button @click="openAddModal">
          <PlusIcon class="h-5 w-5" />
          {{ $t("subscriptions.add") }}
        </Button>
      </div>
    </div>

    <!-- Subscriptions List -->
    <div
      class="bg-white dark:bg-slate-800 rounded-surface shadow dark:shadow-float dark:shadow-slate-700/50 overflow-hidden"
    >
      <div
        v-if="isLoading && subscriptions.length === 0"
        class="p-12 text-center"
      >
        <div
          class="inline-flex items-center justify-center w-16 h-16 bg-primary-100 dark:bg-primary-900 rounded-pill mb-4"
        >
          <ServerIcon class="h-8 w-8 text-primary-600 dark:text-primary-400" />
        </div>
        <p class="text-gray-500 dark:text-gray-400">
          {{ $t("subscriptions.loading") }}
        </p>
      </div>

      <div v-else-if="subscriptions.length === 0" class="p-12 text-center">
        <div
          class="inline-flex items-center justify-center w-16 h-16 bg-gray-100 dark:bg-gray-700 rounded-pill mb-4"
        >
          <ServerIcon class="h-8 w-8 text-gray-400" />
        </div>
        <h3 class="text-lg font-medium text-gray-900 dark:text-gray-100 mb-2">
          {{ $t("subscriptions.empty.title") }}
        </h3>
        <p class="text-gray-500 dark:text-gray-400 mb-6">
          {{ $t("subscriptions.empty.desc") }}
        </p>
        <Button @click="openAddModal">
          <PlusIcon class="h-5 w-5" />
          {{ $t("subscriptions.add") }}
        </Button>
      </div>

      <div v-else>
        <table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
          <thead class="bg-gray-50 dark:bg-gray-700">
            <tr>
              <th
                class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider"
              >
                {{ $t("subscriptions.table.subscription") }}
              </th>
              <th
                class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider"
              >
                {{ $t("subscriptions.table.status") }}
              </th>
              <th
                class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider"
              >
                {{ $t("subscriptions.table.info") }}
              </th>
              <th
                class="px-4 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider"
              >
                {{ $t("subscriptions.table.actions") }}
              </th>
            </tr>
          </thead>
          <tbody
            class="bg-white dark:bg-slate-800 divide-y divide-gray-200 dark:divide-gray-700"
          >
            <tr
              v-for="subscription in subscriptions"
              :key="subscription.id"
              class="hover:bg-gray-50 dark:hover:bg-gray-700"
            >
              <!-- Subscription: name + URL (copy) + node count, stacked -->
              <td class="px-4 py-3 align-top">
                <div class="flex flex-col gap-1 max-w-[18rem]">
                  <span class="text-sm font-medium text-gray-900 dark:text-gray-100 truncate" :title="subscription.name">
                    {{ subscription.name }}
                  </span>
                  <div class="flex items-center gap-1 min-w-0">
                    <span
                      class="text-xs text-gray-500 dark:text-gray-400 truncate"
                      :title="subscription.url"
                    >
                      {{ subscription.url }}
                    </span>
                    <button
                      type="button"
                      class="shrink-0 p-0.5 text-gray-400 hover:text-primary-600 dark:hover:text-primary-400 transition-colors"
                      @click="copyUrl(subscription.url)"
                      :title="$t('subscriptions.tooltip.copy')"
                      :aria-label="$t('subscriptions.tooltip.copy')"
                    >
                      <ClipboardDocumentIcon class="h-3.5 w-3.5" />
                    </button>
                  </div>
                </div>
              </td>
              <!-- Status: state badge + auto-update/interval + last update, stacked -->
              <td class="px-4 py-3 align-top">
                <div class="flex flex-col items-start gap-1.5">
                  <Badge
                    :variant="getStatusBadge(subscription).type"
                    class="inline-flex items-center gap-1"
                  >
                    <component
                      :is="getStatusBadge(subscription).icon"
                      class="h-3 w-3"
                    />
                    {{ getStatusBadge(subscription).text }}
                  </Badge>
                  <div class="flex items-center gap-1.5 flex-wrap">
                    <Badge
                      :variant="subscription.auto_update ? 'success' : 'secondary'"
                      class="inline-flex items-center gap-1"
                    >
                      <component
                        :is="subscription.auto_update ? CheckCircleIcon : XCircleIcon"
                        class="h-3 w-3"
                      />
                      {{
                        subscription.auto_update
                          ? $t("subscriptions.enabled")
                          : $t("subscriptions.disabled")
                      }}
                    </Badge>
                    <span
                      v-if="subscription.auto_update && subscription.update_interval"
                      class="inline-flex items-center gap-1 text-xs text-gray-500 dark:text-gray-400"
                      :title="$t('subscriptions.form.updateInterval')"
                    >
                      <ClockIcon class="h-3 w-3" />
                      {{ subscription.update_interval }}
                    </span>
                  </div>
                  <span
                    class="text-xs text-gray-400 dark:text-gray-500 whitespace-nowrap"
                    :title="$t('subscriptions.table.lastUpdate')"
                  >
                    {{ $t("subscriptions.table.lastUpdate") }}: {{ formatDate(subscription.last_update) }}
                  </span>
                </div>
              </td>
              <!-- Plan info: generic key/value entries -->
              <td class="px-4 py-3 align-top">
                <div
                  v-if="subscription.info && subscription.info.length"
                  class="flex flex-col gap-1"
                >
                  <span
                    v-for="(entry, i) in subscription.info"
                    :key="i"
                    class="text-xs text-gray-600 dark:text-gray-300 whitespace-nowrap"
                    :title="`${entry.key}: ${entry.value}`"
                  >
                    <span class="text-gray-400 dark:text-gray-500">{{ entry.key }}</span>
                    <span v-if="entry.value" class="font-medium ml-1">{{ entry.value }}</span>
                  </span>
                </div>
                <span v-else class="text-xs text-gray-300 dark:text-gray-600">—</span>
              </td>
              <td
                class="px-4 py-3 align-top text-right text-sm font-medium"
              >
                <div class="flex items-center justify-end gap-2">
                  <Button
                    variant="ghost"
                    size="sm"
                    :loading="isUpdating.includes(subscription.id)"
                    @click="updateSubscription(subscription)"
                    :disabled="isUpdating.includes(subscription.id)"
                    :title="$t('subscriptions.tooltip.update')"
                    :aria-label="$t('subscriptions.tooltip.update')"
                  >
                    <!-- Hide the icon while updating: the Button shows only its
                         spinner and is disabled, so the row can't be re-triggered. -->
                    <ArrowPathIcon
                      v-if="!isUpdating.includes(subscription.id)"
                      class="h-4 w-4"
                    />
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    @click="openEditModal(subscription)"
                    :title="$t('subscriptions.tooltip.edit')"
                    :aria-label="$t('subscriptions.tooltip.edit')"
                  >
                    <PencilIcon class="h-4 w-4" />
                  </Button>
                  <Button
                    variant="danger"
                    size="sm"
                    @click="confirmDelete(subscription)"
                    :title="$t('subscriptions.tooltip.del')"
                    :aria-label="$t('subscriptions.tooltip.del')"
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
    <Modal v-model="showModal" :title="modalTitle" size="md" show-close>
      <form @submit.prevent="saveSubscription">
        <div class="space-y-4">
          <Input
            v-model="formData.name"
            :label="$t('subscriptions.form.name')"
            :placeholder="$t('subscriptions.form.namePlaceholder')"
            required
            :error="formErrors.name"
          />

          <Input
            v-model="formData.url"
            :label="$t('subscriptions.form.url')"
            type="url"
            placeholder="https://example.com/subscription"
            required
            :error="formErrors.url"
          />

          <div class="flex items-center gap-4">
            <label class="flex items-center">
              <input
                v-model="formData.auto_update"
                type="checkbox"
                class="rounded border-gray-300 dark:border-gray-600 text-primary-600 focus:ring-primary-500 dark:bg-gray-700"
              />
              <span class="ml-2 text-sm text-gray-700 dark:text-gray-300">{{
                $t("subscriptions.form.autoUpdate")
              }}</span>
            </label>

            <Input
              v-model="formData.update_interval"
              :label="$t('subscriptions.form.updateInterval')"
              placeholder="24h"
              class="flex-1"
              :disabled="!formData.auto_update"
              :error="formErrors.update_interval"
              :hint="$t('subscriptions.form.updateIntervalHint')"
            />
          </div>

          <!-- Update method — for censored networks where direct fetch is
               DNS-poisoned or reset. -->
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              {{ $t("subscriptions.form.fetchMode") }}
            </label>
            <select
              v-model="formData.fetch_mode"
              class="w-full rounded-control border-gray-300 dark:border-gray-600 dark:bg-gray-700 text-sm focus:ring-primary-500 focus:border-primary-500"
            >
              <option value="">{{ $t("subscriptions.form.fetchModeDirect") }}</option>
              <option value="clean_dns">{{ $t("subscriptions.form.fetchModeCleanDns") }}</option>
              <option value="proxy">{{ $t("subscriptions.form.fetchModeProxy") }}</option>
            </select>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ $t("subscriptions.form.fetchModeHint") }}
            </p>
          </div>

          <Input
            v-if="formData.fetch_mode === 'proxy'"
            v-model="formData.proxy_url"
            :label="$t('subscriptions.form.proxyUrl')"
            placeholder="socks5://127.0.0.1:7893"
            :hint="$t('subscriptions.form.proxyUrlHint')"
          />
        </div>
      </form>

      <template #footer>
        <Button variant="secondary" @click="showModal = false">
          {{ $t("common.cancel") }}
        </Button>
        <Button
          :loading="isLoading"
          :disabled="!isFormValid"
          @click="saveSubscription"
        >
          {{ isEditing ? $t("common.update") : $t("common.add") }}
        </Button>
      </template>
    </Modal>

    <!-- Delete Confirmation Modal -->
    <Modal
      v-model="showDeleteConfirm"
      :title="$t('subscriptions.del.title')"
      size="sm"
      show-close
    >
      <div class="text-center">
        <div
          class="mx-auto flex items-center justify-center h-12 w-12 rounded-pill bg-red-100 dark:bg-red-900 mb-4"
        >
          <TrashIcon class="h-6 w-6 text-red-600 dark:text-red-400" />
        </div>
        <h3 class="text-lg font-medium text-gray-900 dark:text-gray-100 mb-2">
          {{ $t("subscriptions.del.confirmHeading") }}
        </h3>
        <p class="text-sm text-gray-500 dark:text-gray-400">
          {{ $t("subscriptions.del.confirmDesc") }}
        </p>
      </div>

      <template #footer>
        <Button variant="secondary" @click="showDeleteConfirm = false">
          {{ $t("common.cancel") }}
        </Button>
        <Button
          variant="danger"
          :loading="isLoading"
          @click="deleteSubscription"
        >
          {{ $t("common.delete") }}
        </Button>
      </template>
    </Modal>
  </div>
</template>

<style scoped>
/* Ensure proper rounded corners for table */
table thead tr:first-child th:first-child {
  border-top-left-radius: 0.5rem;
}

table thead tr:first-child th:last-child {
  border-top-right-radius: 0.5rem;
}

table tbody tr:last-child td:first-child {
  border-bottom-left-radius: 0.5rem;
}

table tbody tr:last-child td:last-child {
  border-bottom-right-radius: 0.5rem;
}
</style>
