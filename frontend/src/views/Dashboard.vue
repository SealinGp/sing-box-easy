<script setup lang="ts">
import type { Component } from "vue";
import Sidebar from "../components/Sidebar.vue";
import {
  ChartBarIcon,
  ArrowDownTrayIcon,
  ArrowUpTrayIcon,
  GlobeAltIcon,
  MapIcon,
  DocumentTextIcon,
  BeakerIcon,
  ServerIcon,
  ShieldCheckIcon,
  CloudArrowDownIcon,
  CogIcon,
} from "@heroicons/vue/24/outline";

interface MenuItem {
  name: string;
  icon: Component;
  path?: string;
  badge?: number | string;
  children?: MenuItem[];
}

// Static menu definition — never mutated. Plain const avoids unnecessary
// reactive tracking that `ref()` would impose.
const menuItems: MenuItem[] = [
  {
    name: "Overview",
    icon: ChartBarIcon,
    path: "/dashboard/overview",
  },
  {
    name: "Proxy",
    icon: ServerIcon,
    children: [
      {
        name: "Inbounds",
        icon: ArrowDownTrayIcon,
        path: "/dashboard/inbounds",
      },
      {
        name: "Outbounds",
        icon: ArrowUpTrayIcon,
        path: "/dashboard/outbounds",
      },
    ],
  },
  {
    name: "Network",
    icon: ShieldCheckIcon,
    children: [
      { name: "DNS", icon: GlobeAltIcon, path: "/dashboard/dns" },
      { name: "Route", icon: MapIcon, path: "/dashboard/route" },
    ],
  },
  {
    name: "Advanced",
    icon: CogIcon,
    children: [
      {
        name: "Experimental",
        icon: BeakerIcon,
        path: "/dashboard/experimental",
      },
      { name: "Config", icon: DocumentTextIcon, path: "/dashboard/config" },
    ],
  },
  {
    name: "Subscriptions",
    icon: CloudArrowDownIcon,
    path: "/dashboard/subscriptions",
  },
];
</script>

<template>
  <div
    class="flex h-screen bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-slate-900"
  >
    <!-- Beautiful Sidebar Component with backdrop -->
    <div class="relative">
      <Sidebar :menu-items="menuItems" />
    </div>

    <!-- Main Content -->
    <div class="flex-1 flex flex-col overflow-hidden">
      <!-- Main Content Area -->
      <main class="flex-1 overflow-auto">
        <router-view />
      </main>
    </div>
  </div>
</template>
