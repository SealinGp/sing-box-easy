<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
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
  CogIcon,
  QueueListIcon,
  UsersIcon,
  AdjustmentsHorizontalIcon,
  AdjustmentsVerticalIcon,
} from "@heroicons/vue/24/outline";

interface MenuItem {
  name: string;
  icon: Component;
  path?: string;
  badge?: number | string;
  children?: MenuItem[];
}

const { t } = useI18n();

// Computed so labels re-translate when the locale changes. Icons/paths are
// static; only the display `name` is localized.
const menuItems = computed<MenuItem[]>(() => [
  {
    name: t("nav.overview"),
    icon: ChartBarIcon,
    path: "/dashboard/overview",
  },
  {
    name: t("nav.proxy"),
    icon: ServerIcon,
    children: [
      {
        name: t("nav.inbounds"),
        icon: ArrowDownTrayIcon,
        path: "/dashboard/inbounds",
      },
      {
        name: t("nav.outbounds"),
        icon: ArrowUpTrayIcon,
        path: "/dashboard/outbounds",
      },
    ],
  },
  {
    name: t("nav.network"),
    icon: ShieldCheckIcon,
    children: [
      { name: t("nav.dns"), icon: GlobeAltIcon, path: "/dashboard/dns" },
      { name: t("nav.route"), icon: MapIcon, path: "/dashboard/route" },
    ],
  },
  {
    name: t("nav.advanced"),
    icon: AdjustmentsHorizontalIcon,
    children: [
      {
        name: t("nav.experimental"),
        icon: BeakerIcon,
        path: "/dashboard/experimental",
      },
      { name: t("nav.config"), icon: DocumentTextIcon, path: "/dashboard/config" },
    ],
  },
  {
    name: t("nav.settings"),
    icon: CogIcon,
    children: [
      {
        name: t("nav.general"),
        icon: AdjustmentsVerticalIcon,
        path: "/dashboard/settings",
      },
      {
        name: t("nav.users"),
        icon: UsersIcon,
        path: "/dashboard/users",
      },
      {
        name: t("nav.logs"),
        icon: QueueListIcon,
        path: "/dashboard/logs",
      },
    ],
  },
]);
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
