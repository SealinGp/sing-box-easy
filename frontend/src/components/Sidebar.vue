<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import type { Component } from 'vue'
import { ChevronRightIcon, Cog6ToothIcon, ArrowLeftOnRectangleIcon } from '@heroicons/vue/24/outline'
import LanguageSwitcher from './LanguageSwitcher.vue'
import { useServiceStore } from '../stores'
import { userService } from '../services'
import { useConfirm } from '../composables/useConfirm'
import { useAppUpdate } from '../composables/useAppUpdate'
import { useAuthMode } from '../composables/useAuthMode'
import { useNavIndicator } from '../composables/useNavIndicator'

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

const props = defineProps<Props>()
const route = useRoute()
const router = useRouter()
const { t, locale } = useI18n()
const { confirm } = useConfirm()

// App version. The bundle carries a build-time stamp (see vite.config.ts
// `define`), but the backend is the authority once it reports its own version —
// after a self-update the two agree again.
const buildVersion = __APP_VERSION__
const {
  currentVersion,
  latestVersion,
  updateOffered,
  refreshStatus,
} = useAppUpdate()

const version = computed(() => currentVersion.value || buildVersion)

// Shared service status, polled here so the running/stopped indicator stays
// live on every page (e.g. right after restarting from the Config page).
const serviceStore = useServiceStore()
let unsubscribe: (() => void) | null = null

// Hide the profile/logout footer entirely when the deployment has
// authentication disabled (e.g. OpenWrt) — there is no account to manage.
const { authEnabled } = useAuthMode()

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
  const ok = await confirm({
    title: t('common.confirmTitle'),
    message: locale.value.startsWith('zh') ? '确定要退出登录吗？' : 'Are you sure you want to sign out?',
    confirmLabel: t('common.confirm'),
    cancelLabel: t('common.cancel'),
    tone: 'danger',
  })
  if (!ok) return

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

