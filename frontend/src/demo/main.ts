import { createApp } from 'vue'
import { createRouter, createWebHashHistory } from 'vue-router'
import { loadPlugins } from '../plugins/plugins'
import { installDemoApi } from './api'
import DemoApp from './DemoApp.vue'
import './style.css'

installDemoApi()
const router = createRouter({
  history: createWebHashHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/', redirect: '/dashboard/proxies' },
    { path: '/dashboard/proxies', component: () => import('../views/dashboard/Proxies.vue') },
    { path: '/dashboard/outbounds/subscriptions', component: () => import('../views/dashboard/Subscriptions.vue') },
    { path: '/dashboard/route/rules', component: () => import('../components/RoutingRules.vue') },
    { path: '/dashboard/dns/servers', component: () => import('../components/DNSServers.vue') },
    { path: '/dashboard/settings', component: () => import('../components/NodeQualitySettingsCard.vue') },
    { path: '/:pathMatch(.*)*', component: () => import('./DemoUnavailable.vue') },
  ],
  scrollBehavior: () => ({ top: 0 }),
})
const app = createApp(DemoApp)
loadPlugins(app)
app.use(router)
app.mount('#app')
