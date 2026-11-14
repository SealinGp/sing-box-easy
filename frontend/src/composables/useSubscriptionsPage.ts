import { ref, onMounted, computed } from "vue";
import { useI18n } from "vue-i18n";
import { apiService } from "../services/api";
import { SubscriptionService } from "../services/subscription";
import { type Subscription } from "../types/api";
import { useNotify } from "../composables/useNotify";
import { parseDurationToHours, isValidDuration } from "../plugins/dayjs";
import { subscriptionHealth } from "../utils/subscriptionHealth";
import {
  summarizeUpdate,
  updateSubscriptionsSequentially,
} from "../utils/subscriptionUpdate";
import { isLinkableSiteInput, safeExternalUrl } from "../utils/safeExternalUrl";
import { summarizePlan } from "../utils/subscriptionInfo";
import { subProbeService } from "../services";
import type { ProbePoint } from "../types/subprobe";

export function useSubscriptionsPage() {
  const subscriptionService = new SubscriptionService(apiService);
  const { t } = useI18n();
  const notify = useNotify();

  interface FormData {
    name: string;
    url: string;
    official_url: string;
    auto_update: boolean;
    update_interval: string;
    fetch_mode: "" | "clean_dns" | "proxy";
    proxy_url: string;
    probe_enabled: boolean;
    probe_url: string;
  }

  const subscriptions = ref<Subscription[]>([]);
  const isLoading = ref(false);
  const isUpdating = ref<string[]>([]);
  const showModal = ref(false);
  const editingSubscription = ref<Subscription | null>(null);
  // Editor for the labels that mark a feed entry as account metadata rather than
  // a proxy node — provider-specific, hence operator-editable.
  const showInfoKeywords = ref(false);

  // Form data
  const formData = ref<FormData>({
    name: "",
    url: "",
    official_url: "",
    auto_update: true,
    update_interval: "24h",
    fetch_mode: "",
    proxy_url: "",
    // On by default for a new subscription: the quality chart is the point of
    // the feature, and an opt-in metric leaves it empty on a fresh install.
    probe_enabled: true,
    probe_url: "",
  });

  const fetchModeOptions = computed(() => [
    { value: "", label: t("subscriptions.form.fetchModeDirect") },
    { value: "clean_dns", label: t("subscriptions.form.fetchModeCleanDns") },
    { value: "proxy", label: t("subscriptions.form.fetchModeProxy") },
  ]);

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
      official_url: "",
      auto_update: true,
      update_interval: "24h",
      fetch_mode: "",
      proxy_url: "",
      probe_enabled: true,
      probe_url: "",
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
      official_url: subscription.official_url ?? "",
      auto_update: subscription.auto_update ?? false,
      update_interval: subscription.update_interval, // Already a string from backend (e.g., "24h")
      fetch_mode: subscription.fetch_mode ?? "",
      proxy_url: subscription.proxy_url ?? "",
      probe_enabled: subscription.probe_enabled ?? true,
      probe_url: subscription.probe_url ?? "",
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

    // The official site ends up in an href, so the same http(s)-only rule the
    // backend enforces on save is applied here — the operator gets the error at
    // the field instead of a rejected request. A bare domain is accepted, since
    // that is what the backend stores after promoting it to https.
    if (
      formData.value.official_url.trim() &&
      !isLinkableSiteInput(formData.value.official_url)
    ) {
      errors.official_url = t("subscriptions.validation.officialUrlScheme");
    }

    // The latency target must be https. sing-box silently discards an http://
    // delay-test URL and substitutes its own default, so an accepted http value
    // would report latency for an endpoint the operator never chose. The backend
    // enforces this too; checking here puts the error on the field.
    if (formData.value.probe_url.trim()) {
      try {
        const parsed = new URL(formData.value.probe_url.trim());
        if (parsed.protocol !== "https:") {
          errors.probe_url = t("subProbe.validation.urlScheme");
        }
      } catch {
        errors.probe_url = t("subProbe.validation.urlInvalid");
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
        official_url: formData.value.official_url.trim(),
        auto_update: formData.value.auto_update,
        update_interval: formData.value.update_interval, // Send as string
        fetch_mode: formData.value.fetch_mode,
        // proxy_url only matters in proxy mode; send empty otherwise.
        proxy_url:
          formData.value.fetch_mode === "proxy"
            ? formData.value.proxy_url.trim()
            : "",
        probe_enabled: formData.value.probe_enabled,
        probe_url: formData.value.probe_url.trim(),
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

  /**
   * The provider's site, or null when this subscription has none (or has one
   * that is not safe to link). Drives whether the name renders as a link.
   */
  const officialLink = (subscription: Subscription) =>
    safeExternalUrl(subscription.official_url);

  // Confirmation already happened in the row's <PopConfirm>, which names the
  // subscription being removed — the centre-screen modal it replaced could not,
  // so "Delete subscription?" gave no way to check WHICH row was clicked.
  const deleteSubscription = async (subscription: Subscription) => {
    try {
      const response = await subscriptionService.deleteSubscription(
        subscription.id
      );
      notify.success(
        response.data.message || t("subscriptions.notify.deletedOk")
      );
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

      // The backend does not report success until sing-box has restarted with
      // the refreshed config, so the nodes shown here and the running service
      // cannot silently drift apart.
      // The same builder the Overview card uses, so one backend result cannot be
      // described two different ways depending on which screen you are on.
      const summary = summarizeUpdate(response.data, t);
      notify.success(
        t("subscriptions.notify.synced", { name: subscription.name, summary })
      );
      await loadSubscriptions();
      // The refresh may have added or removed nodes, which changes the
      // denominator the column shows.
      await loadProbeStatus();
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
    await updateSubscriptionsSequentially(
      subscriptions.value,
      updateSubscription,
    );
  };

  const formatDate = (dateString?: string) => {
    if (!dateString) return t("common.never");
    return new Date(dateString).toLocaleString();
  };

  // The freshness rule itself lives in utils/subscriptionHealth so this page and
  // the Overview card cannot drift into disagreeing about the same subscription;
  // only the presentation is decided here.
  const getSubscriptionStatus = (subscription: Subscription) => {
    switch (subscriptionHealth(subscription)) {
      case "never":
        return {
          type: "warning" as const,
          text: t("subscriptions.status.notUpdated"),
        };
      case "outdated":
        return {
          type: "warning" as const,
          text: t("subscriptions.status.outdated"),
        };
      default:
        return {
          type: "success" as const,
          text: t("subscriptions.status.updated"),
        };
    }
  };

  /**
   * Newest quality sample per subscription, keyed by id.
   *
   * One call for every row rather than one per row: the summary is a column, and
   * N round trips to draw a column is N-1 too many.
   */
  const probeLatest = ref<Record<string, ProbePoint>>({});

  const loadProbeStatus = async () => {
    try {
      const { data } = await subProbeService.getStatus();
      probeLatest.value = data.latest ?? {};
    } catch {
      // Quality is supplementary to this page: a subscription list that fails to
      // render because the prober is unreachable would be a worse page than one
      // with an empty column. The dialog reports the error when it is opened.
      probeLatest.value = {};
    }
  };

  // Lifecycle
  onMounted(() => {
    loadSubscriptions();
    loadProbeStatus();
  });
  return { summarizePlan, t, subscriptions, isLoading, isUpdating, showModal, showInfoKeywords, formData, fetchModeOptions, formErrors, isEditing, modalTitle, isFormValid, loadSubscriptions, openAddModal, openEditModal, saveSubscription, officialLink, deleteSubscription, updateSubscription, updateAllSubscriptions, formatDate, getSubscriptionStatus, probeLatest, loadProbeStatus }
}
