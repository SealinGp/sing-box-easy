<script setup lang="ts">
/**
 * How often subscription nodes are URL-tested, and how much history is kept.
 *
 * The two retention knobs are really one question — how much disk is this
 * allowed to use — so the card answers it with a MEASURED figure (rows actually
 * stored, and their approximate size) rather than leaving the operator to
 * multiply an interval by a window. That matters most on the hardware this
 * panel most often runs on, where the root overlay can be a few tens of MB.
 */
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { subProbeService } from '../services'
import { useNotify } from '../composables/useNotify'
import { formatBytes } from '../utils/formatBytes'

const { t } = useI18n()
const notify = useNotify()

const loading = ref(true)
const saving = ref(false)

const intervalMinutes = ref(10)
const timeoutMs = ref(5000)
const retentionDays = ref(7)
const maxPoints = ref(2016)
const sampleCount = ref(0)

/** Mirrors app/pkg/settings' clamps, which reject anything outside them. */
const LIMITS = {
  intervalMinutes: { min: 1, max: 1440 },
  timeoutMs: { min: 1000, max: 8000 },
  retentionDays: { min: 1, max: 90 },
  maxPoints: { min: 60, max: 20000 },
}

/**
 * Bytes per stored row, measured against the SQLite schema: six integers, a
 * subscription id, and the (sub_id, at) index entry. Approximate on purpose —
 * it is a budgeting figure, and quoting it to the byte would imply a precision
 * the page does not have.
 */
const BYTES_PER_SAMPLE = 80

const storedSize = computed(() => formatBytes(sampleCount.value * BYTES_PER_SAMPLE))

async function load() {
  loading.value = true
  try {
    const { data } = await subProbeService.getStatus()
    intervalMinutes.value = Math.max(1, Math.round(data.interval_secs / 60))
    timeoutMs.value = data.timeout_ms
    retentionDays.value = data.retention
    maxPoints.value = data.max_points
    sampleCount.value = data.sample_count
  } catch (err) {
    // The prober's endpoints answer even when sing-box is down, so a failure
    // here is a real panel problem worth reporting.
    notify.apiError(err, t('subProbe.notify.settingsFailed'))
  } finally {
    loading.value = false
  }
}

onMounted(load)

async function save() {
  saving.value = true
  try {
    const { data } = await subProbeService.updateSettings({
      interval_seconds: intervalMinutes.value * 60,
      timeout_ms: timeoutMs.value,
      retention_days: retentionDays.value,
      max_points: maxPoints.value,
    })
    // Echo the server's clamped values back rather than keeping the typed
    // ones: a value it adjusted must not keep showing as accepted.
    intervalMinutes.value = Math.max(1, Math.round(data.interval_secs / 60))
    timeoutMs.value = data.timeout_ms
    retentionDays.value = data.retention
    maxPoints.value = data.max_points
    // Tightening retention trims immediately server-side, so this number moves.
    sampleCount.value = data.sample_count
    notify.success(t('subProbe.notify.settingsSaved'))
  } catch (err) {
    notify.apiError(err, t('subProbe.notify.settingsFailed'))
  } finally {
    saving.value = false
  }
}

const inputClass =
  'w-32 rounded-control border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-900 px-3 py-2 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-primary-500'
</script>

<template>
  <div class="bg-white dark:bg-gray-800 rounded-surface shadow p-4">
    <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-1">
      {{ $t('subProbe.settings.title') }}
    </h3>
    <p class="text-sm text-gray-500 dark:text-gray-400 mb-4">
      {{ $t('subProbe.settings.desc') }}
    </p>

    <div v-if="loading" class="h-10 flex items-center">
      <div class="animate-spin rounded-pill h-5 w-5 border-b-2 border-primary-600"></div>
    </div>

    <div v-else class="space-y-3">
      <div class="flex flex-wrap gap-3">
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            {{ $t('subProbe.settings.interval') }}
          </label>
          <div class="flex items-center gap-2">
            <input
              v-model.number="intervalMinutes"
              type="number"
              :min="LIMITS.intervalMinutes.min"
              :max="LIMITS.intervalMinutes.max"
              :class="inputClass"
            />
            <span class="text-sm text-gray-500 dark:text-gray-400">min</span>
          </div>
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            {{ $t('subProbe.settings.timeout') }}
          </label>
          <div class="flex items-center gap-2">
            <input
              v-model.number="timeoutMs"
              type="number"
              :min="LIMITS.timeoutMs.min"
              :max="LIMITS.timeoutMs.max"
              :step="500"
              :class="inputClass"
            />
            <span class="text-sm text-gray-500 dark:text-gray-400">ms</span>
          </div>
        </div>
      </div>

      <div class="flex flex-wrap gap-3">
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            {{ $t('subProbe.settings.retention') }}
          </label>
          <div class="flex items-center gap-2">
            <input
              v-model.number="retentionDays"
              type="number"
              :min="LIMITS.retentionDays.min"
              :max="LIMITS.retentionDays.max"
              :class="inputClass"
            />
            <span class="text-sm text-gray-500 dark:text-gray-400">days</span>
          </div>
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            {{ $t('subProbe.settings.maxPoints') }}
          </label>
          <input
            v-model.number="maxPoints"
            type="number"
            :min="LIMITS.maxPoints.min"
            :max="LIMITS.maxPoints.max"
            :class="inputClass"
          />
        </div>
      </div>

      <p class="text-xs text-gray-400 dark:text-gray-500">
        {{ $t('subProbe.settings.maxPointsHint') }}
      </p>

      <!-- The disk question, answered with a measured number. -->
      <p class="text-sm text-gray-500 dark:text-gray-400">
        {{ $t('subProbe.settings.stored') }}:
        <span class="font-medium text-gray-700 dark:text-gray-300">
          {{ $t('subProbe.settings.storedValue', { count: sampleCount, size: storedSize }) }}
        </span>
      </p>

      <p class="text-xs text-gray-400 dark:text-gray-500">
        {{ $t('subProbe.settings.requiresClashApi') }}
      </p>

      <button
        :disabled="saving"
        class="px-4 py-2 text-sm font-medium text-white bg-primary-600 rounded-control hover:bg-primary-700 transition-colors disabled:opacity-50"
        @click="save"
      >
        <span v-if="saving">{{ $t('common.saving') }}</span>
        <span v-else>{{ $t('common.save') }}</span>
      </button>
    </div>
  </div>
</template>
