import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { apiService } from '../services/api'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    redirect: '/init',
  },
  {
    path: '/init',
    name: 'Init',
    component: () => import('../views/InitWizard.vue'),
  },
  {
    path: '/dashboard',
    name: 'Dashboard',
    component: () => import('../views/Dashboard.vue'),
    children: [
      {
        path: '',
        redirect: '/dashboard/overview',
      },
      {
        path: 'overview',
        name: 'DashboardOverview',
        component: () => import('../views/dashboard/Overview.vue'),
      },
      {
        path: 'inbounds',
        name: 'DashboardInbounds',
        component: () => import('../views/dashboard/Inbounds.vue'),
      },
      {
        path: 'outbounds',
        name: 'DashboardOutbounds',
        component: () => import('../views/dashboard/Outbounds.vue'),
      },
      {
        path: 'dns',
        name: 'DashboardDNS',
        component: () => import('../views/dashboard/DNS.vue'),
      },
      {
        path: 'route',
        name: 'DashboardRoute',
        component: () => import('../views/dashboard/Route.vue'),
      },
      {
        path: 'subscriptions',
        name: 'DashboardSubscriptions',
        component: () => import('../views/dashboard/Subscriptions.vue'),
      },
      {
        path: 'service',
        name: 'DashboardService',
        component: () => import('../views/dashboard/Service.vue'),
      },
      {
        path: 'config',
        name: 'DashboardConfig',
        component: () => import('../views/dashboard/Config.vue'),
      },
    ],
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

// Navigation guard to check initialization status
router.beforeEach(async (to, _from, next) => {
  // Skip check for init page
  if (to.path.startsWith('/init')) {
    next()
    return
  }

  try {
    const initStatus = await apiService.getInitStatus()

    // If not fully initialized, redirect to init wizard
    if (!initStatus.singbox_installed || !initStatus.config_generated) {
      next('/init')
      return
    }

    next()
  } catch (error) {
    console.error('Failed to check init status:', error)
    // On error, allow navigation but log the error
    next()
  }
})

export default router