onMounted(async () => {
  unsubscribe = serviceStore.startPolling(5000)
  await fetchUser()
  // Cached server-side for a few minutes, so this is cheap and non-blocking.
  void refreshStatus(false)
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

const syncExpandedItemsWithRoute = (items: MenuItem[]) => {
  const next = new Set(expandedItems.value)
  for (const item of items) {
    if (item.children?.some(child => isActive(child.path))) {
      next.add(item.name)
    }
  }
  expandedItems.value = next
}

watch(
  () => route.path,
  () => syncExpandedItemsWithRoute(props.menuItems),
  { immediate: true }
)

// --- Sliding active-item indicator -----------------------------------------
// Two pills: `primary` is the gradient on the top-level active entry, and
// `secondary` is the tint on the active submenu entry. Each slides to its
// target instead of the target painting its own (non-animatable) gradient.

const navRef = ref<HTMLElement | null>(null)
const navListRef = ref<HTMLElement | null>(null)
const primaryPillRef = ref<HTMLElement | null>(null)
const secondaryPillRef = ref<HTMLElement | null>(null)

const primaryPill = useNavIndicator({
  scroller: navRef,
  content: navListRef,
  indicator: primaryPillRef,
  selector: '[data-nav-pill="primary"]',
})

const secondaryPill = useNavIndicator({
  scroller: navRef,
  content: navListRef,
  indicator: secondaryPillRef,
  selector: '[data-nav-pill="secondary"]',
})

const remeasurePills = () => {
  primaryPill.measure()
  secondaryPill.measure()
}

// The ResizeObserver catches size changes, but a route change only swaps which
// element carries the marker attribute — the list's size never changes — so the
// pills must be told to re-measure explicitly. `flush: 'post'` guarantees the
// new attribute is already in the DOM. Locale changes are watched too: menu
// labels change width, which changes the pill's target size.
watch(
  [() => route.path, expandedItems, locale, () => props.menuItems],
  remeasurePills,
  { flush: 'post' }
)
</script>

<template>
  <div class="w-55 liquid-sidebar h-full flex flex-col shadow-float m-3 mr-0 rounded-surface overflow-hidden">
    <!-- Logo/Brand with Menu label -->
    <div class="p-5 pb-4">
      <!-- Row 1: icon + name -->
      <div class="flex items-center gap-2">
        <img src="/logo.jpg" alt="Sing Box Easy" class="h-7 w-7 rounded-surface flex-shrink-0 shadow-surface" />
        <span class="text-sm font-semibold text-gray-900 dark:text-white">Sing Box Easy</span>
      </div>
      <!-- Row 2 (secondary): app version + update affordance -->
      <div class="flex items-center gap-2 mt-2">
        <span class="text-[10px] font-medium text-gray-400 dark:text-gray-500">{{ version }}</span>
        <!-- "-> vX.Y.Z" update affordance; opens the Settings update panel. -->
        <router-link
          v-if="updateOffered"
          to="/dashboard/settings"
          class="flex items-center gap-1 px-1.5 py-0.5 rounded-pill bg-emerald-500/15 text-[10px] font-semibold text-emerald-700 dark:text-emerald-300 hover:bg-emerald-500/25 transition-colors"
          :title="$t('settings.update.updateTo', { version: latestVersion })"
        >
          <span class="relative flex h-1.5 w-1.5">
            <span class="animate-ping absolute inline-flex h-full w-full rounded-pill bg-emerald-400 opacity-70"></span>
            <span class="relative inline-flex rounded-pill h-1.5 w-1.5 bg-emerald-500"></span>
          </span>
          <span>&rarr; {{ latestVersion }}</span>
        </router-link>
      </div>
      <!-- Row 3: language switch + settings -->
      <div class="flex items-center gap-2 mt-2 mb-4">
        <LanguageSwitcher variant="compact" />
        <router-link
          to="/dashboard/settings"
          class="ml-auto p-1.5 rounded-control transition-colors"
          :class="
            isActive('/dashboard/settings')
              ? 'liquid-item-active'
              : 'liquid-sidebar-muted hover:bg-white/30 dark:hover:bg-white/10'
          "
          :title="$t('nav.settings')"
        >
          <Cog6ToothIcon class="h-4 w-4" />
        </router-link>
      </div>
      <!-- Row 4: live sing-box service status (polled) -->
      <router-link
        to="/dashboard/overview"
        class="liquid-status-link flex items-center gap-2 px-2 py-1.5 rounded-control transition-colors"
        :title="$t('nav.serviceStatusHint')"
      >
        <span class="relative flex h-2.5 w-2.5">
          <span
            v-if="serviceStatus === 'running'"
            class="animate-ping absolute inline-flex h-full w-full rounded-pill bg-green-400 opacity-60"
          ></span>
          <span class="relative inline-flex rounded-pill h-2.5 w-2.5" :class="serviceDotClass"></span>
        </span>
        <span class="text-xs font-semibold liquid-sidebar-text">sing-box</span>
        <span class="text-xs liquid-sidebar-muted ml-auto">{{ serviceLabel }}</span>
      </router-link>
    </div>

    <!-- Menu Items -->
    <nav ref="navRef" class="relative flex-1 px-4 pb-4 overflow-y-auto">
      <!-- Sliding active indicators. These sit behind the rows and carry the
           backgrounds the rows used to paint themselves, so switching pages
           moves one shape instead of cross-fading two. -->
      <div ref="primaryPillRef" class="nav-pill nav-pill-primary" aria-hidden="true"></div>
      <div ref="secondaryPillRef" class="nav-pill nav-pill-secondary" aria-hidden="true"></div>

      <ul ref="navListRef" class="space-y-1">
        <li v-for="item in menuItems" :key="item.name">
          <!-- Parent Item -->
          <div>
            <router-link
              v-if="!item.children"
              :to="item.path || '#'"
              :data-nav-pill="isActive(item.path) ? 'primary' : null"
              :class="[
                'relative z-[1] flex items-center justify-between px-4 py-2.5 rounded-pill text-sm font-medium transition-colors duration-200',
                isActive(item.path)
                  ? 'text-white'
                  : 'liquid-sidebar-text hover:bg-white/40 dark:hover:bg-white/10'
              ]"
            >
              <div class="flex items-center gap-3">
                <component :is="item.icon" class="h-5 w-5 flex-shrink-0" />
                <span>{{ item.name }}</span>
              </div>
              <div v-if="item.badge" class="flex items-center">
                <span
                  :class="[
                    'px-2.5 py-1 text-xs font-semibold rounded-pill',
                    isActive(item.path)
                      ? 'bg-white/25 text-white'
                      : 'bg-white/40 text-gray-900 dark:bg-white/10 dark:text-white'
                  ]"
                >
                  {{ item.badge }}
                </span>
              </div>
            </router-link>

            <!--
              Only the entry matching the current route carries the gradient
              now. Previously a merely-expanded parent got it too, which cannot
              work with a single sliding pill — and painting a menu you are not
              on as "active" was misleading anyway. Expanded-but-inactive gets
              a quieter tint instead.
            -->
            <button
              v-else
              @click="toggleExpanded(item.name)"
              :data-nav-pill="isParentActive(item) ? 'primary' : null"
              :class="[
                'relative z-[1] w-full flex items-center justify-between px-4 py-2.5 rounded-pill text-sm font-medium transition-colors duration-200 cursor-pointer',
                isParentActive(item)
                  ? 'text-white'
                  : isExpanded(item.name)
                    ? 'liquid-sidebar-text bg-white/35 dark:bg-white/8'
                    : 'liquid-sidebar-text hover:bg-white/40 dark:hover:bg-white/10'
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
                    'px-2 py-0.5 text-xs font-medium rounded-pill',
                    isParentActive(item)
                      ? 'bg-white/20 text-white'
                      : 'bg-white/45 text-blue-700 dark:bg-white/10 dark:text-blue-200'
                  ]"
                >
                  {{ item.badge }}
                </span>
                <!--
                  One icon that rotates, not two swapped by v-if. Swapping
                  mounts a fresh element every toggle, so `transition-transform`
                  had nothing to interpolate from and never animated.
                -->
                <ChevronRightIcon
                  class="h-4 w-4 transition-transform duration-300 ease-[cubic-bezier(0.32,0.72,0,1)]"
                  :class="isExpanded(item.name) ? 'rotate-90' : 'rotate-0'"
                />
              </div>
            </button>
          </div>

          <!--
            Children Items.

            The old transition animated `max-height: 0 -> 24rem` while the real
            content is ~5rem tall: the submenu finished moving in the first ~40ms
            of a 200ms transition and then sat still, which reads as a stutter
            (and on collapse, as a delay before it suddenly snaps shut).
            Animating `grid-template-rows: 0fr <-> 1fr` retargets the transition
            at the content's true height, so the whole duration is spent moving.
          -->
          <Transition
            enter-active-class="submenu-transition"
            enter-from-class="submenu-collapsed"
            enter-to-class="submenu-expanded"
            leave-active-class="submenu-transition"
            leave-from-class="submenu-expanded"
            leave-to-class="submenu-collapsed"
          >
            <div v-if="item.children && isExpanded(item.name)" class="submenu-grid ml-8">
              <ul class="submenu-inner space-y-1">
              <li v-for="child in item.children" :key="child.name">
                <router-link
                  :to="child.path || '#'"
                  :data-nav-pill="isActive(child.path) ? 'secondary' : null"
                  :class="[
                    'relative z-[1] flex items-center justify-between px-4 py-2 rounded-pill text-sm transition-colors duration-200',
                    isActive(child.path)
                      ? 'liquid-item-active-text font-semibold'
                      : 'liquid-sidebar-muted hover:text-gray-900 dark:hover:text-white hover:bg-white/30 dark:hover:bg-white/10'
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
                      class="w-4 h-4 rounded-pill border-2 flex-shrink-0"
                      :class="[
                        isActive(child.path)
                          ? 'liquid-item-active-dot'
                          : 'border-gray-400 dark:border-gray-600'
                      ]"
                    />
                    <span>{{ child.name }}</span>
                  </div>
                  <span
                    v-if="child.badge"
                    class="px-2 py-0.5 text-xs font-medium rounded-pill bg-white/35 dark:bg-white/10 liquid-sidebar-muted"
                  >
                    {{ child.badge }}
                  </span>
                </router-link>
              </li>
              </ul>
            </div>
          </Transition>
        </li>
      </ul>
    </nav>

    <!-- Footer/User Section (hidden when authentication is disabled) -->
    <div v-if="authEnabled" class="p-4 border-t border-white/35 dark:border-white/10">
      <div class="flex items-center justify-between gap-2">
        <router-link
          to="/dashboard/profile"
          class="flex items-center gap-3 px-2 py-1.5 rounded-control hover:bg-white/35 dark:hover:bg-white/10 transition-colors flex-1 min-w-0"
          :class="{ 'bg-white/45 dark:bg-white/10': isActive('/dashboard/profile') }"
        >
          <div class="w-8 h-8 rounded-pill bg-gradient-to-br from-blue-500 to-cyan-300 flex items-center justify-center text-white text-sm font-semibold flex-shrink-0 shadow-surface">
            {{ userInitial }}
          </div>
          <div class="flex-1 min-w-0 text-left">
            <p class="text-sm font-semibold liquid-sidebar-text truncate" :title="username">{{ username }}</p>
            <p class="text-[10px] liquid-sidebar-muted capitalize">{{ userRole }}</p>
          </div>
        </router-link>

        <button
          @click="handleLogout"
          class="p-2 rounded-control text-red-500 hover:bg-red-500/10 transition-colors cursor-pointer"
          title="Sign Out"
        >
          <ArrowLeftOnRectangleIcon class="h-4.5 w-4.5" />
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.liquid-sidebar {
  height: calc(100% - 1.5rem);
  background:
    linear-gradient(145deg, rgba(255, 255, 255, 0.48), rgba(255, 255, 255, 0.18)),
    rgba(255, 255, 255, 0.82) !important;
  border: 1px solid rgba(255, 255, 255, 0.58);
  box-shadow: 0 24px 70px rgba(15, 23, 42, 0.16), var(--glass-highlight);
  backdrop-filter: var(--glass-blur);
  -webkit-backdrop-filter: var(--glass-blur);
}

