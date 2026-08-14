import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useServiceStore } from '../stores'
import { userService } from '../services'
import { useAppUpdate } from './useAppUpdate'
import { useConfirm } from './useConfirm'
import type { User } from '../types/api'

/**
 * Everything the navigation shell displays besides the menu itself: the live
 * sing-box status dot, the app version with its update affordance, and the
 * signed-in user.
 *
 * Two shells render these facts — the sidebar on servers and the top bar on
 * OpenWrt routers, where LuCI already owns the left edge of the screen. Only
 * the arrangement differs, so the behaviour lives here instead of being
 * duplicated in both components.
 *
 * Must be called from a component's `setup`: it registers polling on mount and
 * tears it down on unmount.
 */
export const useNavChrome = () => {
  const { t, locale } = useI18n()
  const router = useRouter()
  const { confirm } = useConfirm()

  // App version. The bundle carries a build-time stamp (see vite.config.ts
  // `define`), but the backend is the authority once it reports its own
  // version — after a self-update the two agree again.
  const buildVersion = __APP_VERSION__
  const { currentVersion, latestVersion, updateOffered, refreshStatus } = useAppUpdate()
  const version = computed(() => currentVersion.value || buildVersion)

  // Shared service status, polled here so the running/stopped indicator stays
  // live on every page (e.g. right after restarting from the Config page).
  const serviceStore = useServiceStore()
  let unsubscribe: (() => void) | null = null

  const currentUser = ref<User | null>(null)

  const fetchUser = async () => {
    try {
      currentUser.value = await userService.getMe()
    } catch (err) {
      console.error('Failed to get current user for navigation:', err)
    }
  }

  onMounted(async () => {
    unsubscribe = serviceStore.startPolling(5000)
    // Started before awaiting the user so it overlaps with the update panel's
    // own mount-time call and the two collapse into one request. Awaiting first
    // would let that call finish before this one begins, defeating the
    // de-duplication whenever /user/me happens to be the slower of the two.
    // Cached server-side for a few minutes, so this is cheap and non-blocking.
    void refreshStatus(false)
    await fetchUser()
  })

  onUnmounted(() => {
    unsubscribe?.()
  })

  const userInitial = computed(() => {
    if (!currentUser.value?.username) return 'U'
    return currentUser.value.username.slice(0, 2).toUpperCase()
  })

  const username = computed(() => currentUser.value?.username || 'User')
  const userRole = computed(() => currentUser.value?.role || 'viewer')

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

  const handleLogout = async () => {
    const ok = await confirm({
      title: t('common.confirmTitle'),
      message: locale.value.startsWith('zh')
        ? '确定要退出登录吗？'
        : 'Are you sure you want to sign out?',
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

  return {
    version,
    latestVersion,
    updateOffered,
    currentUser,
    userInitial,
    username,
    userRole,
    serviceStatus,
    serviceDotClass,
    serviceLabel,
    handleLogout,
  }
}
