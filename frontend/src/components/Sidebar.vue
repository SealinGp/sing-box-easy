<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import type { Component } from 'vue'
import { ChevronRightIcon, ChevronDownIcon, Cog6ToothIcon, ArrowLeftOnRectangleIcon } from '@heroicons/vue/24/outline'
import LanguageSwitcher from './LanguageSwitcher.vue'
import { useServiceStore } from '../stores'
import { userService } from '../services'

interface MenuItem {
  name: string
  icon: Component
  path?: string
  badge?: number | string
  children?: MenuItem[]
}

interface Props {
  menuItems: MenuItem[]
}

defineProps<Props>()
const route = useRoute()
const router = useRouter()
const { t } = useI18n()

// App version, injected at build time (see vite.config.ts `define`).
const version = __APP_VERSION__

// Shared service status, polled here so the running/stopped indicator stays
// live on every page (e.g. right after restarting from the Config page).
const serviceStore = useServiceStore()
let unsubscribe: (() => void) | null = null

const currentUser = ref<any>(null)

const fetchUser = async () => {
  try {
    currentUser.value = await userService.getMe()
  } catch (err) {
    console.error('Failed to get current user in sidebar:', err)
  }
}

const userInitial = computed(() => {
  if (!currentUser.value?.username) return 'U'
  return currentUser.value.username.slice(0, 2).toUpperCase()
})

const username = computed(() => currentUser.value?.username || 'User')
const userRole = computed(() => currentUser.value?.role || 'viewer')

const handleLogout = async () => {
  if (confirm('Are you sure you want to sign out?')) {
    try {
      await userService.logout()
      router.push('/login')
    } catch (err) {
      console.error('Logout failed:', err)
      // Force local clean up on error
      localStorage.removeItem('sb_token')
      window.location.href = '/login'
    }
  }
}

onMounted(async () => {
  unsubscribe = serviceStore.startPolling(5000)
  await fetchUser()
})
onUnmounted(() => {
  unsubscribe?.()
})

const serviceStatus = computed(() => serviceStore.status?.status ?? 'unknown')
const serviceDotClass = computed(() => {
  if (serviceStore.error) return 'bg-yellow-500'
  switch (serviceStatus.value) {
    case 'running':
      return 'bg-green-500'
    case 'stopped':
      return 'bg-red-500'
    default:
      return 'bg-gray-400'
  }
})
const serviceLabel = computed(() => {
  if (serviceStore.error) return t('overview.status.unknown')
  switch (serviceStatus.value) {
    case 'running':
      return t('overview.status.running')
    case 'stopped':
      return t('overview.status.stopped')
    default:
      return t('overview.status.unknown')
  }
})

// Track expanded states for menu items with children
const expandedItems = ref<Set<string>>(new Set())

const toggleExpanded = (itemName: string) => {
  const next = new Set(expandedItems.value)
  if (next.has(itemName)) next.delete(itemName); else next.add(itemName)
  expandedItems.value = next
}

const isExpanded = (itemName: string) => {
  return expandedItems.value.has(itemName)
}

const isActive = (path?: string) => {
  if (!path) return false
  return route.path === path || route.path.startsWith(path + '/')
}

const isParentActive = (item: MenuItem) => {
  if (item.path && isActive(item.path)) return true
  if (item.children) {
    return item.children.some(child => isActive(child.path))
  }
  return false
}
</script>

