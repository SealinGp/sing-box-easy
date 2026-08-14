<script setup lang="ts">
/**
 * Editor for the subscription "info label" keywords.
 *
 * Some providers publish account metadata (traffic left, expiry, reset
 * countdown) as ordinary feed entries whose only distinguishing mark is the
 * display name — "剩余流量：4.7 TB" pointing at a perfectly real server. The
 * backend treats such an entry as metadata when its label (the part before the
 * colon) contains one of these keywords, which keeps it out of the outbound list
 * and surfaces it in the Plan Info column instead.
 *
 * Every provider invents its own labels, so the list is operator-editable.
 * Saving an empty list restores the built-in defaults.
 */
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { XMarkIcon, ArrowUturnLeftIcon } from '@heroicons/vue/24/outline'
import Modal from './Modal.vue'
import Button from './Button.vue'
import Input from './Input.vue'
import Badge from './Badge.vue'
import { apiService } from '../services/api'
import { SubscriptionService } from '../services/subscription'
import { useNotify } from '../composables/useNotify'

const props = defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

const { t } = useI18n()
const notify = useNotify()
const subscriptionService = new SubscriptionService(apiService)

// Fallbacks used only until the first successful load; the server is the
// authority on both bounds.
const DEFAULT_MAX_KEYWORDS = 200
const DEFAULT_MAX_LENGTH = 64

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const keywords = ref<string[]>([])
const defaults = ref<string[]>([])
const usingDefaults = ref(true)
const maxKeywords = ref(DEFAULT_MAX_KEYWORDS)
const maxLength = ref(DEFAULT_MAX_LENGTH)
const draft = ref('')
const isLoading = ref(false)
const isSaving = ref(false)

const isFull = computed(() => keywords.value.length >= maxKeywords.value)

const load = async () => {
  try {
    isLoading.value = true
    const response = await subscriptionService.getInfoKeywords()
    const data = response.data
    keywords.value = [...(data.keywords ?? [])]
    defaults.value = [...(data.defaults ?? [])]
    usingDefaults.value = data.using_defaults
    maxKeywords.value = data.limits?.max_keywords ?? DEFAULT_MAX_KEYWORDS
    maxLength.value = data.limits?.max_length ?? DEFAULT_MAX_LENGTH
  } catch (error) {
    notify.apiError(error, t('subscriptions.keywords.loadError'))
  } finally {
    isLoading.value = false
  }
}

// Reload on every open so a change made elsewhere (or a rejected save) can never
// leave a stale list on screen.
watch(
  () => props.modelValue,
  (open) => {
    if (open) {
      draft.value = ''
      load()
    }
  },
  { immediate: true },
)

// Accepts one keyword or a pasted batch ("流量, 到期" / newline-separated).
// Always assigns a NEW array so the list stays immutable.
const addDraft = () => {
  const parts = draft.value
    .split(/[,，\n]/)
    .map((part) => part.trim().toLowerCase())
    .filter(Boolean)
  if (parts.length === 0) return

  const tooLong = parts.find((part) => [...part].length > maxLength.value)
  if (tooLong) {
    notify.error(t('subscriptions.keywords.tooLong', { max: maxLength.value }))
    return
  }

  const merged = [...keywords.value]
  for (const part of parts) {
    if (merged.includes(part)) continue
    if (merged.length >= maxKeywords.value) {
      notify.error(t('subscriptions.keywords.tooMany', { max: maxKeywords.value }))
      break
    }
    merged.push(part)
  }
  keywords.value = merged
  draft.value = ''
}

const removeKeyword = (keyword: string) => {
  keywords.value = keywords.value.filter((k) => k !== keyword)
}

// Backspace on an empty input removes the last chip — the usual tag-input idiom.
const onBackspace = () => {
  if (draft.value !== '') return
  keywords.value = keywords.value.slice(0, -1)
}

const restoreDefaults = () => {
  keywords.value = [...defaults.value]
}

// An untouched list is stored as "no override" (see save), so the two states are
// deliberately indistinguishable to the user.
const sameAsDefaults = computed(
  () =>
    keywords.value.length === defaults.value.length &&
    keywords.value.every((k, i) => k === defaults.value[i]),
)

