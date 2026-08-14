<script setup lang="ts">
/**
 * Horizontal navigation shell.
 *
 * Used on OpenWrt, where the panel is reached from a router that already has
 * LuCI's own sidebar down the left edge. A second vertical menu there competes
 * for the same space and reads as two applications fighting; moving our
 * navigation to the top gives the config forms the full width of the screen,
 * which matters on the small displays these routers are usually managed from.
 *
 * Renders exactly the same `menuItems` tree as the sidebar — parents with
 * children become dropdowns instead of collapsible sections.
 */
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import type { Component } from 'vue'
import { ChevronDownIcon, Cog6ToothIcon, ArrowLeftOnRectangleIcon } from '@heroicons/vue/24/outline'
import LanguageSwitcher from './LanguageSwitcher.vue'
import { useDeployment } from '../composables/useDeployment'
import { useNavChrome } from '../composables/useNavChrome'

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

const { authEnabled } = useDeployment()

const {
  version,
  latestVersion,
  updateOffered,
  userInitial,
  username,
  serviceStatus,
  serviceDotClass,
  serviceLabel,
  handleLogout,
} = useNavChrome()

const isActive = (path?: string) => {
  if (!path) return false
  return route.path === path || route.path.startsWith(path + '/')
}

const isParentActive = (item: MenuItem) => {
  if (item.path && isActive(item.path)) return true
  return item.children?.some((child) => isActive(child.path)) ?? false
}

// --- Dropdowns --------------------------------------------------------------
// Only one menu is open at a time; `null` means none. Navigating always closes
// the menu, otherwise the dropdown would linger over the page it just opened.

const openMenu = ref<string | null>(null)
const navRef = ref<HTMLElement | null>(null)

const toggleMenu = (name: string) => {
  openMenu.value = openMenu.value === name ? null : name
}

const closeMenu = () => {
  openMenu.value = null
}

watch(() => route.path, closeMenu)

const handleDocumentClick = (event: MouseEvent) => {
  if (!openMenu.value) return
  const target = event.target as Node | null
  if (target && navRef.value?.contains(target)) return
  closeMenu()
}

const handleEscape = (event: KeyboardEvent) => {
  if (event.key === 'Escape') closeMenu()
}

onMounted(() => {
  document.addEventListener('click', handleDocumentClick)
  document.addEventListener('keydown', handleEscape)
})

onBeforeUnmount(() => {
  document.removeEventListener('click', handleDocumentClick)
  document.removeEventListener('keydown', handleEscape)
})

const serviceTitle = computed(() => `sing-box — ${serviceLabel.value}`)
</script>

