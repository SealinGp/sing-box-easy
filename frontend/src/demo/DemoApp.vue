<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { useServiceStore } from '../stores/service'
import Toast from '../volt/Toast.vue'
import ConfirmDialog from '../components/ConfirmDialog.vue'
import { GlobeAltIcon, ServerStackIcon, ShareIcon, SignalIcon, AdjustmentsHorizontalIcon, ArrowTopRightOnSquareIcon } from '@heroicons/vue/24/outline'
const { locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const route = useRoute()
const nav = computed(() => [
  { path: '/dashboard/proxies', label: zh.value ? '代理节点' : 'Proxies', icon: SignalIcon },
  { path: '/dashboard/outbounds/subscriptions', label: zh.value ? '订阅' : 'Subscriptions', icon: ServerStackIcon },
  { path: '/dashboard/route/rules', label: zh.value ? '路由规则' : 'Routing rules', icon: ShareIcon },
  { path: '/dashboard/dns/servers', label: 'DNS', icon: GlobeAltIcon },
  { path: '/dashboard/settings#node-quality', label: zh.value ? '质量标准' : 'Quality standards', icon: AdjustmentsHorizontalIcon },
])
const title = computed(() => nav.value.find(n => n.path.split('#')[0] === route.path)?.label ?? (zh.value ? '演示范围' : 'Demo scope'))
onMounted(() => useServiceStore().fetch())
</script>
<template>
  <div class="min-h-screen bg-bg-primary text-text-primary">
    <a href="#demo-content" class="sr-only focus:not-sr-only focus:absolute focus:z-50 focus:bg-bg-secondary focus:p-3">{{ zh ? '跳到内容' : 'Skip to content' }}</a>
    <header class="border-b border-border bg-bg-secondary">
      <div class="mx-auto flex max-w-screen-2xl flex-wrap items-center gap-3 px-4 py-4 sm:px-6">
        <RouterLink to="/" class="text-lg font-semibold tracking-tight">sing-box-easy</RouterLink>
        <span class="rounded-control bg-primary-50 px-2 py-1 text-xs font-medium text-primary-700">{{ zh ? '静态演示' : 'Static demo' }}</span>
        <div class="ml-auto flex items-center gap-3 text-sm">
          <button type="button" class="rounded-control px-2 py-1 hover:bg-bg-tertiary focus-visible:outline-2 focus-visible:outline-primary-500" @click="locale = zh ? 'en' : 'zh'">{{ zh ? 'English' : '中文' }}</button>
          <a href="https://github.com/SealinGp/sing-box-easy#-安装" class="inline-flex items-center gap-1 rounded-control bg-primary-600 px-3 py-2 text-white hover:bg-primary-700 focus-visible:outline-2 focus-visible:outline-primary-500">
            {{ zh ? '安装指南' : 'Install' }}<ArrowTopRightOnSquareIcon class="h-4 w-4" aria-hidden="true" />
          </a>
        </div>
      </div>
    </header>
    <div class="mx-auto max-w-screen-2xl px-4 py-5 sm:px-6">
      <aside class="mb-5 rounded-surface border border-border bg-bg-secondary px-4 py-3 text-sm text-text-secondary">
        {{ zh ? '安装前先逛一逛。全部为虚构数据，不连接设备。节点切换、模拟测速及质量标准仅在当前页面会话生效，刷新即还原；配置写入、订阅更新等操作需安装后使用。' : 'Explore before installing. All data is fictional; no device is connected. Node selection, simulated tests and quality standards last for this session only and reset on reload. Install the app to write configuration or refresh subscriptions.' }}
      </aside>
      <nav :aria-label="zh ? '演示导航' : 'Demo navigation'" class="mb-6 flex flex-wrap gap-2">
        <RouterLink v-for="item in nav" :key="item.path" :to="item.path" :aria-current="route.path === item.path.split('#')[0] ? 'page' : undefined"
          class="inline-flex min-h-11 items-center gap-2 rounded-control border px-3 py-2 text-sm font-medium focus-visible:outline-2 focus-visible:outline-primary-500"
          :class="route.path === item.path.split('#')[0] ? 'border-primary-500 bg-primary-50 text-primary-700' : 'border-border bg-bg-secondary text-text-secondary hover:bg-bg-tertiary'">
          <component :is="item.icon" class="h-4 w-4" aria-hidden="true" />{{ item.label }}
        </RouterLink>
      </nav>
      <main id="demo-content" tabindex="-1" class="min-w-0">
        <h1 :class="['/dashboard/route/rules', '/dashboard/dns/servers', '/dashboard/settings'].includes(route.path) ? 'sr-only' : 'mb-4 text-2xl font-semibold'">{{ title }}</h1>
        <RouterView />
      </main>
      <footer class="mt-10 border-t border-border py-5 text-xs text-text-secondary">
        sing-box-easy · {{ zh ? '本演示复用真实产品组件，数据仅供界面参考。' : 'Real product components with illustrative data.' }}
        <a href="https://github.com/SealinGp/sing-box-easy" class="ml-2 underline">GitHub</a>
      </footer>
    </div>
    <Toast position="top-center" /><ConfirmDialog />
  </div>
</template>