.liquid-sidebar-text {
  color: #172033;
}

.liquid-sidebar-muted {
  color: #64748b;
}

.liquid-status-link {
  background: rgba(255, 255, 255, 0.48);
  border: 1px solid rgba(148, 163, 184, 0.22);
  box-shadow: var(--glass-highlight);
}

.liquid-nav-active {
  background: linear-gradient(135deg, var(--color-primary), var(--color-primary-light));
}

/*
 * Sliding active indicators.
 *
 * These carry the backgrounds the nav rows used to paint on themselves. A
 * `linear-gradient` cannot be interpolated by CSS, so a per-row gradient can
 * only ever snap on/off; moving it to one persistent element and animating
 * `transform` gives a real transition. `transform`/`opacity` are also
 * compositor-driven, so the slide survives the main-thread work of mounting the
 * incoming route — the stutter that made page switches feel laggy.
 *
 * Position/size are written imperatively by useNavIndicator.
 */
.nav-pill {
  position: absolute;
  top: 0;
  left: 0;
  border-radius: 9999px;
  opacity: 0;
  z-index: 0;
  pointer-events: none;
  will-change: transform, width, height;
}

.nav-pill.is-animated {
  transition:
    transform 320ms cubic-bezier(0.32, 0.72, 0, 1),
    width 320ms cubic-bezier(0.32, 0.72, 0, 1),
    height 320ms cubic-bezier(0.32, 0.72, 0, 1),
    opacity 180ms ease;
}

