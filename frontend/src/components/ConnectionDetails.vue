<script setup lang="ts">
/**
 * One connection in full — everything its row had to truncate.
 *
 * Rendered INSIDE the table, directly under the row that was clicked (see
 * ConnectionsTable). It used to be a drawer on the right edge: click in the
 * middle of the screen, read the answer at the far side. The detail belongs to
 * the row, so it opens at the row.
 */
import { displayName } from '../utils/proxyGroups'
import { formatBytes } from '../utils/formatBytes'
import { formatRate } from '../utils/flowOverlay'
import { durationOf, formatDuration, type TrackedConnection } from '../utils/connectionsTable'
import { FOCUS_PAGES, focusLink, splitRuleByRuleSets } from '../utils/focusTarget'
import { useConnectionLinks } from '../composables/useConnectionLinks'

defineProps<{
  connection: TrackedConnection
  label: string
  now: number
  closing: boolean
}>()

const emit = defineEmits<{ (e: 'dismiss'): void; (e: 'close'): void }>()

const { ruleSetTags, isGroup } = useConnectionLinks()
</script>

<template>
  <div class="detail" :aria-label="$t('connections.details.title')">
    <dl>
      <div>
        <dt>{{ $t('connections.details.source') }}</dt>
        <dd class="tabular-nums"><template v-if="label">{{ label }} · </template>{{ connection.source_ip }}:{{ connection.source_port }}</dd>
      </div>
      <div>
        <dt>{{ $t('connections.details.destination') }}</dt>
        <dd class="tabular-nums">{{ connection.destination_ip || '–' }}:{{ connection.destination_port }}</dd>
      </div>
      <div>
        <dt>{{ $t('connections.details.inbound') }}</dt>
        <dd>{{ connection.inbound }} · {{ connection.network }}</dd>
      </div>
      <div>
        <dt>{{ $t('connections.details.speed') }}</dt>
        <dd class="tabular-nums">↓ {{ formatRate(connection.down_rate) }}/s · ↑ {{ formatRate(connection.up_rate) }}/s</dd>
      </div>
      <div>
        <dt>{{ $t('connections.details.traffic') }}</dt>
        <dd class="tabular-nums">↓ {{ formatBytes(connection.download) }} · ↑ {{ formatBytes(connection.upload) }}</dd>
      </div>
      <div>
        <dt>{{ $t('connections.details.duration') }}</dt>
        <dd class="tabular-nums">{{ formatDuration(durationOf(connection, now)) }}</dd>
      </div>
      <div v-if="connection.process_path">
        <dt>{{ $t('connections.details.process') }}</dt>
        <dd class="font-mono text-xs">{{ connection.process_path }}</dd>
      </div>
      <div class="wide">
        <dt>{{ $t('connections.details.chain') }}</dt>
        <dd>
          <template v-for="(hop, index) in connection.chains" :key="index">
            <span v-if="index" class="text-gray-400"> → </span>
            <!--
              Only the FIRST hop — the outbound the rule named — and only when
              it is a group: that is what has a card on the Proxies page. The
              hops after it are that group's members, shown inside its card.
            -->
            <RouterLink
              v-if="index === 0 && isGroup(hop)"
              :to="focusLink(FOCUS_PAGES.proxies, hop)"
              :title="$t('connections.details.openProxyGroup', { name: hop })"
              class="detail-link"
            >{{ displayName(hop) }}</RouterLink>
            <span v-else :title="hop">{{ displayName(hop) }}</span>
          </template>
        </dd>
      </div>
      <div class="wide">
        <dt>{{ $t('connections.details.rule') }}</dt>
        <dd class="font-mono text-xs">
          <template v-for="(part, index) in splitRuleByRuleSets(connection.rule, ruleSetTags)" :key="index">
            <RouterLink
              v-if="part.ruleSet"
              :to="focusLink(FOCUS_PAGES.ruleSets, part.ruleSet)"
              :title="$t('connections.details.openRuleSet', { name: part.ruleSet })"
              class="detail-link"
            >{{ part.text }}</RouterLink>
            <template v-else>{{ part.text }}</template>
          </template>
        </dd>
      </div>
    </dl>
    <div class="flex flex-wrap gap-2">
      <button
        v-if="!connection.closed"
        type="button"
        class="rounded-control border border-red-400 px-3 py-1 text-xs font-medium text-red-600 hover:bg-red-50 disabled:cursor-wait disabled:opacity-60 dark:hover:bg-red-950/40"
        :disabled="closing"
        @click="emit('close')"
      >
        {{ $t('connections.details.close') }}
      </button>
      <button
        type="button"
        class="rounded-control border border-gray-300 px-3 py-1 text-xs text-gray-700 hover:border-primary-500 dark:border-gray-600 dark:text-gray-300"
        @click="emit('dismiss')"
      >
        {{ $t('connections.details.dismiss') }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.detail {
  display: flex;
  flex-direction: column;
  gap: 10px;
  white-space: normal;
}
dl {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 8px 20px;
  margin: 0;
}
dl > .wide {
  grid-column: 1 / -1;
}
dt {
  font-size: 11px;
  color: var(--color-text-secondary);
}
/* A name that leads to its card on another page. */
.detail-link {
  color: var(--color-primary);
  text-decoration: underline;
  text-decoration-color: color-mix(in srgb, var(--color-primary) 40%, transparent);
  text-underline-offset: 2px;
}
.detail-link:hover {
  text-decoration-color: currentColor;
}
dd {
  min-width: 0;
  margin: 0;
  overflow-wrap: anywhere;
  font-size: 13px;
  color: var(--color-text-primary);
}
</style>
