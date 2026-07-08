<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { userService } from '../../services'
import type { User } from '../../types/api'
import {
  UserIcon,
  ShieldCheckIcon,
  PlusIcon,
  TrashIcon,
  PencilIcon,
  CalendarIcon,
  SparklesIcon
} from '@heroicons/vue/24/outline'

const currentUser = ref<User | null>(null)
const users = ref<User[]>([])
const activeTab = ref<'profile' | 'users'>('profile')

// Loading states
const profileLoading = ref(false)
const usersLoading = ref(false)

// Messages
const successMsg = ref('')
const errorMsg = ref('')

// Profile Form
const profileUsername = ref('')
const profilePassword = ref('')
const profileConfirmPassword = ref('')

// Admin Create/Edit Forms
const showAddUserModal = ref(false)
const showEditUserModal = ref(false)

const newUserUsername = ref('')
const newUserPassword = ref('')
const newUserRole = ref<'admin' | 'viewer'>('viewer')

const editingUser = ref<User | null>(null)
const editingUsername = ref('')
const editingPassword = ref('')
const editingRole = ref<'admin' | 'viewer'>('viewer')

const fetchProfile = async () => {
  profileLoading.value = true
  try {
    const user = await userService.getMe()
    currentUser.value = user
    profileUsername.value = user.username
  } catch (err: any) {
    errorMsg.value = err.message || 'Failed to load profile'
  } finally {
    profileLoading.value = false
  }
}

const fetchUsers = async () => {
  if (currentUser.value?.role !== 'admin') return
  usersLoading.value = true
  try {
    users.value = await userService.listUsers()
  } catch (err: any) {
    errorMsg.value = err.message || 'Failed to list users'
  } finally {
    usersLoading.value = false
  }
}

const handleUpdateProfile = async () => {
  if (!currentUser.value) return
  if (profilePassword.value && profilePassword.value !== profileConfirmPassword.value) {
    errorMsg.value = 'Passwords do not match'
    return
  }

  successMsg.value = ''
  errorMsg.value = ''
  profileLoading.value = true

  try {
    const updated = await userService.updateUser(currentUser.value.id, {
      username: profileUsername.value,
      password: profilePassword.value || undefined
    })
    currentUser.value = updated
    profilePassword.value = ''
    profileConfirmPassword.value = ''
    successMsg.value = 'Profile updated successfully'
    setTimeout(() => { successMsg.value = '' }, 3000)
  } catch (err: any) {
    errorMsg.value = err.message || 'Failed to update profile'
  } finally {
    profileLoading.value = false
  }
}

const handleAddUser = async () => {
  if (!newUserUsername.value || !newUserPassword.value) {
    errorMsg.value = 'Username and password are required'
    return
  }

  successMsg.value = ''
  errorMsg.value = ''
  usersLoading.value = true

  try {
    await userService.createUser({
      username: newUserUsername.value,
      password: newUserPassword.value,
      role: newUserRole.value
    })
    successMsg.value = `User '${newUserUsername.value}' created successfully`
    newUserUsername.value = ''
    newUserPassword.value = ''
    newUserRole.value = 'viewer'
    showAddUserModal.value = false
    await fetchUsers()
    setTimeout(() => { successMsg.value = '' }, 3000)
  } catch (err: any) {
    errorMsg.value = err.message || 'Failed to create user'
  } finally {
    usersLoading.value = false
  }
}

const startEditUser = (user: User) => {
  editingUser.value = user
  editingUsername.value = user.username
  editingPassword.value = ''
  editingRole.value = user.role
  showEditUserModal.value = true
}

const handleEditUser = async () => {
  if (!editingUser.value) return

  successMsg.value = ''
  errorMsg.value = ''
  usersLoading.value = true

  try {
    await userService.updateUser(editingUser.value.id, {
      username: editingUsername.value,
      password: editingPassword.value || undefined,
      role: editingRole.value
    })
    successMsg.value = `User '${editingUsername.value}' updated successfully`
    showEditUserModal.value = false
    editingUser.value = null
    await fetchUsers()
    setTimeout(() => { successMsg.value = '' }, 3000)
  } catch (err: any) {
    errorMsg.value = err.message || 'Failed to update user'
  } finally {
    usersLoading.value = false
  }
}

