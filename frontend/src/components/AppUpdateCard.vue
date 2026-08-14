<script setup lang="ts">
/**
 * Application self-update panel.
 *
 * Renders "<current> -> <latest> [Update]" and drives the whole update flow:
 * pick a version, confirm, watch progress, then reload once the restarted
 * server answers again. All shared state lives in `useAppUpdate`, so the
 * sidebar badge stays in sync with whatever happens here.
 *
 * `embedded` drops the standalone card chrome so the panel can live inside the
 * Settings "About" card, which already states the running version — the
 * redundant "current version" chip is hidden in that mode.
 */
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowPathIcon, ArrowUpCircleIcon, ChevronDownIcon } from '@heroicons/vue/24/outline'
import { useAppUpdate } from '../composables/useAppUpdate'
import { useConfirm } from '../composables/useConfirm'
import { useNotify } from '../composables/useNotify'

withDefaults(defineProps<{ embedded?: boolean }>(), { embedded: false })

const { t } = useI18n()
const notify = useNotify()
const { confirm } = useConfirm()

const {
  status,
  releases,
  task,
  phase,
  errorMessage,
  checking,
  loadingReleases,
  busy,
  progress,
  hasUpdate,
  currentVersion,
  latestVersion,
  refreshStatus,
  loadReleases,
  startUpdate,
  reset,
} = useAppUpdate()

/** Empty string means "latest release". */
const selectedTag = ref('')
const showNotes = ref(false)

onMounted(async () => {
  const result = await refreshStatus(false)
  if (!result) {
    notify.error(t('settings.update.toast.loadFailed'))
    return
  }
  await fetchReleases(false)
})

const fetchReleases = async (force: boolean) => {
  try {
    await loadReleases(force)
  } catch (err) {
    // Non-fatal: the "update to latest" path still works without the picker.
    notify.apiError(err, t('settings.update.toast.releasesFailed'))
  }
}

const checkAgain = async () => {
  const result = await refreshStatus(true)
  if (!result) {
    notify.error(t('settings.update.toast.loadFailed'))
    return
  }
  await fetchReleases(true)
}

/** The tag that will actually be installed — the explicit pick, or latest. */
const targetVersion = computed(() => selectedTag.value || latestVersion.value)

/** Notes for whatever version is currently selected. */
const selectedNotes = computed(() => {
  if (!selectedTag.value) return status.value?.latest_notes ?? ''
  return releases.value.find((r) => r.tag === selectedTag.value)?.notes ?? ''
})

/** Enabled whenever there is something concrete to install. */
const canUpdate = computed(() => !busy.value && Boolean(targetVersion.value))

const progressLabel = computed(() => {
  switch (phase.value) {
    case 'updating':
      return task.value?.message || t('settings.update.progress.updating')
    case 'restarting':
      return t('settings.update.progress.restarting')
    case 'waiting':
      return t('settings.update.progress.waiting')
    case 'done':
      return t('settings.update.progress.done')
    default:
      return ''
  }
})

const formatDate = (iso: string) => {
  if (!iso) return ''
  const parsed = new Date(iso)
  return Number.isNaN(parsed.getTime()) ? '' : parsed.toLocaleDateString()
}

const runUpdate = async () => {
  const version = targetVersion.value
  if (!version) return

  const ok = await confirm({
    title: t('settings.update.confirmTitle'),
    message: t('settings.update.confirmMessage', { version }),
    confirmLabel: t('settings.update.update'),
    cancelLabel: t('common.cancel'),
    tone: 'danger',
  })
  if (!ok) return

  try {
    await startUpdate(selectedTag.value || undefined)
    notify.success(t('settings.update.toast.started'))
  } catch (err) {
    notify.apiError(err, t('settings.update.toast.startFailed'))
  }
}
</script>

