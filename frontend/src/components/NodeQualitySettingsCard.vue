<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useNodeQualityStore } from '../stores/nodeQuality'
import { DEFAULT_NODE_QUALITY, NODE_QUALITY_WATERMARKS, nodeQualityErrors } from '../utils/nodeQuality'
import { apiErrorMessage } from '../utils/apiErrorMessage'
import Input from './Input.vue'
import Button from './Button.vue'
import FieldHelp from './FieldHelp.vue'

const { t } = useI18n()
const store = useNodeQualityStore()
const draft = reactive({ ...DEFAULT_NODE_QUALITY })
const loading = ref(true)
const saving = ref(false)
const error = ref('')
const saved = ref(false)
const errors = computed(() => nodeQualityErrors(draft))
const invalid = computed(() => Object.values(errors.value).some(Boolean))
const watermarks = computed(() => [...new Set<number>([...NODE_QUALITY_WATERMARKS, draft.good_percent])].sort((a, b) => a - b))
const requiredPasses = computed(() => Math.ceil(draft.samples * draft.good_percent / 100))
async function load() {
  loading.value = true
  await store.load()
  Object.assign(draft, store.rules)
  loading.value = false
}
onMounted(load)
async function save() {
  if (invalid.value) return
  error.value = ''
  saved.value = false
  saving.value = true
  try {
    await store.save({ ...draft })
    Object.assign(draft, store.rules)
    saved.value = true
  } catch (err) {
    error.value = apiErrorMessage(err, t('settings.toast.saveFailed'))
  } finally { saving.value = false }
}
</script>

<template>
  <section id="node-quality" class="rounded-surface bg-white p-4 shadow dark:bg-gray-800">
    <div class="mb-4 flex items-center gap-1">
      <h3 class="text-lg font-semibold text-text-primary">{{ t('settings.nodeQuality.title') }}</h3>
      <FieldHelp :label="t('settings.nodeQuality.title')" :text="t('settings.nodeQuality.help')" />
    </div>
    <p v-if="loading" role="status" class="text-sm text-text-secondary">{{ t('common.loading') }}</p>
    <div v-else-if="store.failed && !store.ready" class="space-y-2">
      <p role="alert" class="text-sm text-red-600">{{ t('settings.toast.loadFailed') }}</p>
      <Button size="sm" @click="load">{{ t('proxies.retry') }}</Button>
    </div>
    <form v-else class="space-y-3" :aria-busy="saving" @submit.prevent="save" @input="saved = false">
      <div class="grid grid-cols-2 gap-3">
        <Input v-model="draft.samples" type="number" :min="1" :max="100" required :disabled="saving"
          :label="t('settings.nodeQuality.samples')" :error="errors.samples ? t('settings.nodeQuality.samplesError') : ''" />
        <Input v-model="draft.max_delay_ms" type="number" :min="1" :max="60000" required :disabled="saving"
          :label="t('settings.nodeQuality.delay')" :error="errors.max_delay_ms ? t('settings.nodeQuality.delayError') : ''" />
      </div>
      <fieldset :disabled="saving" class="space-y-2">
        <legend class="mb-2 text-sm font-medium text-text-primary">{{ t('settings.nodeQuality.watermark') }}</legend>
        <div class="grid grid-cols-2 gap-2 sm:grid-cols-4">
          <label v-for="watermark in watermarks" :key="watermark" class="flex min-h-10 cursor-pointer items-center justify-center gap-2 rounded-control border px-3 py-2 text-sm"
            :class="draft.good_percent === watermark ? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-950 dark:text-primary-300' : 'border-border text-text-secondary'">
            <input v-model="draft.good_percent" type="radio" name="node-quality-watermark" :value="watermark" class="accent-primary-600" @change="saved = false" />
            {{ watermark }}%
          </label>
        </div>
        <p v-if="errors.good_percent" role="alert" class="text-xs text-red-600">{{ t('settings.nodeQuality.percentError') }}</p>
      </fieldset>
      <p v-if="!invalid" class="text-xs text-text-secondary">{{ t('settings.nodeQuality.requiredPasses', { count: draft.samples, passed: requiredPasses, delay: draft.max_delay_ms }) }}</p>
      <p v-if="error" role="alert" class="text-sm text-red-600">{{ error }}</p>
      <p v-if="saved" role="status" class="text-sm text-green-600">{{ t('settings.toast.savedTitle') }}</p>
      <div class="flex flex-wrap gap-2">
        <Button type="submit" size="sm" :loading="saving" :disabled="saving || invalid">{{ t('common.save') }}</Button>
        <Button type="button" variant="ghost" size="sm" :disabled="saving" @click="Object.assign(draft, DEFAULT_NODE_QUALITY); saved = false">{{ t('settings.nodeQuality.defaults') }}</Button>
      </div>
    </form>
  </section>
</template>
