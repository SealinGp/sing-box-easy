<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { useServiceStore } from '../stores/service'
import Toast from '../volt/Toast.vue'
import DemoNavigation from './DemoNavigation.vue'
import ConfirmDialog from '../components/ConfirmDialog.vue'
import { ChartBarIcon, GlobeAltIcon, ServerStackIcon, ShareIcon, SignalIcon, AdjustmentsHorizontalIcon, ArrowTopRightOnSquareIcon } from '@heroicons/vue/24/outline'
const { locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const route = useRoute()
const layout = ref<'sidebar' | 'topbar'>('sidebar')
try { if (localStorage.getItem('sb_demo_layout') === 'topbar') layout.value = 'topbar' } catch { /* Storage is optional. */ }
watch(layout, value => { try { localStorage.setItem('sb_demo_layout', value) } catch { /* Keep the session choice. */ } })
const nav = computed(() => [
  { path: '/dashboard/overview', label: zh.value ? '概览' : 'Overview', icon: ChartBarIcon },
  { path: '/dashboard/proxies', label: zh.value ? '代理节点' : 'Proxies', icon: SignalIcon },
  { path: '/dashboard/outbounds/subscriptions', label: zh.value ? '订阅' : 'Subscriptions', icon: ServerStackIcon },
  { path: '/dashboard/route/rules', label: zh.value ? '路由规则' : 'Routing rules', icon: ShareIcon },
  { path: '/dashboard/dns/servers', label: 'DNS', icon: GlobeAltIcon },
  { path: '/dashboard/settings#node-quality', label: zh.value ? '质量标准' : 'Quality standards', icon: AdjustmentsHorizontalIcon },
])
const title = computed(() => nav.value.find(n => n.path.split('#')[0] === route.path)?.label ?? (zh.value ? '演示范围' : 'Demo scope'))
function skipToContent() { document.getElementById('demo-content')?.focus() }
onMounted(() => useServiceStore().fetch())
</script>
<template>
  <div class="dashboard-shell bg-bg-primary text-text-primary">
    <div v-if="layout === 'sidebar'" class="hidden shrink-0 lg:flex"><DemoNavigation :items="nav" :zh="zh" vertical /></div>
    <div class="dashboard-content">
    <a href="#demo-content" @click.prevent="skipToContent" class="sr-only focus:not-sr-only focus:absolute focus:z-50 focus:bg-bg-secondary focus:p-3">{{ zh ? '跳到内容' : 'Skip to content' }}</a>
    <header class="border-b border-border bg-bg-secondary">
      <div class="mx-auto flex max-w-screen-2xl flex-wrap items-center gap-3 px-4 py-4 sm:px-6">
        <RouterLink to="/" class="text-lg font-semibold tracking-tight">sing-box-easy</RouterLink>
        <span class="rounded-control bg-primary-50 px-2 py-1 text-xs font-medium text-primary-700">{{ zh ? '静态演示' : 'Static demo' }}</span>
        <div class="ml-auto flex flex-wrap items-center gap-3 text-sm">
          <fieldset class="hidden items-center gap-1 rounded-control border border-border p-1 lg:flex">
            <legend class="sr-only">{{ zh ? '导航布局' : 'Navigation layout' }}</legend>
            <label v-for="option in (['sidebar', 'topbar'] as const)" :key="option" class="cursor-pointer rounded-control px-3 py-2 has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-primary-500"
              :class="layout === option ? 'bg-primary-50 text-primary-700' : 'text-text-secondary hover:bg-bg-tertiary'">
              <input v-model="layout" type="radio" name="demo-layout" :value="option" class="sr-only" />
              {{ option === 'sidebar' ? (zh ? '侧边栏' : 'Sidebar') : (zh ? '顶部导航' : 'Top navigation') }}
            </label>
          </fieldset>
          <button type="button" class="rounded-control px-2 py-1 hover:bg-bg-tertiary focus-visible:outline-2 focus-visible:outline-primary-500" @click="locale = zh ? 'en' : 'zh'">{{ zh ? 'English' : '中文' }}</button>
          <a href="https://github.com/SealinGp/sing-box-easy#-安装" class="inline-flex items-center gap-1 rounded-control bg-primary-600 px-3 py-2 text-white hover:bg-primary-700 focus-visible:outline-2 focus-visible:outline-primary-500">
            {{ zh ? '安装指南' : 'Install' }}<ArrowTopRightOnSquareIcon class="h-4 w-4" aria-hidden="true" />
          </a>
        </div>
      </div>
    </header>
    <div class="mx-auto max-w-screen-2xl px-4 py-5 sm:px-6">
      <DemoNavigation :items="nav" :zh="zh" :class="{ 'lg:hidden': layout === 'sidebar' }" />
      <main id="demo-content" tabindex="-1" class="min-w-0">
        <h1 :class="['/dashboard/route/rules', '/dashboard/dns/servers', '/dashboard/settings'].includes(route.path) ? 'sr-only' : 'mb-4 text-2xl font-semibold'">{{ title }}</h1>
        <RouterView />
      </main>
      <footer class="mt-10 border-t border-border py-5 text-xs text-text-secondary">
        sing-box-easy · {{ zh ? '本演示复用真实产品组件，数据仅供界面参考。' : 'Real product components with illustrative data.' }}
        <a href="https://github.com/SealinGp/sing-box-easy" class="ml-2 underline">GitHub</a>
      </footer>
    </div>
    </div>
    <Toast position="top-center" /><ConfirmDialog />
  </div>
</template>
