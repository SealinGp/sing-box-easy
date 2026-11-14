<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { PlusIcon } from '@heroicons/vue/24/outline'
import ContextPanel from '../volt/ContextPanel.vue'
import type { SortCard } from '../utils/overviewSort'

const props = defineProps<{ cards: SortCard[] }>()
const emit = defineEmits<{ select: [id: string] }>()
const { t } = useI18n()
const visible = ref(false)
const search = ref('')
const matches = computed(() => props.cards.filter(card => t(card.title).toLocaleLowerCase().includes(search.value.trim().toLocaleLowerCase())))
function open() { search.value = ''; visible.value = true }
function select(id: string) { visible.value = false; emit('select', id) }
</script>

<template>
  <button type="button" data-add-card class="flex h-full w-full flex-col items-center justify-center gap-2 rounded-surface text-sm text-text-secondary hover:bg-bg-tertiary focus-visible:outline-2 focus-visible:outline-primary" @click="open">
    <PlusIcon class="h-5 w-5" aria-hidden="true" />
    {{ t('overview.layout.addCard') }}
  </button>
  <ContextPanel v-model:visible="visible" :header="t('overview.layout.addCard')" class="max-w-sm">
    <input v-model="search" type="search" autofocus :aria-label="t('overview.layout.searchCards')" :placeholder="t('overview.layout.searchCards')" class="mb-3 w-full rounded-control border border-border bg-bg-secondary px-3 py-2 text-sm" />
    <div class="space-y-1">
      <button v-for="card in matches" :key="card.id" type="button" class="flex w-full items-center gap-2 rounded-control px-3 py-2 text-left text-sm hover:bg-bg-tertiary focus-visible:outline-2 focus-visible:outline-primary" @click="select(card.id)">
        <PlusIcon class="h-4 w-4" aria-hidden="true" />{{ t(card.title) }}
      </button>
      <p v-if="!matches.length" role="status" class="py-3 text-sm text-text-secondary">{{ t('overview.layout.noCards') }}</p>
    </div>
  </ContextPanel>
</template>
