<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from 'primevue/usetoast'
import { parseLogLine, isStartupFailure, type LogLevel } from '../../utils/logLine'
import { useLogStream, MAX_LINES, type LogFeed } from '../../composables/useLogStream'

const { t } = useI18n()
const toast = useToast()

/**
 * Which log is on screen.
 *
 * Two feeds, one viewer. They answer different questions — "what is the proxy
 * doing?" versus "what is the panel doing?" — and the second was previously
 * unanswerable without shell access, which is a poor state for a tool whose
 * job is to remove the need for shell access.
 */
const feed = ref<LogFeed>('singbox')

const TABS: { value: LogFeed; labelKey: string }[] = [
  { value: 'singbox', labelKey: 'logs.tabs.singbox' },
  { value: 'app', labelKey: 'logs.tabs.app' },
]

const lines = ref<string[]>([])
const streaming = ref(true)
const autoScroll = ref(true)

const logContainer = ref<HTMLElement | null>(null)

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

/**
 * The most recent startup failure, kept OUTSIDE the bounded `lines` window.
 *
 * At `level: debug` sing-box logs every DNS lookup, so a `FATAL start service`
 * is pushed out of a 500-line buffer within seconds — the one line the operator
 * needs is the first one lost. Pinning it means it survives both the ring
 * buffer and scrolling.
 */
const startupFailure = ref('')

const dismissStartupFailure = () => {
  startupFailure.value = ''
}

const appendLines = (incoming: string[]) => {
  if (!incoming.length) return

  // Scan before truncation, so a failure in a burst larger than MAX_LINES is
  // still caught.
  for (const line of incoming) {
    if (isStartupFailure(line)) startupFailure.value = parseLogLine(line).text
  }

  const combined = lines.value.concat(incoming)
  lines.value = combined.length > MAX_LINES ? combined.slice(combined.length - MAX_LINES) : combined
  if (autoScroll.value) void scrollToBottom()
}

const replaceLines = (incoming: string[]) => {
  for (const line of incoming) {
    if (isStartupFailure(line)) startupFailure.value = parseLogLine(line).text
  }
  lines.value = incoming
  if (autoScroll.value) void scrollToBottom()
}

/**
 * Transport, source and connection state all live in the composable — this view
 * only decides what to do with the lines.
 */
const stream = useLogStream({ feed, onLines: appendLines, onReplace: replaceLines })
const { source, transport, errored, initialLoading } = stream

/**
 * Switching tabs blanks the buffer immediately.
 *
 * The composable restarts the feed and will replace the lines when the new
 * window arrives, but that is a round trip away. Leaving the old log on screen
 * under the new tab's label for even a moment presents one service's output as
 * another's, which is the single most misleading thing this page could do.
 */
const selectFeed = (next: LogFeed) => {
  if (feed.value === next) return
  lines.value = []
  startupFailure.value = ''
  feed.value = next
}

const sourceNote = computed(() => {
  if (source.value === 'none') return t('logs.sourceNone')
  if (source.value === 'file') return t('logs.sourceFile')
  if (source.value === 'syslog') return t('logs.sourceSyslog')
  // Always shown for the panel's own log, because its limitation is permanent
  // rather than a misconfiguration: the buffer is the life of the process.
  if (source.value === 'memory') return t('logs.sourceMemory')
  return ''
})

/**
 * Which transport is carrying the feed.
 *
 * Surfaced rather than hidden because the two behave differently in a way the
 * operator can see: a streamed line appears the moment sing-box writes it, a
 * polled one can be up to 1.5s late. Someone timing a reconnect needs to know
 * which they are looking at.
 */
const transportNote = computed(() =>
  transport.value === 'poll' ? t('logs.transport.poll') : '',
)

/**
 * Lines are parsed once here rather than per-render: a burst re-renders the
 * whole window.
 */
const parsedLines = computed(() => lines.value.map(parseLogLine))

const LEVEL_CLASS: Record<LogLevel, string> = {
  fatal: 'text-red-300 font-semibold bg-red-950/40',
  error: 'text-red-400',
  warn: 'text-amber-300',
  info: 'text-gray-200',
  debug: 'text-gray-500',
  trace: 'text-gray-600',
}

const toggleStreaming = () => {
  streaming.value = !streaming.value
  if (streaming.value) {
    void stream.start()
  } else {
    stream.stop()
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
  await stream.start()
  if (errored.value) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: t('logs.toast.fetchFailed'),
      life: 3000,
    })
  }
})

