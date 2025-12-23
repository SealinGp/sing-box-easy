import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import InitWizard from '../views/InitWizard.vue'
import Dashboard from '../views/Dashboard.vue'
import Overview from '../views/dashboard/Overview.vue'
import Inbounds from '../views/dashboard/Inbounds.vue'
import Outbounds from '../views/dashboard/Outbounds.vue'
import DNS from '../views/dashboard/DNS.vue'
import Route from '../views/dashboard/Route.vue'
import Subscriptions from '../views/dashboard/Subscriptions.vue'
import Config from '../views/dashboard/Config.vue'
import Log from '../views/dashboard/Log.vue'
import { serviceControlService } from '../services'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    redirect: '/init',
  },
  {
    path: '/init',
    name: 'Init',
    component: InitWizard,
  },
  {
    path: '/dashboard',
    name: 'Dashboard',
    component: Dashboard,
    children: [
      {
        path: '',
        redirect: '/dashboard/overview',
      },
      {
        path: 'overview',
        name: 'DashboardOverview',
        component: Overview,
      },
      {
        path: 'inbounds',
        name: 'DashboardInbounds',
        component: Inbounds,
      },
      {
        path: 'outbounds',
        name: 'DashboardOutbounds',
        component: Outbounds,
      },
      {
        path: 'dns',
        name: 'DashboardDNS',
        component: DNS,
      },
      {
        path: 'route',
        name: 'DashboardRoute',
        component: Route,
      },
      {
        path: 'subscriptions',
        name: 'DashboardSubscriptions',
        component: Subscriptions,
      },
      {
        path: 'config',
        name: 'DashboardConfig',
        component: Config,
      },
      {
        path: 'log',
        name: 'DashboardLog',
        component: Log,
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
    const { data } = await serviceControlService.getInitStatus()
    const initStatus = data

    // If not fully initialized, redirect to init wizard
    if (!initStatus.initialized) {
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
