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
import StorageUsage from './StorageUsage.vue'
import type { SystemInfo } from '../types/api'
import { BugAntIcon, ClipboardDocumentIcon } from '@heroicons/vue/24/outline'
import { PROJECT_URL, bugReportUrl, diagnosticsText, type Diagnostics } from '../utils/bugReport'
import { writeTextToClipboard } from '../utils/clipboard'

const { t } = useI18n()
const notify = useNotify()

const info = ref<SystemInfo | null>(null)
const loading = ref(true)

// The update panel is the authority on the running panel version — it is what
// the self-update flow rewrites. /system/info agrees, but only until an update
// finishes without a page reload.
const { currentVersion, status: updateStatus } = useAppUpdate()

// Appearance preferences apply immediately and persist in this browser.
const { layoutOverride, setLayoutOverride, authEnabled } = useDeployment()

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

/**
 * What a bug report is filled in with: the facts this card already shows,
 * minus the hostname. An issue is public and permanent, and a hostname can
 * name a person or a network without helping anyone diagnose anything.
 *
 * Computed, so the link is always built from what is on screen right now —
 * including a version that changed under a self-update without a reload.
 */
const diagnostics = computed<Diagnostics>(() => ({
  panelVersion: appVersion.value,
  coreVersion: info.value?.sing_box_version,
  distribution: info.value?.distribution,
  systemType: info.value?.system_type,
  os: info.value?.os,
  arch: info.value?.arch,
  cpuCores: info.value?.cpu_cores,
  kernel: info.value?.kernel,
  serviceBackend: info.value?.service_backend,
  // Read from the raw status, not the composable's `selfUpdate`: that one
  // defaults to "tarball" until the backend answers, and a report must not
  // state an install method nobody measured.
  installMethod: updateStatus.value?.self_update?.method,
  authEnabled: authEnabled.value,
  userAgent: navigator.userAgent,
}))

const reportUrl = computed(() => bugReportUrl(diagnostics.value))

const copyDiagnostics = async () => {
  try {
    await writeTextToClipboard(diagnosticsText(diagnostics.value))
    notify.success(t('settings.about.report.copied'))
  } catch (err) {
    notify.apiError(err, t('common.copyFailed'))
  }
}

const appVersion = computed(() => currentVersion.value || info.value?.app_version || buildVersion)

/**
 * "Not installed" is a claim, so it is only made when the backend actually said
 * so. If the fetch failed there is no information at all — returning the
 * placeholder drops the row (see `rows`) rather than telling an operator with a
 * perfectly healthy sing-box that it is missing.
 */
const singBoxVersion = computed(() => {
  if (!info.value) return placeholder
  const version = info.value.sing_box_version
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
  <div class="bg-white dark:bg-gray-800 rounded-surface shadow p-4">
    <div class="flex items-center justify-between gap-4 mb-4">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">
        {{ $t('settings.about.title') }}
      </h3>
      <a
        :href="PROJECT_URL"
        target="_blank"
        rel="noopener noreferrer"
        class="inline-flex items-center gap-1.5 text-sm text-gray-500 hover:text-primary-600 dark:text-gray-400 dark:hover:text-primary-400 transition-colors"
        :title="PROJECT_URL"
      >
        <!-- GitHub mark (simple-icons, CC0). Heroicons has no brand icons. -->
        <svg class="h-4 w-4" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
          <path
            d="M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61C4.422 18.07 3.633 17.7 3.633 17.7c-1.087-.744.084-.729.084-.729 1.205.084 1.838 1.236 1.838 1.236 1.07 1.835 2.809 1.305 3.495.998.108-.776.417-1.305.76-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23.96-.267 1.98-.399 3-.405 1.02.006 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 22.092 24 17.592 24 12.297c0-6.627-5.373-12-12-12"
          />
        </svg>
        {{ $t('settings.about.source') }}
      </a>
    </div>

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
      <div v-for="row in rows" :key="row.key" class="flex items-baseline justify-between gap-4 min-w-0">
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

    <!-- Storage — dropped entirely when the host reports no filesystems. -->
    <StorageUsage v-if="!loading" :disks="info?.disks ?? []" />

    <!--
      Report a bug. Lives in this card because this card IS the report's
      contents: what is running, and on what. The link opens GitHub's own form
      with those fields filled in — the panel sends nothing itself, and the
      operator reads the issue before submitting it as themselves.

      "Copy" covers what the link cannot: adding the same block to an issue
      that already exists, or to a chat.
    -->
    <div v-if="!loading" class="mt-5 pt-5 border-t border-gray-200 dark:border-gray-700">
      <h4 class="text-sm font-semibold text-gray-900 dark:text-gray-100 mb-1">
        {{ $t('settings.about.report.title') }}
      </h4>
      <p class="text-sm text-gray-500 dark:text-gray-400 mb-3">{{ $t('settings.about.report.desc') }}</p>
      <div class="flex flex-wrap gap-2">
        <a
          :href="reportUrl"
          target="_blank"
          rel="noopener noreferrer"
          class="inline-flex items-center gap-1.5 rounded-control bg-primary-600 px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-primary-700"
        >
          <BugAntIcon class="h-4 w-4" />
          {{ $t('settings.about.report.open') }}
        </a>
        <button
          type="button"
          class="inline-flex items-center gap-1.5 rounded-control border border-gray-300 bg-white px-3 py-1.5 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-300 dark:hover:bg-gray-800"
          @click="copyDiagnostics"
        >
          <ClipboardDocumentIcon class="h-4 w-4" />
          {{ $t('settings.about.report.copy') }}
        </button>
      </div>
    </div>

    <!-- Language -->
    <div class="mt-5 pt-5 border-t border-gray-200 dark:border-gray-700">
      <h4 class="text-sm font-semibold text-gray-900 dark:text-gray-100 mb-1">
        {{ $t('settings.language.title') }}
      </h4>
      <p class="text-sm text-gray-500 dark:text-gray-400 mb-3">{{ $t('settings.language.desc') }}</p>
      <LanguageSwitcher variant="full" />
    </div>

    <!-- Navigation preference -->
    <div class="mt-5 pt-5 border-t border-gray-200 dark:border-gray-700">
      <div class="flex items-center gap-2 mb-1">
        <h4 id="navigation-layout-label" class="text-sm font-semibold text-gray-900 dark:text-gray-100">
          {{ $t('settings.about.layout.title') }}
        </h4>
      </div>
      <p class="text-sm text-gray-500 dark:text-gray-400 mb-3">
        {{ $t('settings.about.layout.desc') }}
      </p>
      <div role="group" aria-labelledby="navigation-layout-label" class="inline-flex rounded-control border border-gray-300 dark:border-gray-600 overflow-hidden">
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

    <!-- Self-update -->
    <div class="mt-5 pt-5 border-t border-gray-200 dark:border-gray-700">
      <AppUpdateCard embedded />
    </div>

  </div>
</template>
