<script setup lang="ts">
import { reactive } from 'vue'
import SubscriptionOrphans from '../../components/SubscriptionOrphans.vue'
import SubscriptionCollection from '../../components/SubscriptionCollection.vue'
import FieldHelp from '../../components/FieldHelp.vue'
import Button from "../../components/Button.vue";
import Input from "../../components/Input.vue";
import SubscriptionInfoKeywords from "../../components/SubscriptionInfoKeywords.vue";
import {
  PlusIcon,
  ArrowPathIcon,
  CloudArrowDownIcon,
  TagIcon,
} from "@heroicons/vue/24/outline";
import { Select } from "../../volt";
import { ContextPanel } from '../../volt'
import { useSubscriptionsPage } from '../../composables/useSubscriptionsPage'
const page = useSubscriptionsPage()
const collection = reactive(page)
const { subscriptions, isLoading, isUpdating, showModal, showInfoKeywords, formData, fetchModeOptions, formErrors, isEditing, modalTitle, isFormValid, loadSubscriptions, openAddModal, saveSubscription, updateAllSubscriptions, } = page
</script>
<template>
  <div>
    <!-- Header: subtitle + actions -->
    <div class="flex flex-wrap items-center justify-between gap-3 mb-4">
      <p class="text-gray-500 dark:text-gray-400">
        {{ $t("subscriptions.subtitle") }}
      </p>
      <!--
        These two read as the same button until you know the app: both were
        `variant="secondary"` with an `ArrowPathIcon` and near-synonymous
        labels, but they do very different things.

          Reload list  — re-reads what this panel already has. Local, read-only,
                         instant. Circular arrow = "refresh the view", and it is
                         the quietest control here.
          Update all   — walks every subscription, fetches the provider's feed
                         and re-imports nodes. Network, mutating, slow. It shares
                         `CloudArrowDownIcon` with the per-row update button,
                         because it is that same operation applied to every row.
      -->
      <div class="flex flex-wrap items-center gap-2">
        <Button
          variant="ghost"
          :loading="isLoading && !isUpdating.length"
          @click="loadSubscriptions"
          :disabled="isLoading"
          :title="$t('subscriptions.tooltip.refreshList')"
        >
          <ArrowPathIcon class="h-4 w-4" />
          {{ $t("subscriptions.refresh") }}
        </Button>
        <Button
          v-if="subscriptions.length > 0"
          variant="secondary"
          :loading="isUpdating.length === subscriptions.length"
          @click="updateAllSubscriptions"
          :disabled="isUpdating.length > 0"
          :title="$t('subscriptions.tooltip.updateAll')"
        >
          <CloudArrowDownIcon class="h-4 w-4" />
          {{ $t("subscriptions.updateAll") }}
        </Button>
        <Button
          variant="secondary"
          @click="showInfoKeywords = true"
          :title="$t('subscriptions.keywords.tooltip')"
        >
          <TagIcon class="h-4 w-4" />
          {{ $t("subscriptions.keywords.button") }}
        </Button>
        <Button @click="openAddModal">
          <PlusIcon class="h-4 w-4" />
          {{ $t("subscriptions.add") }}
        </Button>
      </div>
    </div>

    <SubscriptionOrphans :refresh-key="subscriptions.map(s => s.id + s.last_update).join('|') + isLoading + isUpdating.join('|')" @cleaned="loadSubscriptions" />
    <SubscriptionCollection :page="collection" />

    <!-- Info-label keywords editor -->
    <SubscriptionInfoKeywords v-model="showInfoKeywords" />

    <!-- Add/Edit Modal -->
    <ContextPanel v-model:visible="showModal" :header="modalTitle" modal class="w-full max-w-lg">
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

          <!--
            Optional, and usually filled in for you: a refresh takes the site
            from the feed when the provider reports one. Editing it here pins
            it — later refreshes leave a non-empty value alone.
          -->
          <Input
            v-model="formData.official_url"
            :label="$t('subscriptions.form.officialUrl')"
            type="url"
            placeholder="https://example.com"
            :error="formErrors.official_url"
            :hint="$t('subscriptions.form.officialUrlHint')"
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
              v-if="formData.auto_update"
              v-model="formData.update_interval"
              :label="$t('subscriptions.form.updateInterval')"
              placeholder="24h"
              class="flex-1"
              :error="formErrors.update_interval"
              :hint="$t('subscriptions.form.updateIntervalHint')"
            />
          </div>

          <!-- Update method — for censored networks where direct fetch is
               DNS-poisoned or reset. -->
          <div>
            <div class="flex items-center gap-1"><label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              {{ $t("subscriptions.form.fetchMode") }}
            </label><FieldHelp :label="$t('subscriptions.form.fetchMode')" :text="$t('subscriptions.form.fetchModeHint')" /></div>
            <Select
              class="w-full"
              v-model="formData.fetch_mode"
              :options="fetchModeOptions"
              optionLabel="label"
              optionValue="value"
              :placeholder="$t('subscriptions.form.fetchModeDirect')"
            />

          </div>

          <Input
            v-if="formData.fetch_mode === 'proxy'"
            v-model="formData.proxy_url"
            :label="$t('subscriptions.form.proxyUrl')"
            placeholder="socks5://127.0.0.1:7893"
            :hint="$t('subscriptions.form.proxyUrlHint')"
          />

          <!--
            Quality probing. The URL is per-subscription because a provider
            whose nodes cannot reach Google reads as 0% available against
            gstatic while working perfectly — the operator has to be able to
            point the test somewhere the feed can actually get to.
          -->
          <div class="border-t border-gray-100 pt-4 dark:border-gray-700">
            <div class="flex items-center gap-1"><label class="flex items-center">
              <input
                v-model="formData.probe_enabled"
                type="checkbox"
                class="rounded border-gray-300 dark:border-gray-600 text-primary-600 focus:ring-primary-500 dark:bg-gray-700"
              />
              <span class="ml-2 text-sm text-gray-700 dark:text-gray-300">
                {{ $t("subProbe.form.enabled") }}
              </span>
            </label><FieldHelp :label="$t('subProbe.form.enabled')" :text="$t('subProbe.form.enabledHint')" /></div>


            <Input
              v-if="formData.probe_enabled"
              v-model="formData.probe_url"
              class="mt-3"
              :label="$t('subProbe.form.url')"
              placeholder="https://www.gstatic.com/generate_204"
              :error="formErrors.probe_url"
              :hint="$t('subProbe.form.urlHint')"
            />
          </div>
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
    </ContextPanel>

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
