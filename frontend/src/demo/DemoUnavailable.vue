<script setup lang="ts">
import { useI18n } from 'vue-i18n'
const { locale } = useI18n()
</script>
<template>
  <div class="rounded-surface border border-border bg-bg-secondary p-6">
    <p class="mb-4 text-text-secondary">{{ locale.startsWith('zh') ? '此页面需要安装后连接 sing-box 使用。可以继续浏览上方的演示页面，或查看安装指南。' : 'This page requires an installed sing-box instance. Explore the demo tabs above or read the installation guide.' }}</p>
    <RouterLink to="/dashboard/proxies" class="text-primary-600 underline">{{ locale.startsWith('zh') ? '返回代理节点' : 'Back to proxies' }}</RouterLink>
  </div>
</template>