<template>
  <div :class="embedded ? '' : 'bg-white dark:bg-gray-800 rounded-surface shadow p-5'">
    <div class="flex items-start justify-between gap-3 mb-1">
      <h3
        :class="
          embedded
            ? 'text-sm font-semibold text-gray-900 dark:text-gray-100'
            : 'text-lg font-semibold text-gray-900 dark:text-gray-100'
        "
      >
        {{ $t('settings.update.title') }}
      </h3>
      <button
        @click="checkAgain"
        :disabled="checking || busy"
        class="flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium rounded-control text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors disabled:opacity-50 cursor-pointer"
      >
        <ArrowPathIcon class="h-3.5 w-3.5" :class="{ 'animate-spin': checking }" />
        <span>{{ checking ? $t('settings.update.checking') : $t('settings.update.checkAgain') }}</span>
      </button>
    </div>
    <p class="text-sm text-gray-500 dark:text-gray-400 mb-4">{{ $t('settings.update.desc') }}</p>

    <!-- Version summary: current -> latest [Update] -->
    <div class="flex flex-wrap items-center gap-3 mb-3">
      <!-- The About card states the running version right above this panel,
           so repeating it there would just be noise. -->
      <div v-if="!embedded" class="flex items-center gap-2">
        <span class="text-xs text-gray-500 dark:text-gray-400">
          {{ $t('settings.update.currentVersion') }}
        </span>
        <span class="font-mono text-sm font-semibold text-gray-900 dark:text-gray-100">
          {{ currentVersion || '—' }}
        </span>
      </div>

      <template v-if="latestVersion && latestVersion !== currentVersion">
        <span v-if="!embedded" class="text-gray-400 dark:text-gray-500">&rarr;</span>
        <div class="flex items-center gap-2">
          <span class="text-xs text-gray-500 dark:text-gray-400">
            {{ $t('settings.update.latestVersion') }}
          </span>
          <span
            class="font-mono text-sm font-semibold"
            :class="hasUpdate ? 'text-emerald-600 dark:text-emerald-400' : 'text-gray-900 dark:text-gray-100'"
          >
            {{ latestVersion }}
          </span>
        </div>
      </template>

      <button
        v-if="!busy"
        @click="runUpdate"
        :disabled="!canUpdate"
        class="ml-auto flex items-center gap-1.5 px-4 py-2 text-sm font-medium text-white bg-primary-600 rounded-control hover:bg-primary-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
      >
        <ArrowUpCircleIcon class="h-4 w-4" />
        <span>{{ targetVersion ? $t('settings.update.updateTo', { version: targetVersion }) : $t('settings.update.update') }}</span>
      </button>
    </div>

    <!-- Status line -->
    <p v-if="status?.check_error" class="text-sm text-amber-600 dark:text-amber-400 mb-3">
      {{ $t('settings.update.checkFailed', { error: status.check_error }) }}
    </p>
    <p v-else-if="!status?.current_known" class="text-sm text-gray-500 dark:text-gray-400 mb-3">
      {{ $t('settings.update.devBuild') }}
    </p>
    <p v-else-if="hasUpdate" class="text-sm text-emerald-600 dark:text-emerald-400 mb-3">
      {{ $t('settings.update.updateAvailable') }}
    </p>
    <p v-else class="text-sm text-gray-500 dark:text-gray-400 mb-3">
      {{ $t('settings.update.upToDate') }}
    </p>

    <!-- Version picker -->
    <div v-if="!busy" class="flex flex-wrap items-end gap-3">
      <div>
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
          {{ $t('settings.update.chooseVersion') }}
        </label>
        <select
          v-model="selectedTag"
          :disabled="loadingReleases"
          class="min-w-56 rounded-control border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-900 px-3 py-2 text-sm text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-primary-500 disabled:opacity-50"
        >
          <option value="">
            {{ $t('settings.update.latestOption') }}{{ latestVersion ? ` (${latestVersion})` : '' }}
          </option>
          <option v-for="release in releases" :key="release.tag" :value="release.tag">
            {{ release.tag }}
            <template v-if="release.is_current"> — {{ $t('settings.update.current') }}</template>
            <template v-else-if="release.prerelease"> — {{ $t('settings.update.prerelease') }}</template>
            <template v-if="release.published_at"> ({{ formatDate(release.published_at) }})</template>
          </option>
        </select>
      </div>

      <button
        v-if="selectedNotes"
        @click="showNotes = !showNotes"
        class="flex items-center gap-1 px-2 py-2 text-sm text-primary-600 dark:text-primary-400 hover:underline cursor-pointer"
      >
        <span>{{ $t('settings.update.releaseNotes') }}</span>
        <ChevronDownIcon class="h-4 w-4 transition-transform" :class="{ 'rotate-180': showNotes }" />
      </button>
    </div>

    <!-- Release notes -->
    <pre
      v-if="showNotes && selectedNotes && !busy"
      class="mt-3 max-h-64 overflow-auto rounded-control bg-gray-50 dark:bg-gray-900 p-3 text-xs whitespace-pre-wrap text-gray-700 dark:text-gray-300"
    >{{ selectedNotes }}</pre>

    <!-- Progress -->
    <div v-if="busy" class="mt-2">
      <div class="flex items-center justify-between mb-1.5">
        <span class="text-sm text-gray-700 dark:text-gray-300">{{ progressLabel }}</span>
        <span class="text-xs font-mono text-gray-500 dark:text-gray-400">{{ progress }}%</span>
      </div>
      <div class="h-2 w-full rounded-pill bg-gray-200 dark:bg-gray-700 overflow-hidden">
        <div
          class="h-full rounded-pill bg-primary-600 transition-all duration-300"
          :class="{ 'animate-pulse': phase === 'restarting' || phase === 'waiting' }"
          :style="{ width: `${progress}%` }"
        ></div>
      </div>
      <p v-if="task?.to_version" class="mt-1.5 text-xs text-gray-500 dark:text-gray-400 font-mono">
        {{ task.from_version }} &rarr; {{ task.to_version }}
      </p>
    </div>

    <!-- Failure -->
    <div
      v-if="phase === 'failed'"
      class="mt-3 rounded-control border border-red-200 dark:border-red-900 bg-red-50 dark:bg-red-950/40 p-3"
    >
      <p class="text-sm font-medium text-red-700 dark:text-red-300">
        {{ $t('settings.update.progress.failed') }}
      </p>
      <p class="mt-1 text-xs text-red-600 dark:text-red-400 break-words">{{ errorMessage }}</p>
      <button
        @click="reset"
        class="mt-2 px-3 py-1.5 text-xs font-medium text-red-700 dark:text-red-300 border border-red-300 dark:border-red-800 rounded-control hover:bg-red-100 dark:hover:bg-red-900/40 transition-colors cursor-pointer"
      >
        {{ $t('settings.update.retry') }}
      </button>
    </div>
  </div>
</template>
