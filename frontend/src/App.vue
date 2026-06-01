<script setup lang="ts">
import Toast from './volt/Toast.vue'
import ConfirmDialog from './components/ConfirmDialog.vue'
</script>

<template>
  <div class="min-h-screen bg-gray-100 dark:bg-slate-600">
    <router-view />
    <Toast position="top-center" />
    <ConfirmDialog />
  </div>
</template>
