<script setup lang="ts">
import type { Component } from 'vue'
import {
  ChartBarIcon,
  ArrowDownTrayIcon,
  ArrowUpTrayIcon,
  GlobeAltIcon,
  MapIcon,
  SignalIcon,
  Cog6ToothIcon,
  DocumentTextIcon,
} from '@heroicons/vue/24/outline'

interface MenuItem {
  name: string
  icon: Component
  path: string
}

const menuItems: MenuItem[] = [
  { name: 'Overview', icon: ChartBarIcon, path: '/dashboard/overview' },
  { name: 'Inbounds', icon: ArrowDownTrayIcon, path: '/dashboard/inbounds' },
  { name: 'Outbounds', icon: ArrowUpTrayIcon, path: '/dashboard/outbounds' },
  { name: 'DNS', icon: GlobeAltIcon, path: '/dashboard/dns' },
  { name: 'Route', icon: MapIcon, path: '/dashboard/route' },
  { name: 'Subscriptions', icon: SignalIcon, path: '/dashboard/subscriptions' },
  { name: 'Service', icon: Cog6ToothIcon, path: '/dashboard/service' },
  { name: 'Config', icon: DocumentTextIcon, path: '/dashboard/config' },
]
</script>

<template>
  <div class="flex h-screen bg-gray-100 dark:bg-bg-dark">

    <!-- Sidebar -->
    <div class="w-64 bg-gray-100 dark:bg-bg-dark shadow-lg">
      <div class="p-6 border-b">
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">sing-box Easy</h1>
        <p class="text-sm text-gray-500">Management Dashboard</p>
      </div>
      <nav class="p-4">
        <router-link
          v-for="item in menuItems"
          :key="item.path"
          :to="item.path"
          class="flex items-center gap-3 px-4 py-3 mb-2 rounded-lg text-gray-700 dark:text-gray-300 hover:bg-blue-50 hover:text-blue-600 transition-colors"
          active-class="bg-blue-500 text-blue-600"
          
        >
          <component :is="item.icon" class="h-5 w-5" />
          <span>{{ item.name }}</span>
        </router-link>
      </nav>
    </div>

    <!-- Main Content -->
    <div class="flex-1 overflow-auto">
      <router-view />
    </div>
  </div>
</template>