<template>
  <!--
    `relative z-30` is load-bearing, not decoration. `.liquid-topbar` sets
    `backdrop-filter`, which makes the header a stacking context; while the
    header stayed unpositioned that context painted with ordinary in-flow
    content, so the page's cards — later in the DOM — covered the dropdowns no
    matter how high their own z-index went. Positioning the header lifts the
    whole subtree into the positioned paint phase. 30 keeps it under the
    app's `z-50` modal overlays.
  -->
  <header class="liquid-topbar relative z-30 shadow-float m-3 mb-0 rounded-surface px-4 py-2">
    <div class="flex items-center gap-3">
      <!-- Brand -->
      <router-link to="/dashboard/overview" class="flex items-center gap-2 flex-shrink-0">
        <img src="/logo.jpg" alt="Sing Box Easy" class="h-7 w-7 rounded-surface shadow-surface" />
        <div class="hidden sm:flex flex-col leading-tight">
          <span class="text-sm font-semibold text-gray-900 dark:text-white">Sing Box Easy</span>
          <span class="text-[10px] font-medium text-gray-400 dark:text-gray-500">{{ version }}</span>
        </div>
      </router-link>

      <!-- "-> vX.Y.Z" update affordance; opens the Settings update panel. -->
      <router-link
        v-if="updateOffered"
        to="/dashboard/settings"
        class="flex items-center gap-1 px-1.5 py-0.5 rounded-pill bg-emerald-500/15 text-[10px] font-semibold text-emerald-700 dark:text-emerald-300 hover:bg-emerald-500/25 transition-colors flex-shrink-0"
        :title="$t('settings.update.updateTo', { version: latestVersion })"
      >
        <span class="relative flex h-1.5 w-1.5">
          <span class="animate-ping absolute inline-flex h-full w-full rounded-pill bg-emerald-400 opacity-70"></span>
          <span class="relative inline-flex rounded-pill h-1.5 w-1.5 bg-emerald-500"></span>
        </span>
        <span>&rarr; {{ latestVersion }}</span>
      </router-link>

      <!-- Menu -->
      <!--
        The menu wraps rather than scrolls on narrow router screens. A
        horizontally scrolling container would establish its own clipping
        context, and the dropdowns — absolutely positioned below their parent —
        would be cut off by it.
      -->
      <nav ref="navRef" class="flex-1 min-w-0">
        <ul class="flex flex-wrap items-center gap-1">
          <li v-for="item in menuItems" :key="item.name" class="relative flex-shrink-0">
            <!-- Leaf entry -->
            <router-link
              v-if="!item.children"
              :to="item.path || '#'"
              :class="[
                'flex items-center gap-2 px-3 py-2 rounded-pill text-sm font-medium whitespace-nowrap transition-colors',
                isActive(item.path)
                  ? 'topbar-item-active text-white'
                  : 'topbar-item hover:bg-white/40 dark:hover:bg-white/10',
              ]"
            >
              <component :is="item.icon" class="h-4.5 w-4.5 flex-shrink-0" />
              <span>{{ item.name }}</span>
              <span
                v-if="item.badge"
                :class="[
                  'px-2 py-0.5 text-xs font-semibold rounded-pill',
                  isActive(item.path)
                    ? 'bg-white/25 text-white'
                    : 'bg-white/40 text-gray-900 dark:bg-white/10 dark:text-white',
                ]"
              >
                {{ item.badge }}
              </span>
            </router-link>

            <!-- Parent entry: opens a dropdown of its children -->
            <template v-else>
              <button
                @click="toggleMenu(item.name)"
                :aria-expanded="openMenu === item.name"
                aria-haspopup="true"
                :class="[
                  'flex items-center gap-2 px-3 py-2 rounded-pill text-sm font-medium whitespace-nowrap transition-colors cursor-pointer',
                  isParentActive(item)
                    ? 'topbar-item-active text-white'
                    : openMenu === item.name
                      ? 'topbar-item bg-white/45 dark:bg-white/10'
                      : 'topbar-item hover:bg-white/40 dark:hover:bg-white/10',
                ]"
              >
                <component :is="item.icon" class="h-4.5 w-4.5 flex-shrink-0" />
                <span>{{ item.name }}</span>
                <ChevronDownIcon
                  class="h-3.5 w-3.5 transition-transform duration-200"
                  :class="openMenu === item.name ? 'rotate-180' : 'rotate-0'"
                />
              </button>

              <Transition
                enter-active-class="transition duration-150 ease-out"
                enter-from-class="opacity-0 -translate-y-1"
                enter-to-class="opacity-100 translate-y-0"
                leave-active-class="transition duration-100 ease-in"
                leave-from-class="opacity-100 translate-y-0"
                leave-to-class="opacity-0 -translate-y-1"
              >
                <ul
                  v-if="openMenu === item.name"
                  class="topbar-dropdown absolute left-0 top-full mt-2 min-w-48 rounded-surface p-1.5 shadow-float z-50"
                >
                  <li v-for="child in item.children" :key="child.name">
                    <router-link
                      :to="child.path || '#'"
                      :class="[
                        'flex items-center gap-3 px-3 py-2 rounded-control text-sm whitespace-nowrap transition-colors',
                        isActive(child.path)
                          ? 'topbar-child-active font-semibold'
                          : 'topbar-item hover:bg-white/45 dark:hover:bg-white/10',
                      ]"
                    >
                      <component v-if="child.icon" :is="child.icon" class="h-4 w-4 flex-shrink-0" />
                      <span>{{ child.name }}</span>
                      <span
                        v-if="child.badge"
                        class="ml-auto px-2 py-0.5 text-xs font-medium rounded-pill bg-white/35 dark:bg-white/10"
                      >
                        {{ child.badge }}
                      </span>
                    </router-link>
                  </li>
                </ul>
              </Transition>
            </template>
          </li>
        </ul>
      </nav>

      <!-- Right rail: live status, language, settings, account -->
      <div class="flex items-center gap-1.5 flex-shrink-0">
        <router-link
          to="/dashboard/overview"
          class="topbar-status flex items-center gap-2 px-2.5 py-1.5 rounded-pill transition-colors"
          :title="serviceTitle"
        >
          <span class="relative flex h-2.5 w-2.5">
            <span
              v-if="serviceStatus === 'running'"
              class="animate-ping absolute inline-flex h-full w-full rounded-pill bg-green-400 opacity-60"
            ></span>
            <span class="relative inline-flex rounded-pill h-2.5 w-2.5" :class="serviceDotClass"></span>
          </span>
          <span class="hidden lg:inline text-xs font-semibold topbar-item">sing-box</span>
          <span class="hidden xl:inline text-xs text-gray-500 dark:text-gray-400">{{ serviceLabel }}</span>
        </router-link>

        <LanguageSwitcher variant="compact" />

        <router-link
          to="/dashboard/settings"
          class="p-2 rounded-control transition-colors"
          :class="
            isActive('/dashboard/settings')
              ? 'topbar-child-active'
              : 'topbar-item hover:bg-white/40 dark:hover:bg-white/10'
          "
          :title="$t('nav.settings')"
        >
          <Cog6ToothIcon class="h-4.5 w-4.5" />
        </router-link>

        <!-- Account controls only exist when this deployment requires login. -->
        <template v-if="authEnabled">
          <router-link
            to="/dashboard/profile"
            class="flex items-center gap-2 pl-1 pr-2 py-1 rounded-pill hover:bg-white/40 dark:hover:bg-white/10 transition-colors"
            :class="{ 'bg-white/45 dark:bg-white/10': isActive('/dashboard/profile') }"
            :title="username"
          >
            <span
              class="w-7 h-7 rounded-pill bg-gradient-to-br from-blue-500 to-cyan-300 flex items-center justify-center text-white text-xs font-semibold shadow-surface"
            >
              {{ userInitial }}
            </span>
            <span class="hidden lg:inline text-sm font-medium topbar-item max-w-24 truncate">{{ username }}</span>
          </router-link>

          <button
            @click="handleLogout"
            class="p-2 rounded-control text-red-500 hover:bg-red-500/10 transition-colors cursor-pointer"
            :title="$t('nav.signOut')"
          >
            <ArrowLeftOnRectangleIcon class="h-4.5 w-4.5" />
          </button>
        </template>
      </div>
    </div>
  </header>
