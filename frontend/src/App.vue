<script setup lang="ts">
import Toast from './volt/Toast.vue'
</script>

<template>
  <div class="min-h-screen bg-gray-50 dark:bg-slate-900">
    <router-view />
    <Toast />
  </div>
</template>
