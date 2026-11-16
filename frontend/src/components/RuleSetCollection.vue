<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Table from './Table.vue'
import IconButton from './IconButton.vue'
import PopConfirm from './PopConfirm.vue'
import RuleSetRefreshAction from './RuleSetRefreshAction.vue'
import type { RuleSet } from '../types/api'

const props = defineProps<{
  ruleSets: RuleSet[]
  loading: boolean
  isFocused: (tag: string) => boolean
  references: Record<string, { loading: boolean; cascade: boolean; details: string }>
}>()
defineEmits<{ edit: [ruleSet: RuleSet]; delete: [tag: string]; references: [tag: string] }>()
const { t } = useI18n()
const selectedKeys = ref<string[]>([])
const busy = ref(false)
const remote = computed(() => new Set(props.ruleSets.filter(set => set.type === 'remote').map(set => set.tag)))
const isSelectable = (tag: string) => remote.value.has(tag)
const columns = computed(() => [
  { key: 'tag', label: t('route.ruleSets.fields.tag'), class: 'w-56' },
  { key: 'type', label: t('route.ruleSets.fields.type'), class: 'w-24' },
  { key: 'format', label: t('route.ruleSets.fields.format'), class: 'w-24' },
  { key: 'url', label: t('route.ruleSets.fields.url'), class: 'w-64' },
  { key: 'detour', label: t('route.ruleSets.fields.downloadDetour'), class: 'w-40' },
  { key: 'interval', label: t('route.ruleSets.fields.updateInterval'), class: 'w-28' },
  { key: 'actions', label: t('common.actions'), class: 'col-actions w-36', minWidth: 144 },
])
</script>

<template>
  <Table v-model:selected-keys="selectedKeys" :columns="columns" :loading="loading" :empty="!ruleSets.length"
    :is-row-selectable="isSelectable" :selection-disabled="busy" column-storage-key="route-rule-sets" class="table-surface">
    <template #empty>{{ t('route.ruleSets.empty') }}</template>
    <template #selectionActions="{ selectedKeys: selected }">
      <RuleSetRefreshAction :tags="selected" @busy="busy = $event" />
    </template>
    <tr v-for="ruleSet in ruleSets" :key="ruleSet.tag" :class="{ 'focus-target': isFocused(ruleSet.tag) }">
      <td>{{ ruleSet.tag }}</td>
      <td>{{ ruleSet.type }}</td>
      <td>{{ ruleSet.format || '—' }}</td>
      <td>{{ ruleSet.url || '—' }}</td>
      <td>{{ ruleSet.download_detour || '—' }}</td>
      <td>{{ ruleSet.update_interval || '—' }}</td>
      <td class="col-actions"><div class="flex items-center justify-end gap-1" :inert="busy">
        <RuleSetRefreshAction v-if="ruleSet.type === 'remote'" :tag="ruleSet.tag" @busy="busy = $event" />
        <IconButton action="edit" @click="$emit('edit', ruleSet)" />
        <PopConfirm :message="t('route.ruleSets.confirm.delete')" :target="ruleSet.tag"
          :details="references[ruleSet.tag]?.details" :loading="references[ruleSet.tag]?.loading ?? false"
          :loading-label="t('route.ruleSets.cascade.checking')"
          :confirm-label="references[ruleSet.tag]?.cascade ? t('route.ruleSets.cascade.confirm') : t('common.delete')"
          tone="danger" align="right" @open="$emit('references', ruleSet.tag)" @confirm="$emit('delete', ruleSet.tag)" />
      </div></td>
    </tr>
  </Table>
</template>
