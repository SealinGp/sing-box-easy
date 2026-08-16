<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { serviceControlService } from '../../services'
import { useToast } from 'primevue/usetoast'

const { t } = useI18n()
const toast = useToast()

// Bounded recent history + poll cadence. The viewer keeps only the most recent
// MAX_LINES lines so memory stays flat for a long-running, chatty proxy.
const MAX_LINES = 500
const POLL_MS = 1500
const INITIAL_LINES = 300

const lines = ref<string[]>([])
const cursor = ref('')
const source = ref<'journald' | 'syslog' | 'file' | 'none' | ''>('')
const polling = ref(true)
const autoScroll = ref(true)
const initialLoading = ref(true)
const errored = ref(false)

const logContainer = ref<HTMLElement | null>(null)
let timer: ReturnType<typeof setInterval> | null = null

const sourceNote = computed(() => {
  if (source.value === 'none') return t('logs.sourceNone')
  if (source.value === 'file') return t('logs.sourceFile')
  if (source.value === 'syslog') return t('logs.sourceSyslog')
  return ''
})

const scrollToBottom = async () => {
  await nextTick()
  const el = logContainer.value
  if (el) el.scrollTop = el.scrollHeight
}

// If the user scrolls up, stop yanking them back to the bottom; re-enable when
// they return to (near) the bottom.
const onScroll = () => {
  const el = logContainer.value
  if (!el) return
  const nearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 40
  autoScroll.value = nearBottom
}

const appendLines = (incoming: string[]) => {
  if (!incoming.length) return
  const combined = lines.value.concat(incoming)
  lines.value = combined.length > MAX_LINES ? combined.slice(combined.length - MAX_LINES) : combined
  if (autoScroll.value) void scrollToBottom()
}

const fetchOnce = async () => {
  try {
    // First call seeds a full window; subsequent calls fetch only what's new
    // via the journald cursor.
    const lineCount = cursor.value ? MAX_LINES : INITIAL_LINES
    const { data } = await serviceControlService.getServiceLogs(lineCount, cursor.value)
    source.value = data.source
    if (cursor.value) {
      appendLines(data.lines || [])
    } else {
      // Initial load: replace the buffer wholesale.
      lines.value = (data.lines || []).slice(-MAX_LINES)
      if (autoScroll.value) void scrollToBottom()
    }
    // Keep the previous cursor when none is returned (no new entries this tick).
    if (data.cursor) cursor.value = data.cursor
    errored.value = false
  } catch (err: any) {
    errored.value = true
    if (initialLoading.value) {
      toast.add({
        severity: 'error',
        summary: t('common.error'),
        detail: err.message || t('logs.toast.fetchFailed'),
        life: 3000,
      })
    }
  } finally {
    initialLoading.value = false
  }
}

const tick = () => {
  if (typeof document !== 'undefined' && document.hidden) return
  void fetchOnce()
}

const startPolling = () => {
  if (timer === null) timer = setInterval(tick, POLL_MS)
}
const stopPolling = () => {
  if (timer !== null) {
    clearInterval(timer)
    timer = null
  }
}

const togglePolling = () => {
  polling.value = !polling.value
  if (polling.value) {
    startPolling()
  } else {
    stopPolling()
  }
}

const clearLogs = () => {
  lines.value = []
}

const jumpToBottom = () => {
  autoScroll.value = true
  void scrollToBottom()
}

onMounted(async () => {
  await fetchOnce()
  if (polling.value) startPolling()
})

onUnmounted(stopPolling)
</script>

<template>
  <div class="page-shell h-screen flex flex-col overflow-hidden">
    <!-- Header -->
    <div class="flex justify-between items-center mb-4 shrink-0">
      <h2 class="text-3xl font-bold text-gray-900 dark:text-gray-100">{{ $t('logs.title') }}</h2>
      <div class="flex items-center gap-3">
        <span class="text-xs text-gray-500 dark:text-gray-400">
          {{ $t('logs.lineCount', { n: lines.length, max: MAX_LINES }) }}
        </span>

        <button
          @click="jumpToBottom"
          :disabled="autoScroll"
          class="px-3 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-control hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors disabled:opacity-50"
        >
          {{ $t('logs.jumpToBottom') }}
        </button>

        <button
          @click="clearLogs"
          class="px-3 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-control hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
        >
          {{ $t('logs.clear') }}
        </button>

        <button
          @click="togglePolling"
          class="px-4 py-2 text-sm font-medium rounded-control transition-colors"
          :class="polling
            ? 'text-amber-700 dark:text-amber-300 bg-amber-50 dark:bg-amber-900/20 border border-amber-300 dark:border-amber-700 hover:bg-amber-100 dark:hover:bg-amber-900/40'
            : 'text-white bg-primary-600 hover:bg-primary-700'"
        >
          {{ polling ? $t('logs.pause') : $t('logs.resume') }}
        </button>
      </div>
    </div>

    <!-- Status row -->
    <div class="mb-2 flex items-center gap-3 text-sm shrink-0">
      <span class="flex items-center gap-1.5">
        <span
          class="w-2 h-2 rounded-pill"
          :class="polling && !errored ? 'bg-green-500 animate-pulse' : errored ? 'bg-red-500' : 'bg-gray-400'"
        ></span>
        <span class="text-gray-600 dark:text-gray-400">
          {{ errored ? $t('logs.disconnected') : polling ? $t('logs.live') : $t('logs.paused') }}
        </span>
      </span>
      <span v-if="sourceNote" class="text-xs text-amber-600 dark:text-amber-400">{{ sourceNote }}</span>
    </div>

    <!-- Log surface -->
    <div
      ref="logContainer"
      @scroll="onScroll"
      class="flex-1 min-h-0 overflow-y-auto bg-gray-900 dark:bg-black rounded-surface shadow-float p-4 font-mono text-xs leading-relaxed"
    >
      <div v-if="initialLoading" class="flex items-center justify-center h-full">
        <div class="animate-spin rounded-pill h-7 w-7 border-b-2 border-primary-500"></div>
      </div>
      <div v-else-if="lines.length === 0" class="flex items-center justify-center h-full text-gray-500">
        {{ $t('logs.empty') }}
      </div>
      <div v-else>
        <div
          v-for="(line, i) in lines"
          :key="i"
          class="whitespace-pre-wrap break-all text-gray-200 hover:bg-white/5 px-1 -mx-1 rounded"
        >{{ line }}</div>
      </div>
    </div>
  </div>
</template>
