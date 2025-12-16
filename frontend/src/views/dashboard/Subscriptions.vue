<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { apiService } from '../../services/api'
import { SubscriptionService } from '../../services/subscription'
import { Code, type Subscription, type Outbound } from '../../types/api'
import Button from '../../components/Button.vue'
import Modal from '../../components/Modal.vue'
import Input from '../../components/Input.vue'
import Badge from '../../components/Badge.vue'
import Notification from '../../components/Notification.vue'
import {
  PlusIcon,
  PencilIcon,
  TrashIcon,
  ArrowPathIcon,
  ServerIcon,
  CheckCircleIcon,
  XCircleIcon,
  ClockIcon
} from '@heroicons/vue/24/outline'

const subscriptionService = new SubscriptionService(apiService)

interface NotificationMessage {
  id: string
  type: 'success' | 'error' | 'info'
  message: string
  duration?: number
}

const subscriptions = ref<Subscription[]>([])
const isLoading = ref(false)
const isUpdating = ref<string[]>([])
const showModal = ref(false)
const editingSubscription = ref<Subscription | null>(null)
const showDeleteConfirm = ref(false)
const deletingSubscriptionId = ref<string>('')
const notifications = ref<NotificationMessage[]>([])

// Form data
const formData = ref({
  name: '',
  url: '',
  auto_update: true,
  update_interval: '24h'
})

// Form errors
const formErrors = ref<Record<string, string>>({})

// Computed properties
const isEditing = computed(() => !!editingSubscription.value)
const modalTitle = computed(() => isEditing.value ? 'Edit Subscription' : 'Add Subscription')
const isFormValid = computed(() => {
  return formData.value.name.trim() !== '' && formData.value.url.trim() !== ''
})

// Methods
const showNotification = (type: 'success' | 'error' | 'info', message: string, duration = 5000) => {
  const id = Date.now().toString()
  notifications.value.push({ id, type, message, duration })

  if (duration > 0) {
    setTimeout(() => {
      removeNotification(id)
    }, duration)
  }
}

const removeNotification = (id: string) => {
  const index = notifications.value.findIndex(n => n.id === id)
  if (index > -1) {
    notifications.value.splice(index, 1)
  }
}

const loadSubscriptions = async () => {
  try {
    isLoading.value = true
    const response = await subscriptionService.getSubscriptions()

    if (response.code === Code.Success) {
      subscriptions.value = response.data.subscriptions || []
    } else {
      showNotification('error', response.msg || 'Failed to load subscriptions')
    }
  } catch (error) {
    console.error('Error loading subscriptions:', error)
    showNotification('error', 'Error loading subscriptions')
  } finally {
    isLoading.value = false
  }
}

const resetForm = () => {
  formData.value = {
    name: '',
    url: '',
    auto_update: true,
    update_interval: '24h'
  }
  formErrors.value = {}
  editingSubscription.value = null
}

const openAddModal = () => {
  resetForm()
  showModal.value = true
}

const openEditModal = (subscription: Subscription) => {
  formData.value = {
    name: subscription.name,
    url: subscription.url,
    auto_update: subscription.enabled,
    update_interval: `${subscription.update_interval}h`
  }
  editingSubscription.value = subscription
  showModal.value = true
}

const validateForm = () => {
  const errors: Record<string, string> = {}

  if (!formData.value.name.trim()) {
    errors.name = 'Name is required'
  }

  if (!formData.value.url.trim()) {
    errors.url = 'URL is required'
  } else {
    try {
      new URL(formData.value.url)
    } catch {
      errors.url = 'Invalid URL format'
    }
  }

  formErrors.value = errors
  return Object.keys(errors).length === 0
}

const saveSubscription = async () => {
  if (!validateForm()) {
    return
  }

  try {
    const subscriptionData = {
      name: formData.value.name,
      url: formData.value.url,
      auto_update: formData.value.auto_update,
      update_interval: formData.value.update_interval
    }

    let response
    if (isEditing.value && editingSubscription.value) {
      response = await subscriptionService.updateSubscription(editingSubscription.value.id, subscriptionData)
    } else {
      response = await subscriptionService.addSubscription(subscriptionData)
    }

    if (response.code === Code.Success) {
      showNotification('success', response.data.message || 'Subscription saved successfully')
      showModal.value = false
      resetForm()
      await loadSubscriptions()
    } else {
      showNotification('error', response.msg || 'Failed to save subscription')
    }
  } catch (error) {
    console.error('Error saving subscription:', error)
    showNotification('error', 'Error saving subscription')
  }
}

const confirmDelete = (subscription: Subscription) => {
  deletingSubscriptionId.value = subscription.id
  showDeleteConfirm.value = true
}