const handleDeleteUser = async (user: User) => {
  if (!confirm(`Are you sure you want to delete user '${user.username}'?`)) return

  successMsg.value = ''
  errorMsg.value = ''
  usersLoading.value = true

  try {
    await userService.deleteUser(user.id)
    successMsg.value = `User '${user.username}' deleted successfully`
    await fetchUsers()
    setTimeout(() => { successMsg.value = '' }, 3000)
  } catch (err: any) {
    errorMsg.value = err.message || 'Failed to delete user'
  } finally {
    usersLoading.value = false
  }
}

onMounted(async () => {
  await fetchProfile()
  if (currentUser.value?.role === 'admin') {
    await fetchUsers()
  }
})

const isAdmin = computed(() => currentUser.value?.role === 'admin')

const formatDate = (dateStr: string) => {
  if (!dateStr) return 'N/A'
  try {
    return new Date(dateStr).toLocaleString()
  } catch {
    return dateStr
  }
}
</script>

<template>
  <div class="p-6 max-w-6xl mx-auto space-y-6 animate-fade-in">
    <!-- Header -->
    <div class="flex items-center justify-between pb-4 border-b border-gray-200 dark:border-gray-800">
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white flex items-center gap-2">
          <SparklesIcon class="h-6 w-6 text-violet-500" />
          User Profile & Accounts
        </h1>
        <p class="text-gray-500 dark:text-gray-400 mt-1 text-sm">
          Manage your account credentials and system operators
        </p>
      </div>
    </div>

    <!-- Alert Messages -->
    <div v-if="successMsg" class="p-4 rounded-xl bg-green-500/10 border border-green-500/20 text-green-700 dark:text-green-300 text-sm animate-fade-in flex items-center gap-2">
      <span class="w-1.5 h-1.5 rounded-full bg-green-500 animate-ping"></span>
      <span>{{ successMsg }}</span>
    </div>

    <div v-if="errorMsg" class="p-4 rounded-xl bg-red-500/10 border border-red-500/20 text-red-700 dark:text-red-300 text-sm animate-fade-in flex items-center gap-2">
      <span class="w-1.5 h-1.5 rounded-full bg-red-500 animate-ping"></span>
      <span>{{ errorMsg }}</span>
    </div>

    <!-- Tab Selection (If Admin) -->
    <div v-if="isAdmin" class="flex border-b border-gray-200 dark:border-gray-800 gap-4">
      <button
        @click="activeTab = 'profile'"
        class="pb-3 text-sm font-semibold border-b-2 transition-all px-2 focus:outline-none cursor-pointer"
        :class="activeTab === 'profile' ? 'border-violet-600 text-violet-600 dark:text-violet-400' : 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700'"
      >
        My Profile
      </button>
      <button
        @click="activeTab = 'users'"
        class="pb-3 text-sm font-semibold border-b-2 transition-all px-2 focus:outline-none cursor-pointer"
        :class="activeTab === 'users' ? 'border-violet-600 text-violet-600 dark:text-violet-400' : 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700'"
      >
        Manage Accounts
      </button>
    </div>

    <!-- Tab 1: Profile Details -->
    <div v-if="activeTab === 'profile'" class="grid grid-cols-1 md:grid-cols-3 gap-6">
      <!-- Profile Info Sidebar Card -->
      <div class="bg-white dark:bg-slate-900 border border-gray-100 dark:border-gray-800 rounded-2xl p-6 shadow-sm flex flex-col items-center text-center relative overflow-hidden">
        <div class="absolute top-0 inset-x-0 h-20 bg-gradient-to-r from-violet-600 to-indigo-600 opacity-10"></div>
        <div class="w-20 h-20 rounded-full bg-gradient-to-br from-violet-500 to-indigo-600 flex items-center justify-center text-white text-3xl font-bold shadow-lg shadow-violet-500/20 mt-6 relative z-10">
          {{ currentUser?.username.slice(0, 2).toUpperCase() }}
        </div>
        <h2 class="text-lg font-bold text-gray-900 dark:text-white mt-4">{{ currentUser?.username }}</h2>
        <span class="px-3 py-1 rounded-full text-xs font-semibold mt-2 inline-flex items-center gap-1.5"
              :class="currentUser?.role === 'admin' ? 'bg-violet-100 text-violet-700 dark:bg-violet-950/40 dark:text-violet-400' : 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-400'">
          <ShieldCheckIcon class="h-4.5 w-4.5" />
          {{ currentUser?.role === 'admin' ? 'Administrator' : 'Viewer' }}
        </span>

        <div class="w-full border-t border-gray-100 dark:border-gray-800 mt-6 pt-6 text-left space-y-4">
          <div class="flex items-center gap-3 text-sm text-gray-600 dark:text-gray-400">
            <CalendarIcon class="h-5 w-5 text-gray-400" />
            <div>
              <p class="text-xs font-medium text-gray-400">Created At</p>
              <p class="font-medium mt-0.5">{{ formatDate(currentUser?.created_at || '') }}</p>
            </div>
          </div>
        </div>
      </div>

      <!-- Update Form Card -->
      <div class="md:col-span-2 bg-white dark:bg-slate-900 border border-gray-100 dark:border-gray-800 rounded-2xl p-6 shadow-sm">
        <h3 class="text-md font-bold text-gray-900 dark:text-white mb-6 flex items-center gap-2">
          <UserIcon class="h-5 w-5 text-violet-500" />
          Update Credentials
        </h3>
        <form @submit.prevent="handleUpdateProfile" class="space-y-6">
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label class="block text-xs font-semibold text-gray-500 dark:text-gray-400 mb-2">Username</label>
              <input
                type="text"
                v-model="profileUsername"
                required
                class="w-full bg-gray-50 dark:bg-slate-800 border border-gray-200 dark:border-gray-700 rounded-xl px-4 py-2.5 text-sm text-gray-900 dark:text-white focus:outline-none focus:border-violet-500"
              />
            </div>
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 border-t border-gray-100 dark:border-gray-800 pt-6">
            <div>
              <label class="block text-xs font-semibold text-gray-500 dark:text-gray-400 mb-2">New Password (leave empty to keep current)</label>
              <input
                type="password"
                v-model="profilePassword"
                placeholder="••••••••"
                class="w-full bg-gray-50 dark:bg-slate-800 border border-gray-200 dark:border-gray-700 rounded-xl px-4 py-2.5 text-sm text-gray-900 dark:text-white focus:outline-none focus:border-violet-500"
              />
            </div>
            <div>
              <label class="block text-xs font-semibold text-gray-500 dark:text-gray-400 mb-2">Confirm New Password</label>
              <input
                type="password"
                v-model="profileConfirmPassword"
                placeholder="••••••••"
                class="w-full bg-gray-50 dark:bg-slate-800 border border-gray-200 dark:border-gray-700 rounded-xl px-4 py-2.5 text-sm text-gray-900 dark:text-white focus:outline-none focus:border-violet-500"
              />
            </div>
          </div>

          <div class="flex justify-end pt-4">
            <button
              type="submit"
              :disabled="profileLoading"
              class="bg-violet-600 hover:bg-violet-500 text-white font-medium text-sm px-6 py-2.5 rounded-xl shadow-md transition-colors duration-200 disabled:opacity-50 cursor-pointer flex items-center gap-2"
            >
              <span v-if="profileLoading" class="w-4 h-4 border-2 border-white/20 border-t-white rounded-full animate-spin"></span>
              Save Changes
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Tab 2: Users Management -->
    <div v-else-if="activeTab === 'users' && isAdmin" class="space-y-6">
      <!-- Users Table Control Header -->
      <div class="flex items-center justify-between">
        <h3 class="text-md font-bold text-gray-900 dark:text-white flex items-center gap-2">
          <ShieldCheckIcon class="h-5 w-5 text-violet-500" />
          Operator Accounts
        </h3>
        <button
          @click="showAddUserModal = true"
          class="bg-violet-600 hover:bg-violet-500 text-white text-xs font-semibold px-4 py-2 rounded-xl flex items-center gap-1.5 shadow-sm transition-colors cursor-pointer"
        >
          <PlusIcon class="h-4 w-4" />
          Add User
        </button>
      </div>

      <!-- Users List -->
      <div class="bg-white dark:bg-slate-900 border border-gray-100 dark:border-gray-800 rounded-2xl shadow-sm overflow-hidden">
        <div class="overflow-x-auto">
          <table class="w-full text-left border-collapse">
            <thead>
              <tr class="bg-gray-50 dark:bg-slate-800/40 text-xs font-semibold text-gray-500 dark:text-gray-400 border-b border-gray-100 dark:border-gray-800">
                <th class="px-6 py-4">ID</th>
                <th class="px-6 py-4">Username</th>
                <th class="px-6 py-4">Role</th>
                <th class="px-6 py-4">Created At</th>
                <th class="px-6 py-4 text-right">Actions</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-gray-800 text-sm text-gray-800 dark:text-gray-200">
              <tr v-for="user in users" :key="user.id" class="hover:bg-gray-50/55 dark:hover:bg-slate-800/20 transition-colors">
                <td class="px-6 py-4 font-mono text-xs text-gray-400">#{{ user.id }}</td>
                <td class="px-6 py-4 font-medium">{{ user.username }}</td>
                <td class="px-6 py-4">
                  <span class="px-2.5 py-0.5 rounded-full text-xs font-semibold"
                        :class="user.role === 'admin' ? 'bg-violet-100 text-violet-700 dark:bg-violet-950/40 dark:text-violet-400' : 'bg-gray-100 text-gray-700 dark:bg-gray-850 dark:text-gray-400'">
                    {{ user.role }}
                  </span>
                </td>
                <td class="px-6 py-4 text-gray-500 dark:text-gray-400 text-xs">{{ formatDate(user.created_at) }}</td>
                <td class="px-6 py-4 text-right flex justify-end gap-2">
                  <button
                    @click="startEditUser(user)"
                    class="p-1.5 rounded-lg text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors cursor-pointer"
                    title="Edit User"
                  >
                    <PencilIcon class="h-4.5 w-4.5" />
                  </button>
                  <button
                    @click="handleDeleteUser(user)"
                    :disabled="user.id === currentUser?.id"
                    class="p-1.5 rounded-lg text-red-500 hover:bg-red-500/10 disabled:opacity-30 disabled:hover:bg-transparent transition-colors cursor-pointer"
                    title="Delete User"
                  >
                    <TrashIcon class="h-4.5 w-4.5" />
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- Create User Modal -->
    <div v-if="showAddUserModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
      <div class="bg-white dark:bg-slate-900 rounded-2xl border border-gray-100 dark:border-gray-800 max-w-md w-full p-6 shadow-2xl animate-scale-up">
        <h3 class="text-md font-bold text-gray-900 dark:text-white mb-4">Add Operator Account</h3>
        <form @submit.prevent="handleAddUser" class="space-y-4">
          <div>
            <label class="block text-xs font-semibold text-gray-500 dark:text-gray-400 mb-1">Username</label>
            <input
              type="text"
              v-model="newUserUsername"
              required
              class="w-full bg-gray-50 dark:bg-slate-800 border border-gray-200 dark:border-gray-700 rounded-xl px-4 py-2.5 text-sm text-gray-900 dark:text-white focus:outline-none"
            />
          </div>
          <div>
            <label class="block text-xs font-semibold text-gray-500 dark:text-gray-400 mb-1">Password</label>
            <input
              type="password"
              v-model="newUserPassword"
              required
              class="w-full bg-gray-50 dark:bg-slate-800 border border-gray-200 dark:border-gray-700 rounded-xl px-4 py-2.5 text-sm text-gray-900 dark:text-white focus:outline-none"
            />
          </div>
          <div>
            <label class="block text-xs font-semibold text-gray-500 dark:text-gray-400 mb-1">Role</label>
            <select
              v-model="newUserRole"
              class="w-full bg-gray-50 dark:bg-slate-800 border border-gray-200 dark:border-gray-700 rounded-xl px-4 py-2.5 text-sm text-gray-900 dark:text-white focus:outline-none"
            >
              <option value="viewer">Viewer (Read-only logs / status)</option>
              <option value="admin">Administrator (Full control)</option>
            </select>
          </div>
          <div class="flex justify-end gap-3 pt-2">
            <button
              type="button"
              @click="showAddUserModal = false"
              class="px-4 py-2 text-xs font-semibold border border-gray-200 dark:border-gray-700 text-gray-600 dark:text-gray-300 rounded-xl hover:bg-gray-55/10 cursor-pointer"
            >
              Cancel
            </button>
            <button
              type="submit"
              class="bg-violet-600 hover:bg-violet-500 text-white text-xs font-semibold px-4 py-2 rounded-xl cursor-pointer"
            >
              Create
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Edit User Modal -->
    <div v-if="showEditUserModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
      <div class="bg-white dark:bg-slate-900 rounded-2xl border border-gray-100 dark:border-gray-800 max-w-md w-full p-6 shadow-2xl animate-scale-up">
        <h3 class="text-md font-bold text-gray-900 dark:text-white mb-4">Edit User</h3>
        <form @submit.prevent="handleEditUser" class="space-y-4">
          <div>
            <label class="block text-xs font-semibold text-gray-500 dark:text-gray-400 mb-1">Username</label>
            <input
              type="text"
              v-model="editingUsername"
              required
              class="w-full bg-gray-50 dark:bg-slate-800 border border-gray-200 dark:border-gray-700 rounded-xl px-4 py-2.5 text-sm text-gray-900 dark:text-white focus:outline-none"
            />
          </div>
          <div>
            <label class="block text-xs font-semibold text-gray-500 dark:text-gray-400 mb-1">Reset Password (leave empty to keep current)</label>
            <input
              type="password"
              v-model="editingPassword"
              placeholder="••••••••"
              class="w-full bg-gray-50 dark:bg-slate-800 border border-gray-200 dark:border-gray-700 rounded-xl px-4 py-2.5 text-sm text-gray-900 dark:text-white focus:outline-none"
            />
          </div>
          <div>
            <label class="block text-xs font-semibold text-gray-500 dark:text-gray-400 mb-1">Role</label>
            <select
              v-model="editingRole"
              :disabled="editingUser?.id === currentUser?.id"
              class="w-full bg-gray-50 dark:bg-slate-800 border border-gray-200 dark:border-gray-700 rounded-xl px-4 py-2.5 text-sm text-gray-900 dark:text-white focus:outline-none disabled:opacity-50"
            >
              <option value="viewer">Viewer (Read-only logs / status)</option>
              <option value="admin">Administrator (Full control)</option>
            </select>
          </div>
          <div class="flex justify-end gap-3 pt-2">
            <button
              type="button"
              @click="showEditUserModal = false; editingUser = null"
              class="px-4 py-2 text-xs font-semibold border border-gray-200 dark:border-gray-700 text-gray-600 dark:text-gray-300 rounded-xl hover:bg-gray-55/10 cursor-pointer"
            >
              Cancel
            </button>
            <button
              type="submit"
              class="bg-violet-600 hover:bg-violet-500 text-white text-xs font-semibold px-4 py-2 rounded-xl cursor-pointer"
            >
              Save
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>
