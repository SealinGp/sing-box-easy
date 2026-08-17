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

export function parseLogLine(line: string): ParsedLogLine {
  const text = stripAnsi(line)

  if (CRASH_LOOP.test(text)) {
    return { level: 'fatal', text }
  }

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
