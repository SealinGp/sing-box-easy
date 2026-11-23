<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ArrowRightIcon, ServerStackIcon } from '@heroicons/vue/24/outline'
import Card from '../Card.vue'
import Button from '../Button.vue'
import IconButton from '../IconButton.vue'
import StatusIndicator from '../StatusIndicator.vue'
import type { ManagedPolicy, ManagedSnapshot } from '../../services/trafficPolicy'

defineProps<{ snapshot: ManagedSnapshot; editingId?: string; busy: boolean; feedback: string; feedbackPolicyId: string | null }>()
defineEmits<{ edit: [policy: ManagedPolicy]; review: [event: Event, policy: ManagedPolicy] }>()
const { t } = useI18n()
const conditionLabel = (kind: string) => t({
  domain_suffix: 'lab.domainSuffix', domain: 'lab.exactDomain',
  rule_set: 'lab.ruleSet', ip_cidr: 'lab.ipRange',
}[kind] ?? kind)
</script>

<template>
  <div class="space-y-3">
    <slot name="create" />
    <Card v-if="!snapshot.policies.length && editingId === undefined">
      <p>{{ t('managed.noPolicies') }}</p>
    </Card>
    <article v-for="p in snapshot.policies" :key="p.id" class="min-w-0 space-y-3" :data-policy-id="p.id">
      <Card padding="sm">
        <div class="grid min-w-0 gap-3 md:grid-cols-[minmax(0,1fr)_minmax(0,2fr)_minmax(0,1.5fr)_auto] md:items-center">
          <div class="min-w-0">
            <h2 class="break-words text-sm font-semibold">{{ p.name }}</h2>
            <StatusIndicator :variant="p.status === 'synced' ? 'success' : 'warning'" class="mt-1 text-xs">
              {{ t('managed.' + (p.status || 'synced')) }}
            </StatusIndicator>
          </div>
          <div class="min-w-0 space-y-1 text-xs">
            <template v-for="(condition, i) in p.conditions" :key="i">
              <span v-if="i" class="block font-medium text-text-secondary">{{ p.mode === 'and' ? '∧' : '∨' }}</span>
              <div class="flex flex-wrap items-baseline gap-1.5">
                <span class="text-text-secondary">{{ conditionLabel(condition.kind) }} ∈</span>
                <span v-for="value in condition.values" :key="value" class="max-w-full break-all rounded bg-bg-tertiary px-1.5 py-0.5">{{ value }}</span>
              </div>
            </template>
          </div>
          <dl class="min-w-0 space-y-2 text-xs">
            <div class="flex min-w-0 items-start gap-2">
              <dt class="flex shrink-0 items-center gap-1 text-text-secondary"><ArrowRightIcon class="h-3.5 w-3.5" />{{ t('lab.outbound') }}</dt>
              <dd class="min-w-0 break-words font-medium">{{ p.outbound }}</dd>
            </div>
            <div class="flex min-w-0 items-start gap-2">
              <dt class="flex shrink-0 items-center gap-1 text-text-secondary"><ServerStackIcon class="h-3.5 w-3.5" />DNS</dt>
              <dd class="min-w-0 break-words">
                <span class="font-medium">{{ snapshot.groups.find(g => g.id === p.group_id)?.name || p.dns_server || t('lab.leaveDnsUnchanged') }}</span>
                <span v-if="p.group_id" class="mt-0.5 block text-text-secondary">{{ snapshot.groups.find(g => g.id === p.group_id)?.servers.join(' + ') }}</span>
              </dd>
            </div>
          </dl>
          <div class="flex items-center justify-end gap-1">
            <IconButton v-if="p.status === 'synced' || p.status === 'missing_dependency'" action="edit"
              :label="t('lab.editNamed', { name: p.name })" :disabled="busy || editingId !== undefined"
              :aria-expanded="editingId === p.id" @click="$emit('edit', p)" />
            <Button v-if="p.status !== 'synced'" size="sm" variant="secondary" :disabled="busy || editingId !== undefined"
              @click="$emit('review', $event, p)">{{ t('managed.review') }}</Button>
            <IconButton action="delete" :label="t('managed.deleteNamed', { name: p.name })" :disabled="busy || editingId !== undefined"
              @click="$emit('review', $event, p)" />
          </div>
        </div>
      </Card>
      <p v-if="feedback && feedbackPolicyId === p.id" role="status" class="text-sm text-success">{{ feedback }}</p>
      <slot v-if="editingId === p.id" name="editor" />
    </article>
  </div>
</template>