.nav-pill-primary {
  background: linear-gradient(135deg, var(--color-primary), var(--color-primary-light));
  box-shadow: 0 6px 18px rgba(21, 117, 255, 0.28);
}

.nav-pill-secondary {
  background: rgba(21, 117, 255, 0.16);
}

/*
 * Submenu expand/collapse. `grid-template-rows: 0fr -> 1fr` animates to the
 * content's real height, unlike a fixed `max-height` guess. The inner element
 * needs `min-height: 0` for the row to actually shrink below its content.
 */
.submenu-grid {
  display: grid;
  grid-template-rows: 1fr;
}

.submenu-inner {
  min-height: 0;
  overflow: hidden;
  padding-top: 0.5rem;
}

.submenu-transition {
  transition:
    grid-template-rows 300ms cubic-bezier(0.32, 0.72, 0, 1),
    opacity 200ms ease;
}

.submenu-collapsed {
  grid-template-rows: 0fr;
  opacity: 0;
}

.submenu-expanded {
  grid-template-rows: 1fr;
  opacity: 1;
}

@media (prefers-reduced-motion: reduce) {
  .nav-pill.is-animated {
    transition: opacity 120ms ease;
  }

  .submenu-transition {
    transition: opacity 120ms ease;
  }
}

/*
 * Active state for secondary items (sub-menu entries, the settings button).
 *
 * A white wash (the previous `bg-white/45`) is invisible here: the sidebar is
 * already rgba(255,255,255,0.82) under a white gradient, so white-on-white
 * left active and inactive looking identical in light mode. Tint with the
 * primary colour instead, which reads against both themes.
 */
.liquid-item-active {
  background: rgba(21, 117, 255, 0.16);
  color: var(--color-primary-dark);
}

/*
 * Text-only half of `liquid-item-active`, for submenu rows: the background is
 * now painted by the sliding `.nav-pill-secondary` instead of the row itself.
 * `color` is interpolatable, so this still cross-fades cleanly.
 */
.liquid-item-active-text {
  color: var(--color-primary-dark);
}

.liquid-item-active-dot {
  border-color: var(--color-primary);
  background-color: var(--color-primary);
}

@media (prefers-color-scheme: dark) {
  .liquid-item-active {
    background: rgba(107, 165, 255, 0.22);
    color: #cfe2ff;
  }

  .liquid-item-active-text {
    color: #cfe2ff;
  }

  .nav-pill-secondary {
    background: rgba(107, 165, 255, 0.22);
  }

  .liquid-item-active-dot {
    border-color: var(--color-primary-light);
    background-color: var(--color-primary-light);
  }
}

@media (prefers-color-scheme: dark) {
  .liquid-sidebar {
    background:
      linear-gradient(145deg, rgba(255, 255, 255, 0.13), rgba(255, 255, 255, 0.045)),
      rgba(9, 14, 21, 0.94) !important;
    border-right-color: rgba(255, 255, 255, 0.14);
  }

  .liquid-sidebar-text {
    color: #f4f8ff;
  }

  .liquid-sidebar-muted {
    color: #a9b8cc;
  }

  .liquid-status-link {
    background: rgba(255, 255, 255, 0.105);
  }
}

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
