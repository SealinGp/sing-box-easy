<script setup lang="ts">
import type { Component } from 'vue'
import { useRoute } from 'vue-router'
import logo from '../../public/logo.svg'
import '../style/navigation.css'
defineProps<{
  items: { path: string; label: string; icon: Component }[]
  zh: boolean
  vertical?: boolean
}>()
const route = useRoute()
</script>
<template>
  <aside v-if="vertical" class="navigation-sidebar">
    <RouterLink to="/" class="nav-brand">
      <img :src="logo" alt="" width="30" height="30" />
      <span>Sing Box Easy<span class="nav-brand-caption">{{ zh ? '静态演示' : 'Static demo' }}</span></span>
    </RouterLink>
    <nav class="sidebar-groups" :aria-label="zh ? '演示导航' : 'Demo navigation'">
      <h2 class="nav-group-label">{{ zh ? '浏览功能' : 'Explore' }}</h2>
      <ul>
        <li v-for="item in items" :key="item.path" class="sidebar-item-row">
          <RouterLink :to="item.path" class="nav-destination" :class="{ selected: route.path === item.path.split('#')[0] }"
            :aria-current="route.path === item.path.split('#')[0] ? 'page' : undefined">
            <component :is="item.icon" class="sidebar-item-icon h-4 w-4 shrink-0" aria-hidden="true" />{{ item.label }}
          </RouterLink>
        </li>
      </ul>
    </nav>
    <a href="https://github.com/SealinGp/sing-box-easy" class="nav-destination m-4 text-sm">GitHub ↗</a>
  </aside>
  <nav v-else :aria-label="zh ? '演示导航' : 'Demo navigation'" class="mb-6 flex flex-wrap gap-2">
    <RouterLink v-for="item in items" :key="item.path" :to="item.path" :aria-current="route.path === item.path.split('#')[0] ? 'page' : undefined"
      class="inline-flex min-h-11 items-center gap-2 rounded-control border px-3 py-2 text-sm font-medium focus-visible:outline-2 focus-visible:outline-primary-500"
      :class="route.path === item.path.split('#')[0] ? 'border-primary-500 bg-primary-50 text-primary-700' : 'border-border bg-bg-secondary text-text-secondary hover:bg-bg-tertiary'">
      <component :is="item.icon" class="h-4 w-4" aria-hidden="true" />{{ item.label }}
    </RouterLink>
  </nav>
</template>
