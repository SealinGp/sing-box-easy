<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { MagnifyingGlassIcon, ArrowTurnDownLeftIcon } from '@heroicons/vue/24/outline'
import Dialog from '../volt/Dialog.vue'
import { searchMenu, type MenuGroup } from '../navigation/menu'
const props = defineProps<{ groups: MenuGroup[] }>()
const visible = defineModel<boolean>({ required: true })
const router = useRouter()
const query = ref('')
const active = ref(0)
const input = ref<HTMLInputElement>()
const results = computed(() => searchMenu(props.groups, query.value))
watch(query, () => { active.value = 0 })
watch(visible, value => { if (value) { query.value = ''; active.value = 0 } })
async function focusInput() { await nextTick(); input.value?.focus() }
function move(delta: number) {
  if (!results.value.length) return
  active.value = (active.value + delta + results.value.length) % results.value.length
  document.getElementById('nav-result-' + active.value)?.scrollIntoView({ block: 'nearest' })
}
async function navigate(index = active.value) {
  const result = results.value[index]
  if (!result) return
  await router.push(result.path)
  visible.value = false
}
</script>

<template>
  <Dialog v-model:visible="visible" modal dismissable-mask :header="$t('nav.search')" :close-button-props="{ 'aria-label': $t('nav.close') }" class="max-w-xl" @show="focusInput">
    <div class="navigation-search">
      <div class="palette-input">
        <MagnifyingGlassIcon class="h-5 w-5 shrink-0" />
        <input ref="input" v-model="query" role="combobox" :aria-label="$t('nav.search')" aria-autocomplete="list" aria-expanded="true" aria-controls="nav-search-results" :aria-activedescendant="results.length ? 'nav-result-' + active : undefined" :placeholder="$t('nav.searchPlaceholder')" autocomplete="off" @keydown.down.prevent="move(1)" @keydown.up.prevent="move(-1)" @keydown.enter.prevent="navigate()" />
      </div>
      <div id="nav-search-results" role="listbox" :aria-label="$t('nav.navigation')" class="palette-results">
        <div v-for="(item, index) in results" :id="'nav-result-' + index" :key="item.path" role="option" :aria-selected="active === index" class="palette-result" :class="{ selected: active === index }" @pointermove="active = index" @mousedown.prevent @click="navigate(index)">
          <component :is="item.icon" class="h-5 w-5 shrink-0" />
          <span class="palette-result-title">{{ item.name }}<small>{{ item.group }}</small></span>
          <ArrowTurnDownLeftIcon v-if="active === index" class="h-4 w-4 shrink-0" />
        </div>
      </div>
      <p v-if="!results.length" class="palette-empty" role="status">{{ $t('nav.noResults') }}</p>
      <div class="palette-footer"><span>↑ ↓ {{ $t('nav.browse') }}</span><span>↵ {{ $t('nav.open') }}</span><span>Esc {{ $t('nav.close') }}</span></div>
    </div>
  </Dialog>
</template>