// The composable unregisters its own stream and timer on unmount — including
// aborting the fetch, which is what lets the server kill the journalctl child
// it is holding open for this tab.
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
          @click="toggleStreaming"
          class="px-4 py-2 text-sm font-medium rounded-control transition-colors"
          :class="streaming
            ? 'text-amber-700 dark:text-amber-300 bg-amber-50 dark:bg-amber-900/20 border border-amber-300 dark:border-amber-700 hover:bg-amber-100 dark:hover:bg-amber-900/40'
            : 'text-white bg-primary-600 hover:bg-primary-700'"
        >
          {{ streaming ? $t('logs.pause') : $t('logs.resume') }}
        </button>
      </div>
    </div>

    <!--
      Feed switch. Buttons rather than <TabNav>, which is route-driven: these
      two views share one viewer and one set of controls, so a route per feed
      would remount the whole page — tearing down the stream and re-fetching a
      window — to change a single variable.

      Styled to match TabNav all the same, because to the reader they are the
      same affordance as the tabs on every other page.
    -->
    <div class="mb-2 shrink-0 border-b border-gray-200 dark:border-gray-700">
      <nav class="-mb-px flex space-x-4" role="tablist">
        <button
          v-for="tab in TABS"
          :key="tab.value"
          type="button"
          role="tab"
          :aria-selected="feed === tab.value"
          @click="selectFeed(tab.value)"
          :class="[
            'py-1 px-0.5 border-b-2 font-medium text-sm transition-colors cursor-pointer',
            feed === tab.value
              ? 'border-primary-500 text-primary-600 dark:text-primary-400'
              : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-300',
          ]"
        >
          {{ $t(tab.labelKey) }}
        </button>
      </nav>
    </div>

    <!-- Status row -->
    <div class="mb-2 flex items-center gap-3 text-sm shrink-0">
      <span class="flex items-center gap-1.5">
        <span
          class="w-2 h-2 rounded-pill"
          :class="streaming && !errored ? 'bg-green-500 animate-pulse' : errored ? 'bg-red-500' : 'bg-gray-400'"
        ></span>
        <span class="text-gray-600 dark:text-gray-400">
          {{ errored ? $t('logs.disconnected') : streaming ? $t('logs.live') : $t('logs.paused') }}
        </span>
      </span>
      <!-- Which transport is carrying the feed. A polled line can be up to
           1.5s late where a streamed one cannot, and someone timing a
           reconnect needs to know which they are watching. -->
      <span v-if="transportNote" class="text-xs text-gray-500 dark:text-gray-400">{{ transportNote }}</span>
      <span v-if="sourceNote" class="text-xs text-amber-600 dark:text-amber-400">{{ sourceNote }}</span>
    </div>

    <!--
      Pinned startup failure. Sits above the log surface because the line it
      reports is, by construction, the one most likely to have scrolled away.
    -->
    <div
      v-if="startupFailure && feed === 'singbox'"
      class="mb-2 shrink-0 rounded-surface border border-red-300 dark:border-red-800 bg-red-50 dark:bg-red-950/40 px-3 py-2"
    >
      <div class="flex items-start justify-between gap-3">
        <div class="min-w-0">
          <p class="text-sm font-semibold text-red-700 dark:text-red-300">
            {{ $t('logs.startupFailure.title') }}
          </p>
          <p class="mt-0.5 text-xs text-red-600 dark:text-red-400">
            {{ $t('logs.startupFailure.hint') }}
          </p>
          <pre class="mt-1.5 overflow-x-auto whitespace-pre-wrap break-all font-mono text-[11px] text-red-800 dark:text-red-200">{{ startupFailure }}</pre>
        </div>
        <button
          @click="dismissStartupFailure"
          class="shrink-0 text-xs text-red-600 dark:text-red-400 hover:underline cursor-pointer"
        >
          {{ $t('logs.startupFailure.dismiss') }}
        </button>
      </div>
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
        {{ feed === 'app' ? $t('logs.emptyApp') : $t('logs.empty') }}
      </div>
      <div v-else>
        <div
          v-for="(line, i) in parsedLines"
          :key="i"
          class="whitespace-pre-wrap break-all hover:bg-white/5 px-1 -mx-1 rounded"
          :class="LEVEL_CLASS[line.level]"
        >{{ line.text }}</div>
      </div>
    </div>
  </div>
</template>
