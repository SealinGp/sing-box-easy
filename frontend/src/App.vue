<script setup lang="ts">
import Toast from './volt/Toast.vue'
</script>

<template>
  <div class="min-h-screen bg-gray-100 dark:bg-slate-600">
    <router-view />
    <Toast position="top-center" />
  </div>
</template>
