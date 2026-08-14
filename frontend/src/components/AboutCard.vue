<script setup lang="ts">
/**
 * Settings "About" card.
 *
 * A single place answering "what am I running, and on what?" — the panel and
 * sing-box versions, the host's platform and architecture, the display
 * language, and the self-update controls. These used to be three separate
 * cards; grouping them means an operator reporting a problem can screenshot one
 * card instead of three.
 */
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { systemService } from '../services'
import { useAppUpdate } from '../composables/useAppUpdate'
import { useDeployment, type LayoutOverride } from '../composables/useDeployment'
import { useNotify } from '../composables/useNotify'
import LanguageSwitcher from './LanguageSwitcher.vue'
import AppUpdateCard from './AppUpdateCard.vue'
import type { SystemInfo } from '../types/api'

const { t } = useI18n()
const notify = useNotify()

const info = ref<SystemInfo | null>(null)
const loading = ref(true)

// The update panel is the authority on the running panel version — it is what
// the self-update flow rewrites. /system/info agrees, but only until an update
// finishes without a page reload.
const { currentVersion } = useAppUpdate()

// Dev builds can force the navigation layout to preview the OpenWrt top bar
// without a router. Absent from release builds entirely.
const { isDevBuild, layoutOverride, setLayoutOverride } = useDeployment()

const layoutOptions: { value: LayoutOverride; label: string }[] = [
  { value: 'auto', label: 'settings.about.layout.auto' },
  { value: 'sidebar', label: 'settings.about.layout.sidebar' },
  { value: 'topbar', label: 'settings.about.layout.topbar' },
]

const buildVersion = __APP_VERSION__

onMounted(async () => {
  try {
    info.value = await systemService.getInfo()
  } catch (err) {
    notify.apiError(err, t('settings.about.loadFailed'))
  } finally {
    loading.value = false
  }
})

const placeholder = '—'

const appVersion = computed(() => currentVersion.value || info.value?.app_version || buildVersion)

const singBoxVersion = computed(() => {
  const version = info.value?.sing_box_version
  if (!version || version === 'unknown') return t('settings.about.notInstalled')
  return version
})

/**
 * e.g. "OpenWrt 23.05.2", falling back to the coarse family. A host that
 * reports neither (a macOS dev machine) yields an empty string, which drops the
 * row entirely rather than displaying a useless literal "unknown".
 */
const platform = computed(() => {
  if (!info.value) return ''
  const { distribution, system_type: family } = info.value
  if (distribution) return distribution
  return family === 'unknown' ? '' : family
})

/** e.g. "linux/arm64 · 4 cores" */
const architecture = computed(() => {
  if (!info.value) return placeholder
  const target = `${info.value.os}/${info.value.arch}`
  const cores = info.value.cpu_cores
  return cores > 0 ? `${target} · ${t('settings.about.cores', { count: cores }, cores)}` : target
})

/** Rows are filtered so a host that reports nothing useful shows no blanks. */
const rows = computed(() => {
  const entries = [
    { key: 'app', label: t('settings.about.app'), value: appVersion.value, mono: true },
    { key: 'singBox', label: t('settings.about.singBox'), value: singBoxVersion.value, mono: true },
    { key: 'platform', label: t('settings.about.platform'), value: platform.value, mono: false },
    { key: 'arch', label: t('settings.about.architecture'), value: architecture.value, mono: true },
    { key: 'kernel', label: t('settings.about.kernel'), value: info.value?.kernel ?? '', mono: true },
    { key: 'hostname', label: t('settings.about.hostname'), value: info.value?.hostname ?? '', mono: true },
    {
      key: 'serviceBackend',
      label: t('settings.about.serviceBackend'),
      value: info.value?.service_backend ?? '',
      mono: true,
    },
  ]
  return entries.filter((row) => row.value && row.value !== placeholder)
})
</script>

<template>
  <div class="bg-white dark:bg-gray-800 rounded-surface shadow p-5">
    <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">
      {{ $t('settings.about.title') }}
    </h3>

    <!-- Host + version facts -->
    <div v-if="loading" class="h-10 flex items-center">
      <div class="animate-spin rounded-pill h-5 w-5 border-b-2 border-primary-600"></div>
    </div>
    <!--
      One row per fact, label left and value right. A two-column split was
      tried and rejected: the card sits in a grid column roughly 460px wide, so
      halving it truncates hostnames and long kernel strings.
    -->
    <dl v-else class="space-y-2">
      <div v-for="row in rows" :key="row.key" class="flex items-baseline justify-between gap-6 min-w-0">
        <dt class="text-sm text-gray-500 dark:text-gray-400 flex-shrink-0">{{ row.label }}</dt>
        <dd
          class="text-sm text-gray-900 dark:text-gray-100 truncate text-right"
          :class="row.mono ? 'font-mono' : ''"
          :title="row.value"
        >
          {{ row.value }}
        </dd>
      </div>
    </dl>

    <!-- Language -->
    <div class="mt-5 pt-5 border-t border-gray-200 dark:border-gray-700">
      <h4 class="text-sm font-semibold text-gray-900 dark:text-gray-100 mb-1">
        {{ $t('settings.language.title') }}
      </h4>
      <p class="text-sm text-gray-500 dark:text-gray-400 mb-3">{{ $t('settings.language.desc') }}</p>
      <LanguageSwitcher variant="full" />
    </div>

    <!-- Self-update -->
    <div class="mt-5 pt-5 border-t border-gray-200 dark:border-gray-700">
      <AppUpdateCard embedded />
    </div>

    <!--
      Dev-only: force the navigation layout. `isDevBuild` is a compile-time
      constant, so this block is dropped from release bundles.
    -->
    <div v-if="isDevBuild" class="mt-5 pt-5 border-t border-dashed border-amber-400/60">
      <div class="flex items-center gap-2 mb-1">
        <h4 class="text-sm font-semibold text-gray-900 dark:text-gray-100">
          {{ $t('settings.about.layout.title') }}
        </h4>
        <span
          class="px-1.5 py-0.5 rounded-pill bg-amber-400/20 text-[10px] font-semibold text-amber-700 dark:text-amber-300"
        >
          dev
        </span>
      </div>
      <p class="text-sm text-gray-500 dark:text-gray-400 mb-3">
        {{ $t('settings.about.layout.desc') }}
      </p>
      <div class="inline-flex rounded-control border border-gray-300 dark:border-gray-600 overflow-hidden">
        <button
          v-for="option in layoutOptions"
          :key="option.value"
          @click="setLayoutOverride(option.value)"
          :aria-pressed="layoutOverride === option.value"
          class="px-3 py-1.5 text-sm transition-colors cursor-pointer border-r border-gray-300 dark:border-gray-600 last:border-r-0"
          :class="
            layoutOverride === option.value
              ? 'bg-primary-600 text-white font-medium'
              : 'bg-white dark:bg-gray-900 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800'
          "
        >
          {{ $t(option.label) }}
        </button>
      </div>
    </div>
  </div>
</template>
