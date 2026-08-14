<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { Component } from "vue";
import Sidebar from "../components/Sidebar.vue";
import Topbar from "../components/Topbar.vue";
import { useDeployment } from "../composables/useDeployment";
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

// OpenWrt routers are administered through LuCI, which already owns a sidebar
// on the left. Stacking a second one there wastes the width these small
// screens do not have, so that platform gets a top bar instead. The platform is
// resolved by the router guard before this view mounts, so the correct layout
// renders on the first paint.
const { isOpenWrt, authEnabled } = useDeployment();

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
      // User management is meaningless where there is no login flow (OpenWrt
      // under `server.auth: auto`): accounts created there could never be used
      // to sign in, and the UI would not say so. The router guard blocks the
      // route too, so a bookmark cannot reach it either.
      ...(authEnabled.value
        ? [
            {
              name: t("nav.users"),
              icon: UsersIcon,
              path: "/dashboard/users",
            },
          ]
        : []),
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
  <!-- OpenWrt: horizontal navigation, content below it. -->
  <div v-if="isOpenWrt" class="liquid-app flex flex-col h-screen overflow-hidden">
    <Topbar :menu-items="menuItems" />

    <main class="flex-1 overflow-auto">
      <router-view />
    </main>
  </div>

  <!-- Everything else: the standard sidebar layout. -->
  <div v-else class="liquid-app flex h-screen">
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
