<script setup lang="ts">
/**
 * The Overview grid.
 *
 * Every tile is a self-contained card that fetches what it renders, so this
 * view holds no state at all — it only decides what appears and in what order.
 */
import ServiceStatusCard from '../../components/ServiceStatusCard.vue'
import SubscriptionsOverviewCard from '../../components/SubscriptionsOverviewCard.vue'
import DnsProbeCard from '../../components/DnsProbeCard.vue'
import RouteProbeCard from '../../components/RouteProbeCard.vue'
import ApiEndpointsCard from '../../components/ApiEndpointsCard.vue'
</script>

<template>
  <div class="page-shell">
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 items-start">
      <!-- Is sing-box running, and the controls plus the three places to look
           next when it is not -->
      <ServiceStatusCard />

      <!-- Subscriptions: quota, expiry and freshness at a glance -->
      <SubscriptionsOverviewCard />

      <!-- "Where does this domain actually go?" without leaving the dashboard -->
      <DnsProbeCard />

      <!--
        And where would the connection to it go? The pair answers the two
        halves of one question: DNS decides the address, routing decides the
        outbound. Both are predictions made BEFORE any traffic exists, which is
        what makes them useful right after a config edit.
      -->
      <RouteProbeCard />

      <!--
        Where to go when the two predictions above disagree with reality: the
        Clash dashboard lists the connections that actually happened. One click
        from here, because that is when it is wanted.
      -->
      <ApiEndpointsCard />
    </div>
  </div>
</template>
