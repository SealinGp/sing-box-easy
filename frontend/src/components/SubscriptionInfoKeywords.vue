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
import { ArrowUturnLeftIcon } from '@heroicons/vue/24/outline'
import Modal from './Modal.vue'
import Button from './Button.vue'
import ChipsField from './ChipsField.vue'
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
const isLoading = ref(false)
const isSaving = ref(false)

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
    if (open) load()
  },
  { immediate: true },
)

// The chips control owns entry, removal and pasted batches; this model adds the
// two rules specific to keywords — matching is case-insensitive (so the list
// shows what will actually be stored) and the server's bounds are enforced
// before a save can fail on them. Always assigns a NEW array.
const keywordsModel = computed({
  get: () => keywords.value,
  set: (next: string[]) => {
    const seen = new Set<string>()
    const accepted: string[] = []
    let sawTooLong = false
    let sawTooMany = false

    for (const raw of next) {
      const keyword = raw.trim().toLowerCase()
      if (keyword === '' || seen.has(keyword)) continue
      if ([...keyword].length > maxLength.value) {
        sawTooLong = true
        continue
      }
      if (accepted.length >= maxKeywords.value) {
        sawTooMany = true
        break
      }
      seen.add(keyword)
      accepted.push(keyword)
    }

    if (sawTooLong) notify.error(t('subscriptions.keywords.tooLong', { max: maxLength.value }))
    if (sawTooMany) notify.error(t('subscriptions.keywords.tooMany', { max: maxKeywords.value }))

    keywords.value = accepted
  },
})

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

      <!-- One control for the whole list: the chips ARE the editor, so the
           separate "text field + Add button" row is gone. -->
      <ChipsField
        v-model="keywordsModel"
        :placeholder="$t('subscriptions.keywords.addPlaceholder')"
        :hint="$t('subscriptions.keywords.addHint')"
        :disabled="isLoading"
      />
      <p
        v-if="!isLoading && keywords.length === 0"
        class="text-xs text-gray-400 dark:text-gray-500"
      >
        {{ $t('subscriptions.keywords.emptyHint') }}
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