const deleteSubscription = async () => {
  if (!deletingSubscriptionId.value) return

  try {
    const response = await subscriptionService.deleteSubscription(deletingSubscriptionId.value)

    if (response.code === Code.Success) {
      showNotification('success', response.data.message || 'Subscription deleted successfully')
      showDeleteConfirm.value = false
      deletingSubscriptionId.value = ''
      await loadSubscriptions()
    } else {
      showNotification('error', response.msg || 'Failed to delete subscription')
    }
  } catch (error) {
    console.error('Error deleting subscription:', error)
    showNotification('error', 'Error deleting subscription')
  }
}

const updateSubscription = async (subscription: Subscription) => {
  if (isUpdating.value.includes(subscription.id)) return

  try {
    isUpdating.value.push(subscription.id)
    const response = await subscriptionService.updateSubscriptionContent(subscription.id)

    if (response.code === Code.Success) {
      showNotification('success', `Updated: ${response.data.node_count} nodes found`)
      await loadSubscriptions()
    } else {
      showNotification('error', response.msg || 'Failed to update subscription')
    }
  } catch (error) {
    console.error('Error updating subscription:', error)
    showNotification('error', 'Error updating subscription')
  } finally {
    const index = isUpdating.value.indexOf(subscription.id)
    if (index > -1) {
      isUpdating.value.splice(index, 1)
    }
  }
}

const updateAllSubscriptions = async () => {
  for (const subscription of subscriptions.value) {
    await updateSubscription(subscription)
    // Small delay between requests to avoid overwhelming the server
    await new Promise(resolve => setTimeout(resolve, 1000))
  }
}

const formatDate = (dateString?: string) => {
  if (!dateString) return 'Never'
  return new Date(dateString).toLocaleString()
}

const getIntervalHours = (interval: string) => {
  const match = interval.match(/(\d+)h/)
  return match ? parseInt(match[1]) : 24
}

const getStatusBadge = (subscription: Subscription) => {
  if (subscription.node_count === 0) {
    return { type: 'danger' as const, icon: XCircleIcon, text: 'Empty' }
  }

  if (!subscription.last_update) {
    return { type: 'warning' as const, icon: ClockIcon, text: 'Not Updated' }
  }

  const lastUpdate = new Date(subscription.last_update)
  const hoursSinceUpdate = (Date.now() - lastUpdate.getTime()) / (1000 * 60 * 60)
  const intervalHours = getIntervalHours(subscription.update_interval)

  if (hoursSinceUpdate > intervalHours * 1.5) {
    return { type: 'warning' as const, icon: ClockIcon, text: 'Outdated' }
  }

  return { type: 'success' as const, icon: CheckCircleIcon, text: 'Updated' }
}

// Lifecycle
onMounted(() => {
  loadSubscriptions()
})
</script>

