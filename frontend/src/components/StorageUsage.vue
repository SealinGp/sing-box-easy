<script setup lang="ts">
/**
 * Free space on the filesystems the panel writes to, shown in the Settings
 * "About" card.
 *
 * This exists because a full filesystem is invisible until it breaks something
 * unrelated-looking: SQLite cannot create its journal on a full disk and
 * reports "unable to open database file (14)", which reads like a permissions
 * bug. On OpenWrt in particular the root overlay is often tiny (a 289 MB loop
 * image is a common default) while the box has terabytes mounted elsewhere, so
 * the operator needs to see *which* filesystem is full, not just that a write
 * failed.
 */
import { computed } from 'vue'
import { formatBytes } from '../utils/formatBytes'
import type { DiskUsage } from '../types/api'

const props = defineProps<{ disks: DiskUsage[] }>()

/** Above this, writes are close enough to failing that the operator must act. */
const CRITICAL_PERCENT = 95
/** Above this, worth noticing before it becomes critical. */
const WARNING_PERCENT = 85

type Severity = 'critical' | 'warning' | 'ok'

function severityOf(disk: DiskUsage): Severity {
  if (disk.used_percent >= CRITICAL_PERCENT) return 'critical'
  if (disk.used_percent >= WARNING_PERCENT) return 'warning'
  return 'ok'
}

const rows = computed(() =>
  props.disks.map((disk) => ({
    ...disk,
    severity: severityOf(disk),
    // The mount point is the actionable identity ("/overlay" tells an OpenWrt
    // operator exactly what to fix); the queried path is the fallback.
    name: disk.mount_point || disk.path,
    // Clamped so a filesystem reporting >100% (root reserve exhausted) does not
    // overflow the track.
    width: Math.min(100, Math.max(0, disk.used_percent)),
  })),
)

const hasCritical = computed(() => rows.value.some((row) => row.severity === 'critical'))

const BAR_CLASS: Record<Severity, string> = {
  critical: 'bg-red-500',
  warning: 'bg-amber-500',
  ok: 'bg-primary-600',
}

const TEXT_CLASS: Record<Severity, string> = {
  critical: 'text-red-600 dark:text-red-400 font-medium',
  warning: 'text-amber-600 dark:text-amber-400',
  ok: 'text-gray-500 dark:text-gray-400',
}
</script>

<template>
  <div v-if="rows.length" class="mt-5 pt-5 border-t border-gray-200 dark:border-gray-700">
    <h4 class="text-sm font-semibold text-gray-900 dark:text-gray-100 mb-3">
      {{ $t('settings.about.storage.title') }}
    </h4>

    <div class="space-y-3">
      <div v-for="row in rows" :key="row.path" class="min-w-0">
        <div class="flex items-baseline justify-between gap-3 mb-1 min-w-0">
          <span
            class="text-sm font-mono text-gray-900 dark:text-gray-100 truncate"
            :title="row.device ? `${row.name} (${row.device})` : row.name"
          >
            {{ row.name }}
          </span>
          <span class="text-xs flex-shrink-0" :class="TEXT_CLASS[row.severity]">
            {{ $t('settings.about.storage.free', { free: formatBytes(row.free_bytes) }) }}
          </span>
        </div>

        <div
          class="h-1.5 w-full rounded-pill bg-gray-200 dark:bg-gray-700 overflow-hidden"
          role="progressbar"
          :aria-valuenow="Math.round(row.used_percent)"
          aria-valuemin="0"
          aria-valuemax="100"
          :aria-label="row.name"
        >
          <div
            class="h-full rounded-pill transition-all"
            :class="BAR_CLASS[row.severity]"
            :style="{ width: `${row.width}%` }"
          ></div>
        </div>

        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400 truncate">
          {{
            $t('settings.about.storage.detail', {
              used: formatBytes(row.used_bytes),
              total: formatBytes(row.total_bytes),
              percent: Math.round(row.used_percent),
            })
          }}
        </p>
      </div>
    </div>

    <!--
      Only shown when a filesystem is actually critical: the operator is about
      to hit write failures whose surface error names SQLite, not the disk.
    -->
    <p
      v-if="hasCritical"
      class="mt-3 text-xs text-red-600 dark:text-red-400 leading-relaxed"
    >
      {{ $t('settings.about.storage.criticalHint') }}
    </p>
  </div>
</template>
