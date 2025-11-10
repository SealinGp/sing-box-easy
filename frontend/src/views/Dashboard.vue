<script setup lang="ts">
const menuItems = [
  { name: 'Overview', icon: '📊', path: '/dashboard/overview' },
  { name: 'Inbounds', icon: '📥', path: '/dashboard/inbounds' },
  { name: 'Outbounds', icon: '📤', path: '/dashboard/outbounds' },
  { name: 'DNS', icon: '🌐', path: '/dashboard/dns' },
  { name: 'Route', icon: '🛣️', path: '/dashboard/route' },
  { name: 'Subscriptions', icon: '📡', path: '/dashboard/subscriptions' },
  { name: 'Service', icon: '⚙️', path: '/dashboard/service' },
  { name: 'Config', icon: '📝', path: '/dashboard/config' },
]
</script>

<template>
  <div class="flex h-screen bg-gray-100">
    <!-- Sidebar -->
    <div class="w-64 bg-white shadow-lg">
      <div class="p-6 border-b">
        <h1 class="text-2xl font-bold text-gray-900">sing-box Easy</h1>
        <p class="text-sm text-gray-500">Management Dashboard</p>
      </div>
      <nav class="p-4">
        <router-link
          v-for="item in menuItems"
          :key="item.path"
          :to="item.path"
          class="flex items-center gap-3 px-4 py-3 mb-2 rounded-lg text-gray-700 hover:bg-blue-50 hover:text-blue-600 transition-colors"
          active-class="bg-blue-50 text-blue-600 font-semibold"
        >
          <span class="text-xl">{{ item.icon }}</span>
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