<template>
  <div class="p-8">
    <!-- Header -->
    <div class="flex items-center justify-between mb-6">
      <div>
        <h2 class="text-3xl font-bold text-gray-900 dark:text-gray-100">Subscriptions</h2>
        <p class="text-gray-500 dark:text-gray-400 mt-1">Manage and update node subscriptions</p>
      </div>
      <div class="flex gap-3">
        <Button
          variant="secondary"
          :loading="isLoading && !isUpdating.length"
          @click="loadSubscriptions"
          :disabled="isLoading"
        >
          <ArrowPathIcon class="h-5 w-5" />
          Refresh
        </Button>
        <Button
          v-if="subscriptions.length > 0"
          variant="secondary"
          :loading="isUpdating.length === subscriptions.length"
          @click="updateAllSubscriptions"
          :disabled="isUpdating.length > 0"
        >
          <ArrowPathIcon class="h-5 w-5" />
          Update All
        </Button>
        <Button @click="openAddModal">
          <PlusIcon class="h-5 w-5" />
          Add Subscription
        </Button>
      </div>
    </div>

    <!-- Subscriptions List -->
    <div class="bg-white dark:bg-slate-800 rounded-lg shadow">
      <div v-if="isLoading && subscriptions.length === 0" class="p-12 text-center">
        <div class="inline-flex items-center justify-center w-16 h-16 bg-blue-100 dark:bg-blue-900 rounded-full mb-4">
          <ServerIcon class="h-8 w-8 text-blue-600 dark:text-blue-400" />
        </div>
        <p class="text-gray-500 dark:text-gray-400">Loading subscriptions...</p>
      </div>

      <div v-else-if="subscriptions.length === 0" class="p-12 text-center">
        <div class="inline-flex items-center justify-center w-16 h-16 bg-gray-100 dark:bg-gray-700 rounded-full mb-4">
          <ServerIcon class="h-8 w-8 text-gray-400" />
        </div>
        <h3 class="text-lg font-medium text-gray-900 dark:text-gray-100 mb-2">No subscriptions</h3>
        <p class="text-gray-500 dark:text-gray-400 mb-6">Get started by adding your first subscription</p>
        <Button @click="openAddModal">
          <PlusIcon class="h-5 w-5" />
          Add Subscription
        </Button>
      </div>

      <div v-else class="overflow-x-auto">
        <table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
          <thead class="bg-gray-50 dark:bg-gray-700">
            <tr>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                Name
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                URL
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                Status
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                Nodes
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                Last Update
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                Auto Update
              </th>
              <th class="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                Actions
              </th>
            </tr>
          </thead>
          <tbody class="bg-white dark:bg-slate-800 divide-y divide-gray-200 dark:divide-gray-700">
            <tr v-for="subscription in subscriptions" :key="subscription.id" class="hover:bg-gray-50 dark:hover:bg-gray-700">
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="text-sm font-medium text-gray-900 dark:text-gray-100">
                  {{ subscription.name }}
                </div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="text-sm text-gray-500 dark:text-gray-400 truncate max-w-xs">
                  {{ subscription.url }}
                </div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <Badge
                  :type="getStatusBadge(subscription).type"
                  class="inline-flex items-center gap-1"
                >
                  <component :is="getStatusBadge(subscription).icon" class="h-3 w-3" />
                  {{ getStatusBadge(subscription).text }}
                </Badge>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="text-sm text-gray-900 dark:text-gray-100">
                  {{ subscription.node_count || 0 }}
                </div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="text-sm text-gray-500 dark:text-gray-400">
                  {{ formatDate(subscription.last_update) }}
                </div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <Badge :type="subscription.enabled ? 'success' : 'secondary'">
                  {{ subscription.enabled ? 'Enabled' : 'Disabled' }}
                </Badge>
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                <div class="flex items-center justify-end gap-2">
                  <Button
                    variant="ghost"
                    size="sm"
                    :loading="isUpdating.includes(subscription.id)"
                    @click="updateSubscription(subscription)"
                    :disabled="isUpdating.includes(subscription.id)"
                    title="Update subscription"
                  >
                    <ArrowPathIcon class="h-4 w-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    @click="openEditModal(subscription)"
                    title="Edit subscription"
                  >
                    <PencilIcon class="h-4 w-4" />
                  </Button>
                  <Button
                    variant="danger"
                    size="sm"
                    @click="confirmDelete(subscription)"
                    title="Delete subscription"
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
    <Modal
      v-model="showModal"
      :title="modalTitle"
      size="md"
      show-close
    >
      <form @submit.prevent="saveSubscription">
        <div class="space-y-4">
          <Input
            v-model="formData.name"
            label="Name"
            placeholder="Enter subscription name"
            required
            :error="formErrors.name"
          />

          <Input
            v-model="formData.url"
            label="Subscription URL"
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
                class="rounded border-gray-300 dark:border-gray-600 text-blue-600 focus:ring-blue-500 dark:bg-gray-700"
              />
              <span class="ml-2 text-sm text-gray-700 dark:text-gray-300">Auto Update</span>
            </label>

            <Input
              v-model="formData.update_interval"
              label="Update Interval"
              placeholder="24h"
              class="flex-1"
              :disabled="!formData.auto_update"
            />
          </div>
        </div>
      </form>

      <template #footer>
        <Button
          variant="secondary"
          @click="showModal = false"
        >
          Cancel
        </Button>
        <Button
          :loading="isLoading"
          :disabled="!isFormValid"
          @click="saveSubscription"
        >
          {{ isEditing ? 'Update' : 'Add' }}
        </Button>
      </template>
    </Modal>

    <!-- Delete Confirmation Modal -->
    <Modal
      v-model="showDeleteConfirm"
      title="Delete Subscription"
      size="sm"
      show-close
    >
      <div class="text-center">
        <div class="mx-auto flex items-center justify-center h-12 w-12 rounded-full bg-red-100 dark:bg-red-900 mb-4">
          <TrashIcon class="h-6 w-6 text-red-600 dark:text-red-400" />
        </div>
        <h3 class="text-lg font-medium text-gray-900 dark:text-gray-100 mb-2">
          Delete subscription?
        </h3>
        <p class="text-sm text-gray-500 dark:text-gray-400">
          This action cannot be undone. The subscription and all its nodes will be permanently removed.
        </p>
      </div>

      <template #footer>
        <Button
          variant="secondary"
          @click="showDeleteConfirm = false"
        >
          Cancel
        </Button>
        <Button
          variant="danger"
          :loading="isLoading"
          @click="deleteSubscription"
        >
          Delete
        </Button>
      </template>
    </Modal>

    <!-- Notifications -->
    <div class="fixed bottom-4 right-4 z-50 space-y-2">
      <Notification
        v-for="notification in notifications"
        :key="notification.id"
        :type="notification.type"
        :message="notification.message"
        @close="removeNotification(notification.id)"
      />
    </div>
  </div>
</template>
