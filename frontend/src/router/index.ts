import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { serviceControlService } from '../services'
import { ensureDeployment, useDeployment } from '../composables/useDeployment'

// Eagerly load critical components
import Dashboard from '../views/Dashboard.vue'

// Lazy load routes with meaningful chunk names
const InitWizard = () => import(/* webpackChunkName: "init-wizard" */ '../views/InitWizard.vue')
const Overview = () => import(/* webpackChunkName: "overview" */ '../views/dashboard/Overview.vue')
const Inbounds = () => import(/* webpackChunkName: "inbounds" */ '../views/dashboard/Inbounds.vue')
const Outbounds = () => import(/* webpackChunkName: "outbounds" */ '../views/dashboard/Outbounds.vue')
const OutboundsList = () => import(/* webpackChunkName: "outbounds-list" */ '../components/OutboundsList.vue')
const DNS = () => import(/* webpackChunkName: "dns" */ '../views/dashboard/DNS.vue')
const DNSServers = () => import(/* webpackChunkName: "dns-servers" */ '../components/DNSServers.vue')
const DNSRules = () => import(/* webpackChunkName: "dns-rules" */ '../components/DNSRules.vue')
const DNSSettings = () => import(/* webpackChunkName: "dns-settings" */ '../components/DNSSettings.vue')
const Route = () => import(/* webpackChunkName: "route" */ '../views/dashboard/Route.vue')
const RoutingRules = () => import(/* webpackChunkName: "routing-rules" */ '../components/RoutingRules.vue')
const RuleSets = () => import(/* webpackChunkName: "rule-sets" */ '../components/RuleSets.vue')
const FinalPolicy = () => import(/* webpackChunkName: "final-policy" */ '../components/FinalPolicy.vue')
const Experimental = () => import(/* webpackChunkName: "experimental" */ '../views/dashboard/Experimental.vue')
const CacheFileSettings = () => import(/* webpackChunkName: "cache-file" */ '../components/CacheFileSettings.vue')
const ClashAPISettings = () => import(/* webpackChunkName: "clash-api" */ '../components/ClashAPISettings.vue')
const V2RayAPISettings = () => import(/* webpackChunkName: "v2ray-api" */ '../components/V2RayAPISettings.vue')
const Subscriptions = () => import(/* webpackChunkName: "subscriptions" */ '../views/dashboard/Subscriptions.vue')
const NodeRules = () => import(/* webpackChunkName: "node-rules" */ '../views/dashboard/NodeRules.vue')
const Config = () => import(/* webpackChunkName: "config" */ '../views/dashboard/Config.vue')
const Log = () => import(/* webpackChunkName: "log" */ '../views/dashboard/Log.vue')
const Logs = () => import(/* webpackChunkName: "logs" */ '../views/dashboard/Logs.vue')
const Users = () => import(/* webpackChunkName: "users" */ '../views/dashboard/Users.vue')
const Settings = () => import(/* webpackChunkName: "settings" */ '../views/dashboard/Settings.vue')
const Login = () => import(/* webpackChunkName: "login" */ '../views/Login.vue')
const Profile = () => import(/* webpackChunkName: "profile" */ '../views/dashboard/Profile.vue')

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    redirect: '/init',
  },
  {
    path: '/login',
    name: 'Login',
    component: Login,
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
        redirect: '/dashboard/outbounds/list',
        children: [
          {
            path: 'list',
            name: 'OutboundsList',
            component: OutboundsList,
          },
          {
            path: 'subscriptions',
            name: 'DashboardSubscriptions',
            component: Subscriptions,
          },
          {
            path: 'node-rules',
            name: 'DashboardNodeRules',
            component: NodeRules,
          },
        ],
      },
      {
        path: 'dns',
        name: 'DashboardDNS',
        component: DNS,
        redirect: '/dashboard/dns/servers',
        children: [
          {
            path: 'servers',
            name: 'DNSServers',
            component: DNSServers,
          },
          {
            path: 'rules',
            name: 'DNSRules',
            component: DNSRules,
          },
          {
            path: 'settings',
            name: 'DNSSettings',
            component: DNSSettings,
          },
        ],
      },
      {
        path: 'route',
        name: 'DashboardRoute',
        component: Route,
        redirect: '/dashboard/route/rules',
        children: [
          {
            path: 'rules',
            name: 'RoutingRules',
            component: RoutingRules,
          },
          {
            path: 'rule-sets',
            name: 'RuleSets',
            component: RuleSets,
          },
          {
            path: 'final-policy',
            name: 'FinalPolicy',
            component: FinalPolicy,
          },
        ],
      },
      {
        path: 'experimental',
        name: 'DashboardExperimental',
        component: Experimental,
        redirect: '/dashboard/experimental/cache-file',
        children: [
          {
            path: 'cache-file',
            name: 'CacheFileSettings',
            component: CacheFileSettings,
          },
          {
            path: 'clash-api',
            name: 'ClashAPISettings',
            component: ClashAPISettings,
          },
          {
            path: 'v2ray-api',
            name: 'V2RayAPISettings',
            component: V2RayAPISettings,
          },
        ],
      },
      // Subscriptions and Node Rules are now tabs under Outbounds. Keep the
      // old top-level paths as redirects so existing bookmarks keep working.
      {
        path: 'subscriptions',
        redirect: '/dashboard/outbounds/subscriptions',
      },
      {
        path: 'node-rules',
        redirect: '/dashboard/outbounds/node-rules',
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
      {
        path: 'logs',
        name: 'DashboardLogs',
        component: Logs,
      },
      {
        path: 'users',
        name: 'DashboardUsers',
        component: Users,
      },
      {
        path: 'settings',
        name: 'DashboardSettings',
        component: Settings,
      },
      {
        path: 'profile',
        name: 'DashboardProfile',
        component: Profile,
      },
    ],
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

// Navigation guard to check authentication and initialization state
router.beforeEach(async (to, _, next) => {
  const token = localStorage.getItem('sb_token')

  // 0. When the deployment has authentication disabled (server.auth:
  //    disabled, or "auto" on OpenWrt) there is no login flow and no
  //    profile/user management — skip the token checks entirely.
  //    Resolving it here (rather than in the layout) also means the platform
  //    is already known by the time Dashboard mounts, so the sidebar/top-bar
  //    choice never flashes the wrong layout.
  await ensureDeployment()
  const { authEnabled } = useDeployment()
  if (!authEnabled.value) {
    if (to.path === '/login' || to.path === '/dashboard/profile') {
      next('/dashboard')
      return
    }
  } else {
    // 1. Allow login page without auth
    if (to.path === '/login') {
      if (token) {
        next('/dashboard')
      } else {
        next()
      }
      return
    }

    // 2. Require auth for all other routes
    if (!token) {
      next('/login')
      return
    }
  }

  // 3. If authenticated, check initialization status
  if (!to.path.startsWith('/init')) {
    try {
      const { data } = await serviceControlService.getInitStatus()
      if (!data.initialized) {
        next('/init')
        return
      }
    } catch (err) {
      console.error('Failed to check init state:', err)
    }
  }
  next()
})

export default router