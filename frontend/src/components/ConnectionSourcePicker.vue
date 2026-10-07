<script setup lang="ts">
/**
 * The device filter: a dropdown whose options can be searched.
 *
 * Not `SearchPicker` — that is an action input that clears itself after each
 * pick, and this holds a selection. Not volt/Select either: its filter matches
 * one label field, and a device has to be findable by its address AND by the
 * name the operator gave it.
 */
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { SourceOption } from '../utils/connectionsTable'

const props = defineProps<{
  /** Selected source IP, '' for every device. */
  modelValue: string
  options: SourceOption[]
  /** Row count behind "All devices". */
  total: number
}>()

const emit = defineEmits<{ (e: 'update:modelValue', value: string): void }>()

const open = ref(false)
const query = ref('')
const activeIndex = ref(0)
const root = ref<HTMLElement | null>(null)
const input = ref<HTMLInputElement | null>(null)
const trigger = ref<HTMLButtonElement | null>(null)
const list = ref<HTMLElement | null>(null)

interface Entry { ip: string; label: string; count: number }

const entries = computed<Entry[]>(() => {
  const needle = query.value.trim().toLowerCase()
  const all: Entry[] = [{ ip: '', label: '', count: props.total }, ...props.options]
  if (needle === '') return all
  return all.filter((entry) => entry.ip !== '' && (entry.ip.toLowerCase().includes(needle) || entry.label.toLowerCase().includes(needle)))
})

const selected = computed(() => props.options.find((option) => option.ip === props.modelValue))

watch(entries, (value) => {
  activeIndex.value = Math.min(activeIndex.value, Math.max(0, value.length - 1))
})

const show = async () => {
  query.value = ''
  open.value = true
  activeIndex.value = Math.max(0, entries.value.findIndex((entry) => entry.ip === props.modelValue))
  await nextTick()
  input.value?.focus()
}

const hide = (refocus = false) => {
  open.value = false
  if (refocus) trigger.value?.focus()
}

const pick = (entry: Entry | undefined) => {
  if (!entry) return
  emit('update:modelValue', entry.ip)
  hide(true)
}

const move = async (step: 1 | -1) => {
  const count = entries.value.length
  if (count === 0) return
  activeIndex.value = (activeIndex.value + step + count) % count
  await nextTick()
  list.value?.querySelector('[data-active="true"]')?.scrollIntoView({ block: 'nearest' })
}

const onOutside = (event: PointerEvent) => {
  if (open.value && root.value && !root.value.contains(event.target as Node)) hide()
}

onMounted(() => document.addEventListener('pointerdown', onOutside))
onBeforeUnmount(() => document.removeEventListener('pointerdown', onOutside))
</script>

<template>
  <div ref="root" class="relative">
    <button
      ref="trigger"
      type="button"
      class="flex h-8 max-w-[240px] items-center gap-2 rounded-control border border-gray-300 bg-white px-2.5 text-left text-sm text-gray-900 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100"
      aria-haspopup="listbox"
      :aria-expanded="open"
      :aria-label="$t('connections.device.label')"
      @click="open ? hide() : show()"
    >
      <span class="truncate">{{ selected ? selected.label : $t('connections.device.all') }}</span>
      <svg class="h-3 w-3 shrink-0 text-gray-500" viewBox="0 0 12 12" fill="none" stroke="currentColor" stroke-width="1.5" aria-hidden="true">
        <path d="M2.5 4.5 6 8l3.5-3.5" stroke-linecap="round" stroke-linejoin="round" />
      </svg>
    </button>

    <div
      v-if="open"
      class="absolute left-0 top-[calc(100%+4px)] z-20 flex w-[min(280px,calc(100vw-32px))] flex-col gap-1.5 solid-surface rounded-control border border-gray-300 p-1.5 shadow-float dark:border-gray-600"
    >
      <input
        ref="input"
        v-model="query"
        type="search"
        autocomplete="off"
        :placeholder="$t('connections.device.search')"
        :aria-label="$t('connections.device.search')"
        class="h-8 w-full rounded-control border border-gray-300 bg-white px-2.5 text-sm text-gray-900 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100"
        @input="activeIndex = 0"
        @keydown.down.prevent="move(1)"
        @keydown.up.prevent="move(-1)"
        @keydown.enter.prevent="pick(entries[activeIndex])"
        @keydown.esc.stop.prevent="hide(true)"
      />
      <div ref="list" class="flex max-h-60 flex-col gap-px overflow-auto" role="listbox" :aria-label="$t('connections.device.label')">
        <button
          v-for="(entry, index) in entries"
          :key="entry.ip"
          type="button"
          role="option"
          tabindex="-1"
          :aria-selected="entry.ip === modelValue"
          :data-active="index === activeIndex"
          class="flex w-full items-baseline gap-2 rounded px-2 py-1.5 text-left text-sm hover:bg-primary-50 dark:hover:bg-primary-900/30"
          :class="[
            index === activeIndex ? 'bg-gray-100 dark:bg-gray-700' : '',
            entry.ip === modelValue ? 'font-semibold text-primary-700 dark:text-primary-300' : 'text-gray-900 dark:text-gray-100',
          ]"
          @click="pick(entry)"
        >
          <span class="min-w-0 flex-1 truncate">{{ entry.ip === '' ? $t('connections.device.all') : entry.label }}</span>
          <small v-if="entry.ip !== '' && entry.label !== entry.ip" class="tabular-nums text-gray-500 dark:text-gray-400">{{ entry.ip }}</small>
          <small class="tabular-nums text-gray-500 dark:text-gray-400">{{ entry.count }}</small>
        </button>
        <p v-if="entries.length === 0" class="px-2 py-2 text-xs text-gray-500 dark:text-gray-400">{{ $t('connections.device.none') }}</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
/*
 * Opaque on purpose. The app's surfaces are translucent glass, which is right
 * for a card resting on the page and wrong for something floating OVER a
 * table: the rows underneath show through and both become unreadable.
 */
.solid-surface {
  background: #ffffff;
}
@media (prefers-color-scheme: dark) {
  .solid-surface {
    background: #131a23;
  }
}
</style>
