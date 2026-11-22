import { createApp } from 'vue'
import { createRouter, createWebHashHistory } from 'vue-router'
import { loadPlugins } from '../plugins/plugins'
import { installDemoStreams } from './streams'
import { installDemoApi } from './api'
import DemoApp from './DemoApp.vue'
import './style.css'

installDemoApi()
installDemoStreams()
const router = createRouter({
  history: createWebHashHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/', redirect: '/dashboard/overview' },
    { path: '/dashboard/overview', name: 'DashboardOverview', component: () => import('../views/dashboard/Overview.vue') },
    { path: '/dashboard/proxies', name: 'DashboardProxies', component: () => import('../views/dashboard/Proxies.vue') },
    { path: '/dashboard/outbounds/subscriptions', name: 'DashboardSubscriptions', component: () => import('../views/dashboard/Subscriptions.vue') },
    { path: '/dashboard/route/rules', component: () => import('../components/RoutingRules.vue') },
    { path: '/dashboard/dns/servers', component: () => import('../components/DNSServers.vue') },
    { path: '/dashboard/settings', name: 'DashboardSettings', component: () => import('../components/NodeQualitySettingsCard.vue') },
    ...[['log', 'DashboardLog'], ['logs', 'DashboardLogs'], ['config', 'DashboardConfig']].map(([path, name]) => ({ path: `/dashboard/${path}`, name, component: () => import('./DemoUnavailable.vue') })),
    { path: '/:pathMatch(.*)*', component: () => import('./DemoUnavailable.vue') },
  ],
  scrollBehavior: () => ({ top: 0 }),
})
const app = createApp(DemoApp)
loadPlugins(app)
app.use(router)
app.mount('#app')
