<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from 'primevue/usetoast'
import { ArrowDownTrayIcon, ChevronDoubleDownIcon, PauseIcon, PlayIcon, TrashIcon } from '@heroicons/vue/24/outline'
import { parseLogLine, isStartupFailure, splitLogLine, filterLogEntries, type LogLevel } from '../../utils/logLine'
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
/**
 * Lines received since the buffer was last replaced. Not displayed — it gives
 * each row a render KEY that stays with its line. The window is bounded, so a
 * row's index changes every time an older line falls off; keying by index
 * would make Vue re-patch all 500 rows on every append instead of adding one.
 */
const received = ref(0)
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

  received.value += incoming.length
  const combined = lines.value.concat(incoming)
  lines.value = combined.length > MAX_LINES ? combined.slice(combined.length - MAX_LINES) : combined
  if (autoScroll.value) void scrollToBottom()
}

const replaceLines = (incoming: string[]) => {
  for (const line of incoming) {
    if (isStartupFailure(line)) startupFailure.value = parseLogLine(line).text
  }
  lines.value = incoming
  received.value = incoming.length
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
  received.value = 0
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
 * Lines are split once here rather than per-render: a burst re-renders the
 * whole window.
 */
const entries = computed(() => lines.value.map(splitLogLine))

/**
 * The level filter is a FLOOR, not a single level: someone who picks "warn" is
 * asking what went wrong, and an error is more of an answer to that than a
 * warning is.
 */
const minLevel = ref<LogLevel>('trace')
const LEVEL_OPTIONS: { value: LogLevel; labelKey: string }[] = [
  { value: 'trace', labelKey: 'logs.level.all' },
  { value: 'debug', labelKey: 'logs.level.debug' },
  { value: 'info', labelKey: 'logs.level.info' },
  { value: 'warn', labelKey: 'logs.level.warn' },
  { value: 'error', labelKey: 'logs.level.error' },
]
const query = ref('')

/** Rows on screen, each with the stable key of its line (see `received`). */
const rows = computed(() => {
  const first = received.value - entries.value.length + 1
  return filterLogEntries(entries.value, minLevel.value, query.value).flatMap((index) => {
    const entry = entries.value[index]
    return entry ? [{ seq: first + index, entry }] : []
  })
})

const filtered = computed(() => minLevel.value !== 'trace' || query.value.trim() !== '')

const LEVEL_BADGE: Record<LogLevel, string> = {
  fatal: 'text-white bg-red-600',
  error: 'text-red-700 bg-red-100 dark:text-red-300 dark:bg-red-900/40',
  warn: 'text-amber-700 bg-amber-100 dark:text-amber-300 dark:bg-amber-900/40',
  info: 'text-sky-700 bg-sky-100 dark:text-sky-300 dark:bg-sky-900/40',
  debug: 'text-gray-500 bg-gray-100 dark:text-gray-400 dark:bg-gray-700/60',
  trace: 'text-gray-400 bg-gray-100 dark:text-gray-500 dark:bg-gray-700/60',
}

const LEVEL_TEXT: Record<LogLevel, string> = {
  fatal: 'text-red-700 dark:text-red-300 font-semibold',
  error: 'text-red-700 dark:text-red-300',
  warn: 'text-amber-800 dark:text-amber-200',
  info: 'text-gray-900 dark:text-gray-100',
  debug: 'text-gray-500 dark:text-gray-400',
  trace: 'text-gray-400 dark:text-gray-500',
}

/**
 * Saves what is on screen — the filtered rows, not the raw window — because
 * that is what the operator just narrowed down to and wants to attach to a
 * bug report.
 */
const downloadLogs = () => {
  const text = rows.value.map(({ entry }) => `${entry.time} ${entry.level.toUpperCase()} ${entry.message}`.trim()).join('\n')
  const url = URL.createObjectURL(new Blob([text], { type: 'text/plain;charset=utf-8' }))
  const link = document.createElement('a')
  link.href = url
  link.download = `${feed.value === 'app' ? 'sing-box-easy' : 'sing-box'}-${new Date().toISOString().replace(/[:.]/g, '-')}.log`
  link.click()
  URL.revokeObjectURL(url)
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
  received.value = 0
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

    <!-- Filters -->
    <div class="mb-2 flex shrink-0 flex-wrap items-center gap-2">
      <select
        v-model="minLevel"
        :aria-label="$t('logs.level.label')"
        class="h-8 rounded-control border border-gray-300 bg-white px-2 text-sm text-gray-900 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100"
      >
        <option v-for="option in LEVEL_OPTIONS" :key="option.value" :value="option.value">{{ $t(option.labelKey) }}</option>
      </select>
      <input
        v-model="query"
        type="search"
        :placeholder="$t('logs.searchPlaceholder')"
        :aria-label="$t('logs.searchPlaceholder')"
        class="h-8 min-w-0 flex-[1_1_220px] rounded-control border border-gray-300 bg-white px-2.5 text-sm text-gray-900 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100"
      />

      <!--
        The actions sit beside the filters they act on, as icons. There is no
        page title above them: the nav already says which page this is, and a
        heading row spent a fifth of a short screen saying it again. Each icon
        carries its name as both `aria-label` and `title`, so it is announced
        and shows on hover.
      -->
      <span class="whitespace-nowrap text-xs tabular-nums text-gray-500 dark:text-gray-400">
        {{ filtered
          ? $t('logs.filteredCount', { shown: rows.length, n: lines.length })
          : $t('logs.lineCount', { n: lines.length, max: MAX_LINES }) }}
      </span>
      <div class="flex items-center gap-1">
        <button type="button" class="log-action" :disabled="rows.length === 0" :aria-label="$t('logs.download')" :title="$t('logs.download')" @click="downloadLogs">
          <ArrowDownTrayIcon class="h-4 w-4" />
        </button>
        <button type="button" class="log-action" :disabled="autoScroll" :aria-label="$t('logs.jumpToBottom')" :title="$t('logs.jumpToBottom')" @click="jumpToBottom">
          <ChevronDoubleDownIcon class="h-4 w-4" />
        </button>
        <button type="button" class="log-action" :disabled="lines.length === 0" :aria-label="$t('logs.clear')" :title="$t('logs.clear')" @click="clearLogs">
          <TrashIcon class="h-4 w-4" />
        </button>
        <button
          type="button"
          class="log-action"
          :class="{ 'is-paused': !streaming }"
          :aria-pressed="!streaming"
          :aria-label="streaming ? $t('logs.pause') : $t('logs.resume')"
          :title="streaming ? $t('logs.pause') : $t('logs.resume')"
          @click="toggleStreaming"
        >
          <PauseIcon v-if="streaming" class="h-4 w-4" />
          <PlayIcon v-else class="h-4 w-4" />
        </button>
      </div>
    </div>

    <!--
      Log surface. Rows rather than a terminal block: the level and the time
      are the two things scanned for, so they get their own aligned columns
      instead of being somewhere inside a wrapped line.
    -->
    <div
      ref="logContainer"
      @scroll="onScroll"
      class="flex-1 min-h-0 overflow-y-auto rounded-surface border border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800"
    >
      <div v-if="initialLoading" class="flex items-center justify-center h-full">
        <div class="animate-spin rounded-pill h-7 w-7 border-b-2 border-primary-500"></div>
      </div>
      <div v-else-if="lines.length === 0" class="flex items-center justify-center h-full text-sm text-gray-500 dark:text-gray-400">
        {{ feed === 'app' ? $t('logs.emptyApp') : $t('logs.empty') }}
      </div>
      <div v-else-if="rows.length === 0" class="flex items-center justify-center h-full text-sm text-gray-500 dark:text-gray-400">
        {{ $t('logs.noMatch') }}
      </div>
      <ol v-else class="divide-y divide-gray-100 dark:divide-gray-700/60">
        <li
          v-for="row in rows"
          :key="row.seq"
          class="grid grid-cols-[auto_minmax(0,1fr)] gap-x-3 px-3 py-1.5 hover:bg-gray-50 dark:hover:bg-gray-700/40 sm:grid-cols-[4rem_4.5rem_minmax(0,1fr)]"
        >
          <span class="self-start rounded px-1.5 py-px text-center text-[10px] font-semibold uppercase tracking-wide" :class="LEVEL_BADGE[row.entry.level]">
            {{ row.entry.level }}
          </span>
          <span class="hidden text-xs tabular-nums text-gray-500 dark:text-gray-400 sm:block">{{ row.entry.time }}</span>
          <span class="whitespace-pre-wrap break-all font-mono text-xs leading-relaxed" :class="LEVEL_TEXT[row.entry.level]">{{ row.entry.message }}</span>
        </li>
      </ol>
    </div>
  </div>
</template>

<style scoped>
.log-action {
  display: inline-flex;
  width: 2rem;
  height: 2rem;
  flex: none;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--color-border-dark);
  border-radius: var(--radius-control, 10px);
  color: var(--color-text-secondary);
  transition:
    color 0.15s ease,
    border-color 0.15s ease,
    background-color 0.15s ease;
}
.log-action:hover:not(:disabled) {
  border-color: var(--color-primary);
  color: var(--color-primary);
}
.log-action:focus-visible {
  outline: 2px solid var(--color-primary);
  outline-offset: 2px;
}
.log-action:disabled {
  cursor: default;
  opacity: 0.4;
}
/* Paused is a state, not just a button: it stays lit until streaming resumes. */
.log-action.is-paused {
  border-color: var(--color-primary);
  background: color-mix(in srgb, var(--color-primary) 12%, transparent);
  color: var(--color-primary);
}
</style>
