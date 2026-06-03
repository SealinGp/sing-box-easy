import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { serviceControlService } from '../services'

// Eagerly load critical components
import Dashboard from '../views/Dashboard.vue'

// Lazy load routes with meaningful chunk names
const InitWizard = () => import(/* webpackChunkName: "init-wizard" */ '../views/InitWizard.vue')
const Overview = () => import(/* webpackChunkName: "overview" */ '../views/dashboard/Overview.vue')
const Inbounds = () => import(/* webpackChunkName: "inbounds" */ '../views/dashboard/Inbounds.vue')
const Outbounds = () => import(/* webpackChunkName: "outbounds" */ '../views/dashboard/Outbounds.vue')
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
const Settings = () => import(/* webpackChunkName: "settings" */ '../views/dashboard/Settings.vue')

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
        path: 'settings',
        name: 'DashboardSettings',
        component: Settings,
      },
    ],
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

// Navigation guard to check initialization state
router.beforeEach(async (to, _, next) => {
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