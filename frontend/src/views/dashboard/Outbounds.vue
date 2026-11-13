<script setup lang="ts">
import PageTransition from '../../components/PageTransition.vue'
</script>

<template>
  <!-- These are direct menu destinations now; preserve their nested URLs. -->
  <div class="page-shell">
    <RouterView v-slot="{ Component, route }">
      <PageTransition :view-key="route.path"><component :is="Component" /></PageTransition>
    </RouterView>
  </div>
</template>