const save = async () => {
  try {
    isSaving.value = true
    // An unchanged-from-defaults list is sent as [], which clears the override
    // so future default updates keep flowing through.
    const payload = sameAsDefaults.value ? [] : keywords.value
    const response = await subscriptionService.updateInfoKeywords(payload)
    const data = response.data
    keywords.value = [...(data.keywords ?? [])]
    usingDefaults.value = data.using_defaults
    notify.success(t('subscriptions.keywords.savedOk'))
    visible.value = false
  } catch (error) {
    notify.apiError(error, t('subscriptions.keywords.saveError'))
  } finally {
    isSaving.value = false
  }
}
</script>

<template>
  <Modal v-model="visible" :title="$t('subscriptions.keywords.title')" size="lg" show-close>
    <div class="space-y-4">
      <p class="text-sm text-gray-500 dark:text-gray-400">
        {{ $t('subscriptions.keywords.desc') }}
      </p>

      <div class="flex items-center gap-2">
        <Badge :variant="usingDefaults ? 'secondary' : 'success'">
          {{
            usingDefaults
              ? $t('subscriptions.keywords.usingDefaults')
              : $t('subscriptions.keywords.usingCustom')
          }}
        </Badge>
        <span class="text-xs text-gray-400 dark:text-gray-500">
          {{ $t('subscriptions.keywords.count', { n: keywords.length, max: maxKeywords }) }}
        </span>
      </div>

      <!-- Chip list -->
      <div
        class="flex flex-wrap gap-2 min-h-[3rem] p-2 rounded-control border border-gray-200 dark:border-gray-600 bg-gray-50 dark:bg-gray-900/40"
      >
        <span v-if="isLoading" class="text-xs text-gray-400">{{ $t('common.loading') }}</span>
        <span
          v-else-if="keywords.length === 0"
          class="text-xs text-gray-400 dark:text-gray-500"
        >
          {{ $t('subscriptions.keywords.emptyHint') }}
        </span>
        <span
          v-for="keyword in keywords"
          :key="keyword"
          class="inline-flex items-center gap-1 px-2 py-0.5 rounded-pill bg-primary-100 dark:bg-primary-900 text-xs text-primary-700 dark:text-primary-200"
        >
          {{ keyword }}
          <button
            type="button"
            class="shrink-0 hover:text-red-600 dark:hover:text-red-400 transition-colors"
            @click="removeKeyword(keyword)"
            :title="$t('subscriptions.keywords.remove', { keyword })"
            :aria-label="$t('subscriptions.keywords.remove', { keyword })"
          >
            <XMarkIcon class="h-3.5 w-3.5" />
          </button>
        </span>
      </div>

      <!-- Add. Uses the shared Input so the field inherits the app's control
           styling (border, fill, focus ring, radius) from src/style/controls.css
           instead of restating a one-off set of classes. The keydown listeners
           fall through to Input's root element and catch the events as they
           bubble up from the field. -->
      <div class="flex items-center gap-2">
        <Input
          v-model="draft"
          class="flex-1"
          :placeholder="$t('subscriptions.keywords.addPlaceholder')"
          :disabled="isLoading || isFull"
          @keydown.enter.prevent="addDraft"
          @keydown.delete="onBackspace"
        />
        <Button variant="secondary" :disabled="!draft.trim() || isFull" @click="addDraft">
          {{ $t('common.add') }}
        </Button>
      </div>
      <p class="text-xs text-gray-500 dark:text-gray-400">
        {{ $t('subscriptions.keywords.addHint') }}
      </p>
    </div>

    <template #footer>
      <Button variant="ghost" :disabled="isLoading || sameAsDefaults" @click="restoreDefaults">
        <ArrowUturnLeftIcon class="h-4 w-4" />
        {{ $t('subscriptions.keywords.restoreDefaults') }}
      </Button>
      <Button variant="secondary" @click="visible = false">
        {{ $t('common.cancel') }}
      </Button>
      <Button :loading="isSaving" :disabled="isLoading" @click="save">
        {{ $t('common.save') }}
      </Button>
    </template>
  </Modal>
</template>
