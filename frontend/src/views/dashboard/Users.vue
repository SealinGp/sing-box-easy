<script setup lang="ts">
import { ref, onMounted, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { User } from '../../types/api'
import Button from '../../components/Button.vue'
import Input from '../../components/Input.vue'
import Badge from '../../components/Badge.vue'
import Modal from '../../components/Modal.vue'
import { PlusIcon, PencilIcon, TrashIcon } from '@heroicons/vue/24/outline'
import { userService } from '../../services'
import { useToast } from 'primevue/usetoast'
import { Select } from '../../volt'

const users = ref<User[]>([])
const loading = ref(false)
const toast = useToast()
const { t } = useI18n()

// Modal state
const showModal = ref(false)
const isEditMode = ref(false)
const currentUserId = ref<number | null>(null)
const currentUser = ref<{
  username: string
  password?: string
  role: 'admin' | 'viewer'
}>({
  username: '',
  password: '',
  role: 'viewer',
})

const roleOptions = computed(() => [
  { value: 'admin', label: t('users.form.roles.admin') },
  { value: 'viewer', label: t('users.form.roles.viewer') },
])

// Delete confirmation
const showDeleteConfirm = ref(false)
const deletingUser = ref<User | null>(null)

const fetchUsers = async () => {
  loading.value = true
  try {
    const data = await userService.listUsers()
    users.value = data || []
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: err.message || t('users.toast.fetchFailed'),
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

const openAddModal = () => {
  isEditMode.value = false
  currentUserId.value = null
  currentUser.value = {
    username: '',
    password: '',
    role: 'viewer',
  }
  showModal.value = true
}

const openEditModal = (user: User) => {
  isEditMode.value = true
  currentUserId.value = user.id
  currentUser.value = {
    username: user.username,
    password: '',
    role: user.role,
  }
  showModal.value = true
}

const closeModal = () => {
  showModal.value = false
}

const handleSave = async () => {
  // Validation
  if (!currentUser.value.username?.trim()) {
    toast.add({
      severity: 'error',
      summary: t('users.validation.title'),
      detail: t('users.validation.usernameRequired'),
      life: 3000
    })
    return
  }

  if (!isEditMode.value && !currentUser.value.password?.trim()) {
    toast.add({
      severity: 'error',
      summary: t('users.validation.title'),
      detail: t('users.validation.passwordRequired'),
      life: 3000
    })
    return
  }

  loading.value = true
  try {
    const payload: any = {
      username: currentUser.value.username,
      role: currentUser.value.role,
    }
    if (currentUser.value.password) {
      payload.password = currentUser.value.password
    }

    if (isEditMode.value && currentUserId.value !== null) {
      await userService.updateUser(currentUserId.value, payload)
      toast.add({
        severity: 'success',
        summary: t('common.success'),
        detail: t('users.toast.updatedOk'),
        life: 3000
      })
    } else {
      await userService.createUser(payload)
      toast.add({
        severity: 'success',
        summary: t('common.success'),
        detail: t('users.toast.addedOk'),
        life: 3000
      })
    }
    closeModal()
    await fetchUsers()
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: err.message || t('users.toast.saveFailed'),
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

const openDeleteConfirm = (user: User) => {
  deletingUser.value = user
  showDeleteConfirm.value = true
}

const closeDeleteConfirm = () => {
  showDeleteConfirm.value = false
  deletingUser.value = null
}

const handleDelete = async () => {
  if (!deletingUser.value) return

  loading.value = true
  try {
    await userService.deleteUser(deletingUser.value.id)
    toast.add({
      severity: 'success',
      summary: t('common.success'),
      detail: t('users.toast.deletedOk'),
      life: 3000
    })
    closeDeleteConfirm()
    await fetchUsers()
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: err.message || t('users.toast.deleteFailed'),
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

// Watch for modal close to reset form state
watch(showModal, (newValue) => {
  if (!newValue) {
    setTimeout(() => {
      currentUser.value = {
        username: '',
        password: '',
        role: 'viewer',
      }
    }, 300)
  }
})

watch(showDeleteConfirm, (newValue) => {
  if (!newValue) {
    setTimeout(() => {
      deletingUser.value = null
    }, 300)
  }
})

onMounted(fetchUsers)
</script>

<template>
  <div class="p-8">
    <div class="flex justify-between items-center mb-6">
      <h2 class="text-3xl font-bold text-gray-900 dark:text-gray-100">{{ $t('users.title') }}</h2>
      <Button @click="openAddModal" variant="primary">
        <PlusIcon class="h-5 w-5 mr-2" />
        {{ $t('users.add') }}
      </Button>
    </div>

    <div class="bg-white dark:bg-slate-800 rounded-surface shadow dark:shadow-float dark:shadow-slate-700/50 overflow-hidden">
      <div v-if="loading && users.length === 0" class="flex items-center justify-center py-12">
        <div class="animate-spin rounded-pill h-8 w-8 border-b-2 border-primary-600"></div>
      </div>

      <div v-else-if="users.length === 0" class="text-center py-12">
        <p class="text-gray-500 dark:text-gray-500 mb-4">{{ $t('users.empty') }}</p>
        <Button @click="openAddModal" variant="primary" size="sm">
          <PlusIcon class="h-4 w-4 mr-2" />
          {{ $t('users.addFirst') }}
        </Button>
      </div>

      <div v-else class="overflow-x-auto">
        <table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
          <thead class="bg-gray-50 dark:bg-gray-700">
            <tr>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">{{ $t('users.table.username') }}</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">{{ $t('users.table.role') }}</th>
              <th class="px-4 py-2 text-right text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">{{ $t('users.table.actions') }}</th>
            </tr>
          </thead>
          <tbody class="bg-white dark:bg-slate-800 divide-y divide-gray-200 dark:divide-gray-700">
            <tr v-for="user in users" :key="user.id" class="hover:bg-gray-50 dark:hover:bg-gray-700">
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="text-sm font-medium text-gray-900 dark:text-gray-100">{{ user.username }}</div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <Badge :variant="user.role === 'admin' ? 'success' : 'info'">
                  {{ $t(`users.form.roles.${user.role}`) }}
                </Badge>
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                <div class="user-table-actions flex items-center justify-end gap-2">
                  <Button @click="openEditModal(user)" variant="ghost" size="sm" action>
                    <PencilIcon class="h-4 w-4" />
                  </Button>
                  <Button @click="openDeleteConfirm(user)" variant="ghost" size="sm" action class="text-red-600 dark:text-red-400 hover:text-red-700 dark:hover:text-red-300">
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
      :model-value="showModal"
      @update:model-value="(v: boolean) => { if (!v) closeModal() }"
      :title="isEditMode ? $t('users.modal.edit') : $t('users.modal.add')"
      size="md"
      show-close
    >
      <div class="space-y-4">
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('users.form.username') }}</label>
          <Input
            v-model="currentUser.username"
            :placeholder="$t('users.form.usernamePlaceholder')"
          />
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('users.form.password') }}</label>
          <Input
            v-model="currentUser.password"
            type="password"
            :placeholder="isEditMode ? $t('users.form.passwordPlaceholder') : $t('users.form.passwordRequiredPlaceholder')"
          />
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('users.form.role') }}</label>
          <Select
            class="w-full"
            v-model="currentUser.role"
            :options="roleOptions"
            optionLabel="label"
            optionValue="value"
          />
        </div>
      </div>

      <template #footer>
        <Button @click="closeModal" variant="secondary">{{ $t('common.cancel') }}</Button>
        <Button @click="handleSave" variant="primary" :disabled="loading">
          {{ isEditMode ? $t('common.update') : $t('common.add') }}
        </Button>
      </template>
    </Modal>

    <!-- Delete Confirmation Modal -->
    <Modal
      :model-value="showDeleteConfirm"
      @update:model-value="(v: boolean) => { if (!v) closeDeleteConfirm() }"
      :title="$t('users.del.title')"
      size="sm"
      show-close
    >
      <p class="text-gray-700 dark:text-gray-300">
        {{ $t('users.del.confirm', { username: deletingUser?.username }) }}
      </p>

      <template #footer>
        <Button @click="closeDeleteConfirm" variant="secondary">{{ $t('common.cancel') }}</Button>
        <Button @click="handleDelete" variant="danger" :disabled="loading">
          {{ $t('common.delete') }}
        </Button>
      </template>
    </Modal>
  </div>
</template>
