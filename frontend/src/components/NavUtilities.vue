<script setup lang="ts">
import { useDeployment } from '../composables/useDeployment'
import { useNavChrome } from '../composables/useNavChrome'
import LanguageSwitcher from './LanguageSwitcher.vue'
import { ArrowLeftOnRectangleIcon, ArrowUpRightIcon } from '@heroicons/vue/24/outline'
defineProps<{ compact?: boolean }>()
const { authEnabled } = useDeployment()
const { version, latestVersion, updateOffered, username, userInitial, serviceDotClass, serviceLabel, handleLogout } = useNavChrome()
</script>

<template>
  <div class="nav-utilities" :class="{ compact }">
    <router-link to="/dashboard/overview" class="nav-health" :aria-label="'sing-box: ' + serviceLabel" :title="$t('nav.serviceStatusHint')">
      <span class="h-2 w-2 rounded-full shrink-0" :class="serviceDotClass" />
      <span class="health-engine">sing-box</span>
      <span class="health-label">{{ serviceLabel }}</span>
    </router-link>
    <div class="nav-utility-row">
      <router-link v-if="updateOffered" to="/dashboard/settings" class="nav-version update" :title="$t('settings.update.updateTo', { version: latestVersion })">
        {{ latestVersion }} <ArrowUpRightIcon class="h-3 w-3" />
      </router-link>
      <span v-else class="nav-version">{{ version }}</span>
      <LanguageSwitcher variant="compact" />
      <template v-if="authEnabled">
        <router-link to="/dashboard/profile" class="nav-avatar" :aria-label="username" :title="username">{{ userInitial }}</router-link>
        <button class="nav-icon-button" @click="handleLogout" :aria-label="$t('nav.signOut')" :title="$t('nav.signOut')"><ArrowLeftOnRectangleIcon class="h-4 w-4" /></button>
      </template>
    </div>
  </div>
</template>
