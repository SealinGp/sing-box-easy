/**
 * Parsing for a single sing-box / syslog line in the Logs viewer.
 *
 * Two problems this solves, both observed on a real OpenWrt box:
 *
 * 1. sing-box colours its output with ANSI SGR escapes. Rendered as text they
 *    show up literally ("[31mFATAL[0m"), which is noise on every line.
 * 2. At `level: debug` sing-box logs every DNS lookup, so a `FATAL start
 *    service` is one line among hundreds of identical-looking ones and is
 *    effectively invisible — which is exactly when the operator needs it.
 */

export type LogLevel = 'fatal' | 'error' | 'warn' | 'info' | 'debug' | 'trace'

export interface ParsedLogLine {
  level: LogLevel
  /** The line with ANSI escapes removed. */
  text: string
}

// SGR sequences: ESC [ <params> m. Written with an explicit  rather than a
// literal escape so the source stays copy-pasteable.
const ANSI_SGR = /?\[[0-9;]*m/g

/** Removes ANSI colour escapes, including ones whose ESC byte was lost in transit. */
export function stripAnsi(line: string): string {
  return line.replace(ANSI_SGR, '')
}

/**
 * Level tokens as sing-box emits them: uppercase, standalone, and immediately
 * followed by either whitespace or its `[0025]` elapsed-time suffix. Requiring
 * that shape is what stops the word "error" inside a message body from
 * promoting the whole line.
 */
const LEVEL_TOKEN = /\b(FATAL|ERROR|WARN|INFO|DEBUG|TRACE)\b(?=[\s[])/

/**
 * procd reports a crash loop at daemon.info, but a service that cannot stay up
 * is the single most important thing in the log. Severity follows what
 * happened, not the syslog facility.
 */
const CRASH_LOOP = /in a crash loop/i

/** A fatal abort during startup, as opposed to a fatal at any other time. */
const START_SERVICE_FATAL = /FATAL.*start service/i

/**
 * The panel's own log at info level and above is zap's production encoder:
 * one JSON object per line, with a LOWERCASE level. Nothing in the sing-box
 * path matches that shape, so without this every line of the panel's log —
 * errors included — renders as undifferentiated grey `info`.
 *
 * Only attempted for lines that start with `{`, so this costs one character
 * comparison on the sing-box feed. In dev the panel uses zap's development
 * encoder instead, which emits an uppercase token the normal path already
 * handles.
 */
const ZAP_LEVELS: Record<string, LogLevel> = {
  fatal: 'fatal',
  panic: 'fatal',
  dpanic: 'fatal',
  error: 'error',
  warn: 'warn',
  info: 'info',
  debug: 'debug',
}

function parseStructured(text: string): ParsedLogLine | null {
  if (text.charCodeAt(0) !== 0x7b /* { */) return null

  let entry: Record<string, unknown>
  try {
    entry = JSON.parse(text)
  } catch {
    // A line that merely starts with a brace. Fall through to the token scan.
    return null
  }
  if (typeof entry.level !== 'string') return null

  const level = ZAP_LEVELS[entry.level.toLowerCase()]
  if (!level) return null

  // Rendered back as a readable line rather than as raw JSON: the viewer is
  // for reading, and `{"level":"info","ts":...,"caller":...}` buries the one
  // field anybody is looking for.
  const parts = [
    typeof entry.timestamp === 'string' ? entry.timestamp : '',
    entry.level.toUpperCase(),
    typeof entry.caller === 'string' ? entry.caller : '',
    typeof entry.msg === 'string' ? entry.msg : '',
  ].filter(Boolean)

  // Anything zap attached beyond the standard keys is context the operator
  // asked for by logging it, so it is kept rather than dropped.
  const extras = Object.entries(entry)
    .filter(([key]) => !['level', 'timestamp', 'ts', 'caller', 'msg', 'stacktrace'].includes(key))
    .map(([key, value]) => `${key}=${typeof value === 'string' ? value : JSON.stringify(value)}`)

  return { level, text: [...parts, ...extras].join(' ') }
}

export function parseLogLine(line: string): ParsedLogLine {
  const text = stripAnsi(line)

  if (CRASH_LOOP.test(text)) {
    return { level: 'fatal', text }
  }

  const structured = parseStructured(text)
  if (structured) return structured

  const token = text.match(LEVEL_TOKEN)?.[1]
  if (!token) {
    return { level: 'info', text }
  }

  return { level: token.toLowerCase() as LogLevel, text }
}

/**
 * Whether this line means sing-box failed to start.
 *
 * Deliberately narrow. An ordinary ERROR — a urltest timeout, a failed DNS
 * lookup — is noisy but recoverable; banner-ing those would train the operator
 * to ignore the banner, which defeats the point.
 */
export function isStartupFailure(line: string): boolean {
  const text = stripAnsi(line)
  return START_SERVICE_FATAL.test(text) || CRASH_LOOP.test(text)
}

/* ── Row view ────────────────────────────────────────────────────────────
 *
 * The viewer draws each line as a row: level badge, time, message. That needs
 * the line taken apart rather than merely classified, and what has to come off
 * depends on where the line travelled:
 *
 *   log.output file   +0000 2026-10-07 08:38:38 ERROR [id 24ms] connection: …
 *   journald          Oct 07 16:38:38 host sing-box[123]: +0000 2026-… ERROR …
 *   logread (OpenWrt) Wed Oct  7 16:38:38 2026 daemon.err sing-box[123]: +0000 2026-… ERROR …
 *
 * The syslog prefix and sing-box's own timestamp say the same thing twice, in
 * two time zones, ahead of the only part anyone reads.
 */

export interface LogEntry {
  level: LogLevel
  /** HH:MM:SS, or '' when the line carries no time. */
  time: string
  message: string
}

// logread: `Wed Oct  7 16:38:38 2026 daemon.err sing-box[1]: msg`
// journald: `Oct 07 16:38:38 host sing-box[1]: msg`
const SYSLOG_PREFIX =
  /^(?:[A-Z][a-z]{2}\s+)?[A-Z][a-z]{2}\s+\d{1,2}\s+(\d{2}:\d{2}:\d{2})(?:\s+\d{4})?\s+\S+\s+[^\s:]+:\s+([\s\S]*)$/

// sing-box's own header: optional zone, optional date+time, then the level.
const SINGBOX_HEADER =
  /^(?:[+-]\d{4}\s+)?(?:\d{4}-\d{2}-\d{2}\s+(\d{2}:\d{2}:\d{2})\s+)?(FATAL|PANIC|ERROR|WARN|INFO|DEBUG|TRACE)\s+([\s\S]*)$/

const ISO_CLOCK = /T(\d{2}:\d{2}:\d{2})/

/**
 * Splits one raw line into level, time and message.
 *
 * When a line has both a syslog time and sing-box's own, the SYSLOG one wins:
 * it is the host's local clock — the one on the operator's wall — whereas
 * sing-box frequently logs in UTC, and an 8-hour disagreement between the row
 * and the clock reads as a broken viewer.
 *
 * A line that matches neither shape is returned whole as an `info` message:
 * guessing at a prefix to strip would eat part of a message that only looked
 * like one.
 */
export function splitLogLine(line: string): LogEntry {
  const text = stripAnsi(line)

  if (CRASH_LOOP.test(text)) {
    return { level: 'fatal', time: '', message: text }
  }

  const structured = parseStructuredEntry(text)
  if (structured) return structured

  let body = text
  let time = ''
  const syslog = SYSLOG_PREFIX.exec(text)
  if (syslog) {
    time = syslog[1] ?? ''
    body = syslog[2] ?? ''
  }

  const header = SINGBOX_HEADER.exec(body)
  if (header) {
    const name = header[2] ?? ''
    const token = name === 'PANIC' ? 'fatal' : (name.toLowerCase() as LogLevel)
    return { level: token, time: time || header[1] || '', message: header[3] ?? '' }
  }

  const token = body.match(LEVEL_TOKEN)?.[1]
  return { level: token ? (token.toLowerCase() as LogLevel) : 'info', time, message: body }
}

function parseStructuredEntry(text: string): LogEntry | null {
  if (text.charCodeAt(0) !== 0x7b /* { */) return null
  let entry: Record<string, unknown>
  try {
    entry = JSON.parse(text)
  } catch {
    return null
  }
  if (typeof entry.level !== 'string') return null
  const level = ZAP_LEVELS[entry.level.toLowerCase()]
  if (!level) return null

  const stamp = typeof entry.timestamp === 'string' ? entry.timestamp : typeof entry.ts === 'string' ? entry.ts : ''
  const extras = Object.entries(entry)
    .filter(([key]) => !['level', 'timestamp', 'ts', 'caller', 'msg', 'stacktrace'].includes(key))
    .map(([key, value]) => `${key}=${typeof value === 'string' ? value : JSON.stringify(value)}`)
  const message = [
    typeof entry.caller === 'string' ? entry.caller : '',
    typeof entry.msg === 'string' ? entry.msg : '',
    ...extras,
  ].filter(Boolean).join(' ')

  return { level, time: ISO_CLOCK.exec(stamp)?.[1] ?? '', message }
}

/** Severity order, least to most. */
export const LOG_LEVELS: readonly LogLevel[] = ['trace', 'debug', 'info', 'warn', 'error', 'fatal']

/**
 * Keeps entries at or above `minimum` whose message matches `query`.
 *
 * The query is a regular expression; one that does not compile — the operator
 * is usually mid-keystroke — degrades to a literal match rather than emptying
 * the list. The indexes of the survivors are returned so the caller can keep
 * each row's sequence number.
 */
export function filterLogEntries(entries: readonly LogEntry[], minimum: LogLevel, query: string): number[] {
  const floor = LOG_LEVELS.indexOf(minimum)
  const trimmed = query.trim()
  let matches: ((text: string) => boolean) | null = null
  if (trimmed !== '') {
    try {
      const pattern = new RegExp(trimmed, 'i')
      matches = (text) => pattern.test(text)
    } catch {
      const needle = trimmed.toLowerCase()
      matches = (text) => text.toLowerCase().includes(needle)
    }
  }

  const kept: number[] = []
  entries.forEach((entry, index) => {
    if (LOG_LEVELS.indexOf(entry.level) < floor) return
    if (matches && !matches(entry.message)) return
    kept.push(index)
  })
  return kept
}
