<script setup lang="ts">
/**
 * The Connections table's settings dialog: which columns, in what order, how
 * the exit chain is drawn, and names for devices.
 *
 * Every change applies at once — the table is visible behind the dialog, so it
 * IS the preview, and a Save button would only add a way to lose the edit.
 *
 * Columns reorder two ways on purpose: dragging is the obvious gesture with a
 * mouse and is unavailable on touch and to a keyboard, so each chip also
 * carries ‹ › buttons.
 */
import { computed, ref } from 'vue'
import { Dialog } from '../volt'
import Button from './Button.vue'
import { ALL_COLUMNS, moveColumn, placeColumn, type ColumnKey, type TableSettings } from '../utils/connectionsTable'

const props = defineProps<{
  visible: boolean
  settings: TableSettings
  /** Source IPs seen this session, for the label editor. */
  sources: string[]
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'update', patch: Partial<TableSettings>): void
  (e: 'reset'): void
}>()

const shown = computed({
  get: () => props.visible,
  set: (value: boolean) => emit('update:visible', value),
})

const available = computed(() => ALL_COLUMNS.filter((key) => !props.settings.columns.includes(key)))

const add = (key: ColumnKey) => emit('update', { columns: [...props.settings.columns, key] })
// The last column cannot be removed: a table with no columns is a blank card.
const remove = (key: ColumnKey) => {
  if (props.settings.columns.length > 1) emit('update', { columns: props.settings.columns.filter((column) => column !== key) })
}
const shift = (key: ColumnKey, step: -1 | 1) => emit('update', { columns: moveColumn(props.settings.columns, key, step) })

const dragging = ref<ColumnKey | null>(null)
const over = ref<ColumnKey | null>(null)
const drop = (target: ColumnKey) => {
  if (dragging.value) emit('update', { columns: placeColumn(props.settings.columns, dragging.value, target) })
  dragging.value = null
  over.value = null
}

const setLabel = (ip: string, value: string) => {
  const labels = { ...props.settings.labels }
  const trimmed = value.trim()
  if (trimmed) labels[ip] = trimmed
  else delete labels[ip]
  emit('update', { labels })
}
</script>

<template>
  <Dialog v-model:visible="shown" :header="$t('connections.settings.title')" modal class="w-full max-w-xl">
    <div class="space-y-4 text-sm">
      <section>
        <h4 class="mb-1.5 text-xs font-semibold text-gray-500 dark:text-gray-400">{{ $t('connections.settings.shown') }}</h4>
        <div class="flex min-h-[42px] flex-wrap gap-1.5 rounded-control border border-dashed border-gray-300 p-2 dark:border-gray-600">
          <span
            v-for="(key, index) in settings.columns"
            :key="key"
            draggable="true"
            class="inline-flex cursor-grab items-center gap-0.5 rounded-control border bg-gray-50 py-0.5 pl-2 pr-1 dark:bg-gray-900/40"
            :class="over === key ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/30' : 'border-gray-300 dark:border-gray-600'"
            @dragstart="dragging = key"
            @dragover.prevent="over = key"
            @dragleave="over === key && (over = null)"
            @drop.prevent="drop(key)"
            @dragend="dragging = null; over = null"
          >
            {{ $t(`connections.columns.${key}`) }}
            <button type="button" class="chip-btn" :disabled="index === 0" :aria-label="$t('connections.settings.moveLeft', { name: $t(`connections.columns.${key}`) })" @click="shift(key, -1)">‹</button>
            <button type="button" class="chip-btn" :disabled="index === settings.columns.length - 1" :aria-label="$t('connections.settings.moveRight', { name: $t(`connections.columns.${key}`) })" @click="shift(key, 1)">›</button>
            <button type="button" class="chip-btn" :disabled="settings.columns.length === 1" :aria-label="$t('connections.settings.hide', { name: $t(`connections.columns.${key}`) })" @click="remove(key)">×</button>
          </span>
        </div>
      </section>

      <section>
        <h4 class="mb-1.5 text-xs font-semibold text-gray-500 dark:text-gray-400">{{ $t('connections.settings.available') }}</h4>
        <div class="flex min-h-[42px] flex-wrap gap-1.5 rounded-control border border-dashed border-gray-300 p-2 dark:border-gray-600">
          <button
            v-for="key in available"
            :key="key"
            type="button"
            class="rounded-control border border-gray-300 bg-gray-50 px-2.5 py-1 hover:border-primary-500 hover:text-primary-700 dark:border-gray-600 dark:bg-gray-900/40 dark:hover:text-primary-300"
            @click="add(key)"
          >
            + {{ $t(`connections.columns.${key}`) }}
          </button>
          <span v-if="available.length === 0" class="self-center text-xs text-gray-500 dark:text-gray-400">{{ $t('connections.settings.allShown') }}</span>
        </div>
      </section>

      <section class="flex flex-wrap gap-x-6 gap-y-2">
        <label class="flex items-center gap-2">
          <input type="checkbox" :checked="settings.fullChain" @change="emit('update', { fullChain: ($event.target as HTMLInputElement).checked })" />
          {{ $t('connections.settings.fullChain') }}
        </label>
        <label class="flex items-center gap-2">
          <input type="checkbox" :checked="settings.dense" @change="emit('update', { dense: ($event.target as HTMLInputElement).checked })" />
          {{ $t('connections.settings.dense') }}
        </label>
      </section>

      <section>
        <h4 class="mb-1.5 text-xs font-semibold text-gray-500 dark:text-gray-400">{{ $t('connections.settings.labels') }}</h4>
        <div v-if="sources.length" class="grid grid-cols-[minmax(0,auto)_minmax(0,1fr)] items-center gap-x-3 gap-y-1.5">
          <template v-for="(ip, index) in sources" :key="ip">
            <label :for="`conn-label-${index}`" class="truncate tabular-nums text-gray-700 dark:text-gray-300">{{ ip }}</label>
            <input
              :id="`conn-label-${index}`"
              type="text"
              :value="settings.labels[ip] ?? ''"
              :placeholder="$t('connections.settings.labelPlaceholder')"
              class="h-8 w-full min-w-0 rounded-control border border-gray-300 bg-white px-2.5 text-sm text-gray-900 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100"
              @input="setLabel(ip, ($event.target as HTMLInputElement).value)"
            />
          </template>
        </div>
        <p v-else class="text-xs text-gray-500 dark:text-gray-400">{{ $t('connections.settings.noSources') }}</p>
      </section>
    </div>

    <template #footer>
      <Button variant="secondary" @click="emit('reset')">{{ $t('connections.settings.reset') }}</Button>
      <Button @click="shown = false">{{ $t('connections.settings.done') }}</Button>
    </template>
  </Dialog>
</template>

<style scoped>
.chip-btn {
  border-radius: 4px;
  padding: 2px 6px;
  color: var(--color-text-secondary);
}
.chip-btn:hover:not(:disabled) {
  color: var(--color-primary);
}
.chip-btn:disabled {
  cursor: default;
  opacity: 0.3;
}
</style>