</template>

<style scoped>
.liquid-topbar {
  background:
    linear-gradient(145deg, rgba(255, 255, 255, 0.48), rgba(255, 255, 255, 0.18)),
    rgba(255, 255, 255, 0.82);
  border: 1px solid rgba(255, 255, 255, 0.58);
  box-shadow: 0 18px 50px rgba(15, 23, 42, 0.14), var(--glass-highlight);
  backdrop-filter: var(--glass-blur);
  -webkit-backdrop-filter: var(--glass-blur);
}

.topbar-item {
  color: #172033;
}

.topbar-item-active {
  background: linear-gradient(135deg, var(--color-primary), var(--color-primary-light));
  box-shadow: 0 6px 18px rgba(21, 117, 255, 0.28);
}

.topbar-child-active {
  background: rgba(21, 117, 255, 0.16);
  color: var(--color-primary-dark);
}

.topbar-status {
  background: rgba(255, 255, 255, 0.48);
  border: 1px solid rgba(148, 163, 184, 0.22);
  box-shadow: var(--glass-highlight);
}

.topbar-dropdown {
  background:
    linear-gradient(145deg, rgba(255, 255, 255, 0.6), rgba(255, 255, 255, 0.3)),
    rgba(255, 255, 255, 0.94);
  border: 1px solid rgba(255, 255, 255, 0.6);
  backdrop-filter: var(--glass-blur);
  -webkit-backdrop-filter: var(--glass-blur);
}

@media (prefers-color-scheme: dark) {
  .liquid-topbar {
    background:
      linear-gradient(145deg, rgba(255, 255, 255, 0.13), rgba(255, 255, 255, 0.045)),
      rgba(9, 14, 21, 0.94);
    border-color: rgba(255, 255, 255, 0.14);
  }

  .topbar-item {
    color: #f4f8ff;
  }

  .topbar-child-active {
    background: rgba(107, 165, 255, 0.22);
    color: #cfe2ff;
  }

  .topbar-status {
    background: rgba(255, 255, 255, 0.105);
  }

  .topbar-dropdown {
    background:
      linear-gradient(145deg, rgba(255, 255, 255, 0.1), rgba(255, 255, 255, 0.04)),
      rgba(12, 18, 27, 0.97);
    border-color: rgba(255, 255, 255, 0.14);
  }
}

@media (prefers-reduced-motion: reduce) {
  .topbar-dropdown {
    transition: none;
  }
}
</style>