<template>
  <div class="w-55 bg-white/95 backdrop-blur-sm dark:bg-slate-900/95 h-full flex flex-col shadow-xl">
    <!-- Logo/Brand with Menu label -->
    <div class="p-6 pb-4">
      <!-- Row 1: icon + name -->
      <div class="flex items-center gap-2">
        <img src="/logo.jpg" alt="Sing Box Easy" class="h-7 w-7 rounded-lg flex-shrink-0 shadow-sm" />
        <span class="text-sm font-semibold text-gray-900 dark:text-white">Sing Box Easy</span>
      </div>
      <!-- Row 2 (secondary): version + language switch + settings -->
      <div class="flex items-center gap-2 mt-2 mb-4">
        <span class="text-[10px] font-medium text-gray-400 dark:text-gray-500">{{ version }}</span>
        <LanguageSwitcher variant="compact" class="ml-auto" />
        <router-link
          to="/dashboard/settings"
          class="p-1.5 rounded-lg text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
          :class="{ 'text-violet-600 dark:text-violet-400 bg-violet-100 dark:bg-violet-900/30': isActive('/dashboard/settings') }"
          :title="$t('nav.settings')"
        >
          <Cog6ToothIcon class="h-4 w-4" />
        </router-link>
      </div>
      <!-- Row 3: live sing-box service status (polled) -->
      <router-link
        to="/dashboard/overview"
        class="flex items-center gap-2 px-2 py-1.5 rounded-lg bg-gray-50 dark:bg-slate-800/60 hover:bg-gray-100 dark:hover:bg-slate-800 transition-colors"
        :title="$t('nav.serviceStatusHint')"
      >
        <span class="relative flex h-2.5 w-2.5">
          <span
            v-if="serviceStatus === 'running'"
            class="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-60"
          ></span>
          <span class="relative inline-flex rounded-full h-2.5 w-2.5" :class="serviceDotClass"></span>
        </span>
        <span class="text-xs font-medium text-gray-600 dark:text-gray-300">sing-box</span>
        <span class="text-xs text-gray-500 dark:text-gray-400 ml-auto">{{ serviceLabel }}</span>
      </router-link>
    </div>

    <!-- Menu Items -->
    <nav class="flex-1 p-4 overflow-y-auto">
      <ul class="space-y-1">
        <li v-for="item in menuItems" :key="item.name">
          <!-- Parent Item -->
          <div>
            <router-link
              v-if="!item.children"
              :to="item.path || '#'"
              :class="[
                'flex items-center justify-between px-4 py-2.5 rounded-full text-sm font-medium transition-all duration-200',
                isActive(item.path)
                  ? 'bg-gray-900 text-white dark:bg-violet-600'
                  : 'text-gray-700 dark:text-gray-300 hover:bg-gray-100/80 dark:hover:bg-gray-800/80'
              ]"
            >
              <div class="flex items-center gap-3">
                <component :is="item.icon" class="h-5 w-5 flex-shrink-0" />
                <span>{{ item.name }}</span>
              </div>
              <div v-if="item.badge" class="flex items-center">
                <span
                  :class="[
                    'px-2.5 py-1 text-xs font-semibold rounded-full',
                    isActive(item.path)
                      ? 'bg-white text-gray-900 dark:bg-white/20 dark:text-white'
                      : 'bg-gray-900 text-white dark:bg-violet-600 dark:text-white'
                  ]"
                >
                  {{ item.badge }}
                </span>
              </div>
            </router-link>

            <button
              v-else
              @click="toggleExpanded(item.name)"
              :class="[
                'w-full flex items-center justify-between px-4 py-2.5 rounded-full text-sm font-medium transition-all duration-200',
                isParentActive(item) || isExpanded(item.name)
                  ? 'bg-gray-900 text-white dark:bg-gray-800'
                  : 'text-gray-700 dark:text-gray-300 hover:bg-gray-100/80 dark:hover:bg-gray-800/80'
              ]"
            >
              <div class="flex items-center gap-3">
                <component :is="item.icon" class="h-5 w-5 flex-shrink-0" />
                <span>{{ item.name }}</span>
              </div>
              <div class="flex items-center gap-2">
                <span
                  v-if="item.badge"
                  :class="[
                    'px-2 py-0.5 text-xs font-medium rounded-full',
                    isParentActive(item)
                      ? 'bg-white/20 text-white'
                      : 'bg-violet-100 text-violet-600 dark:bg-violet-900/30 dark:text-violet-400'
                  ]"
                >
                  {{ item.badge }}
                </span>
                <ChevronDownIcon
                  v-if="isExpanded(item.name)"
                  class="h-4 w-4 transition-transform duration-200"
                />
                <ChevronRightIcon
                  v-else
                  class="h-4 w-4 transition-transform duration-200"
                />
              </div>
            </button>
          </div>

          <!-- Children Items -->
          <Transition
            enter-active-class="transition-all duration-200 ease-out"
            enter-from-class="opacity-0 max-h-0"
            enter-to-class="opacity-100 max-h-96"
            leave-active-class="transition-all duration-200 ease-in"
            leave-from-class="opacity-100 max-h-96"
            leave-to-class="opacity-0 max-h-0"
          >
            <ul
              v-if="item.children && isExpanded(item.name)"
              class="mt-2 ml-8 space-y-1 overflow-hidden"
            >
              <li v-for="child in item.children" :key="child.name">
                <router-link
                  :to="child.path || '#'"
                  :class="[
                    'flex items-center justify-between px-4 py-2 rounded-full text-sm transition-all duration-200',
                    isActive(child.path)
                      ? 'bg-gray-200/80 dark:bg-violet-900/30 text-gray-900 dark:text-violet-300 font-medium'
                      : 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200 hover:bg-gray-100/60 dark:hover:bg-gray-800/50'
                  ]"
                >
                  <div class="flex items-center gap-3">
                    <component
                      v-if="child.icon"
                      :is="child.icon"
                      class="h-4 w-4 flex-shrink-0"
                    />
                    <div
                      v-else
                      class="w-4 h-4 rounded-full border-2 flex-shrink-0"
                      :class="[
                        isActive(child.path)
                          ? 'border-violet-600 bg-violet-600'
                          : 'border-gray-400 dark:border-gray-600'
                      ]"
                    />
                    <span>{{ child.name }}</span>
                  </div>
                  <span
                    v-if="child.badge"
                    class="px-2 py-0.5 text-xs font-medium rounded-full bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400"
                  >
                    {{ child.badge }}
                  </span>
                </router-link>
              </li>
            </ul>
          </Transition>
        </li>
      </ul>
    </nav>

    <!-- Footer/User Section -->
    <div class="p-4 border-t border-gray-100 dark:border-gray-800">
      <div class="flex items-center justify-between gap-2">
        <router-link
          to="/dashboard/profile"
          class="flex items-center gap-3 px-2 py-1.5 rounded-xl hover:bg-gray-100 dark:hover:bg-gray-850 transition-colors flex-1 min-w-0"
          :class="{ 'bg-gray-100 dark:bg-gray-800': isActive('/dashboard/profile') }"
        >
          <div class="w-8 h-8 rounded-full bg-gradient-to-br from-violet-500 to-purple-600 flex items-center justify-center text-white text-sm font-semibold flex-shrink-0 shadow-sm">
            {{ userInitial }}
          </div>
          <div class="flex-1 min-w-0 text-left">
            <p class="text-sm font-medium text-gray-900 dark:text-white truncate" :title="username">{{ username }}</p>
            <p class="text-[10px] text-gray-500 dark:text-gray-400 capitalize">{{ userRole }}</p>
          </div>
        </router-link>

        <button
          @click="handleLogout"
          class="p-2 rounded-xl text-red-500 hover:bg-red-500/10 transition-colors cursor-pointer"
          title="Sign Out"
        >
          <ArrowLeftOnRectangleIcon class="h-4.5 w-4.5" />
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* Custom scrollbar for the navigation */
nav::-webkit-scrollbar {
  width: 4px;
}

nav::-webkit-scrollbar-track {
  background: transparent;
}

nav::-webkit-scrollbar-thumb {
  background-color: rgba(209, 213, 219, 0.5);
  border-radius: 9999px;
}

nav::-webkit-scrollbar-thumb:hover {
  background-color: rgba(156, 163, 175, 0.7);
}

@media (prefers-color-scheme: dark) {
  nav::-webkit-scrollbar-thumb {
    background-color: rgba(55, 65, 81, 0.5);
  }

  nav::-webkit-scrollbar-thumb:hover {
    background-color: rgba(75, 85, 99, 0.7);
  }
}
</style>