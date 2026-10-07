import { describe, expect, it } from 'bun:test'
import { parseLogLine, splitLogLine, filterLogEntries, stripAnsi, isStartupFailure } from './logLine'

// Samples are verbatim from a real OpenWrt box (bin/log.md), escapes included.
const FATAL =
  'Mon Aug 17 22:06:33 2026 daemon.err sing-box[5691]: [31mFATAL[0m[0025] start service: (initialize rule-set[23]: initial rule-set: sea-ruelsets-disney: Get "https://gh-proxy.com/...": timeout: no recent network activity)'
const CRASH_LOOP =
  'Mon Aug 17 22:06:33 2026 daemon.info procd: Instance sing-box::sing-box.main s in a crash loop 6 crashes, 25 seconds since last crash'
const ERROR =
  'Mon Aug 17 22:06:13 2026 daemon.err sing-box[5691]: +0000 2026-08-17 14:06:13 [31mERROR[0m outbound/urltest[自动选择]: timeout: no recent network activity'
const DEBUG =
  'Mon Aug 17 22:06:13 2026 daemon.err sing-box[5691]: +0000 2026-08-17 14:06:13 [37mDEBUG[0m dns: lookup domain ipaste.4ippi.ru'
const INFO =
  'Mon Aug 17 21:58:15 2026 daemon.err sing-box[23244]: +0000 2026-08-17 13:58:15 [36mINFO[0m inbound/direct[dns-in]: inbound packet connection'
const WARN =
  'Mon Aug 17 22:06:18 2026 daemon.err sing-box[5691]: [33mWARN[0m outbound: close outbound/hysteria2[...] take too much time to finish!'

describe('stripAnsi', () => {
  it('removes SGR colour escapes', () => {
    expect(stripAnsi('[31mFATAL[0m done')).toBe('FATAL done')
  })

  it('leaves plain text untouched', () => {
    expect(stripAnsi('nothing to strip')).toBe('nothing to strip')
  })

  it('handles multi-parameter sequences', () => {
    expect(stripAnsi('[38;5;31m3931007974[0m ok')).toBe('3931007974 ok')
  })
})

describe('parseLogLine', () => {
  it('classifies each sing-box level', () => {
    expect(parseLogLine(FATAL).level).toBe('fatal')
    expect(parseLogLine(ERROR).level).toBe('error')
    expect(parseLogLine(WARN).level).toBe('warn')
    expect(parseLogLine(INFO).level).toBe('info')
    expect(parseLogLine(DEBUG).level).toBe('debug')
  })

  it('strips escapes from the rendered text', () => {
    expect(parseLogLine(FATAL).text).not.toContain('')
  })

  it('treats a procd crash loop as fatal even though syslog tags it info', () => {
    // The severity that matters is what happened, not the syslog facility:
    // procd logs the crash loop at daemon.info.
    expect(parseLogLine(CRASH_LOOP).level).toBe('fatal')
  })

  it('defaults to info when no level token is present', () => {
    expect(parseLogLine('some line with no level at all').level).toBe('info')
  })

  it('does not misread a level word inside a message body', () => {
    // "error" appearing in prose must not upgrade the line's severity.
    const line = 'Mon Aug 17 22:06:13 2026 daemon.info sing-box[1]: INFO dns: no error reported'
    expect(parseLogLine(line).level).toBe('info')
  })

  it('is safe on an empty line', () => {
    expect(parseLogLine('').level).toBe('info')
    expect(parseLogLine('').text).toBe('')
  })
})

describe('isStartupFailure', () => {
  it('matches a fatal start-service abort', () => {
    expect(isStartupFailure(FATAL)).toBe(true)
  })

  it('matches a procd crash loop', () => {
    expect(isStartupFailure(CRASH_LOOP)).toBe(true)
  })

  it('ignores ordinary errors', () => {
    // A urltest timeout is noisy but not a startup failure; banner-ing it would
    // train the operator to ignore the banner.
    expect(isStartupFailure(ERROR)).toBe(false)
    expect(isStartupFailure(DEBUG)).toBe(false)
    expect(isStartupFailure(WARN)).toBe(false)
  })
})

/**
 * The panel's own log, added when the Logs page grew a second tab.
 *
 * zap's production encoder emits one JSON object per line with a LOWERCASE
 * level, which nothing in the sing-box path matches — so before this every line
 * of the panel's log, errors included, rendered as undifferentiated grey info.
 */
describe('structured (zap) lines', () => {
  it('reads the level from a JSON entry', () => {
    const line = '{"level":"error","timestamp":"2026-08-27T16:00:00.000+0800","caller":"app/svr.go:42","msg":"boom"}'
    expect(parseLogLine(line).level).toBe('error')
  })

  it('renders as a readable line, not as raw JSON', () => {
    const line = '{"level":"info","timestamp":"2026-08-27T16:00:00.000+0800","caller":"app/svr.go:42","msg":"listening"}'
    const { text } = parseLogLine(line)

    expect(text).toContain('listening')
    expect(text).toContain('INFO')
    expect(text).not.toContain('"level"')
  })

  it('keeps fields the operator chose to log', () => {
    const line = '{"level":"warn","msg":"slow","elapsed_ms":1200,"tag":"proxy"}'
    const { text } = parseLogLine(line)

    expect(text).toContain('elapsed_ms=1200')
    expect(text).toContain('tag=proxy')
  })

  it('maps panic levels onto fatal, which is the only one the UI paints red', () => {
    expect(parseLogLine('{"level":"dpanic","msg":"x"}').level).toBe('fatal')
    expect(parseLogLine('{"level":"panic","msg":"x"}').level).toBe('fatal')
  })

  // A sing-box line that merely begins with a brace must not be swallowed by
  // the JSON path — it has to fall through to the token scan.
  it('falls through when a brace-leading line is not JSON', () => {
    const line = '{not json} ERROR something failed'
    const parsed = parseLogLine(line)

    expect(parsed.level).toBe('error')
    expect(parsed.text).toBe(line)
  })

  it('falls through for JSON without a recognised level', () => {
    const line = '{"foo":"bar"}'
    expect(parseLogLine(line).text).toBe(line)
  })
})

describe('splitLogLine', () => {
  const BODY = '[5612797024 24ms] connection: open connection to 203.0.113.7:58886 using outbound/direct[直连]: connection refused'

  it('strips an OpenWrt logread prefix and sing-box\'s own header, keeping the local time', () => {
    const entry = splitLogLine(`Wed Oct  7 16:38:38 2026 daemon.err sing-box[22865]: +0000 2026-10-07 08:38:38 ERROR ${BODY}`)
    expect(entry).toEqual({ level: 'error', time: '16:38:38', message: BODY })
  })

  it('strips a journald prefix', () => {
    const entry = splitLogLine(`Oct 07 16:38:39 router sing-box[931]: +0800 2026-10-07 16:38:39 INFO inbound/tun[tun-in]: inbound connection from 192.168.1.20:51458`)
    expect(entry).toEqual({ level: 'info', time: '16:38:39', message: 'inbound/tun[tun-in]: inbound connection from 192.168.1.20:51458' })
  })

  it('reads a bare log.output line, using sing-box\'s time when it is the only one', () => {
    expect(splitLogLine(`+0000 2026-10-07 08:38:38 DEBUG router: match[12] => route(direct)`))
      .toEqual({ level: 'debug', time: '08:38:38', message: 'router: match[12] => route(direct)' })
  })

  it('handles timestamps being disabled in the sing-box config', () => {
    expect(splitLogLine('Wed Oct  7 16:38:38 2026 daemon.info sing-box[1]: WARN dns: exchange failed'))
      .toEqual({ level: 'warn', time: '16:38:38', message: 'dns: exchange failed' })
    expect(splitLogLine('INFO sing-box started')).toEqual({ level: 'info', time: '', message: 'sing-box started' })
  })

  it('strips ANSI colour before matching', () => {
    expect(splitLogLine('\u001b[31mERROR\u001b[0m [1 0ms] boom').level).toBe('error')
  })

  it('leaves an unrecognised line whole rather than guessing at a prefix', () => {
    const line = 'something happened: 12:00:00 and then more'
    expect(splitLogLine(line)).toEqual({ level: 'info', time: '', message: line })
  })

  it('keeps a multi-line message intact', () => {
    expect(splitLogLine('+0000 2026-10-07 08:38:38 FATAL start service: line one\nline two').message)
      .toBe('start service: line one\nline two')
  })

  it('maps PANIC onto fatal and a crash loop onto fatal', () => {
    expect(splitLogLine('PANIC runtime error').level).toBe('fatal')
    expect(splitLogLine('procd: sing-box is in a crash loop').level).toBe('fatal')
  })

  it('reads the panel\'s own JSON lines', () => {
    const entry = splitLogLine('{"level":"warn","timestamp":"2026-10-07T16:40:01.120+0800","caller":"apiv1/x.go:10","msg":"slow","elapsed_ms":1200}')
    expect(entry).toEqual({ level: 'warn', time: '16:40:01', message: 'apiv1/x.go:10 slow elapsed_ms=1200' })
  })
})

describe('filterLogEntries', () => {
  const entries = [
    { level: 'debug' as const, time: '', message: 'router: match[12]' },
    { level: 'info' as const, time: '', message: 'inbound connection from 192.168.1.2' },
    { level: 'error' as const, time: '', message: 'connection refused' },
    { level: 'fatal' as const, time: '', message: 'start service failed' },
  ]

  it('keeps entries at or above the minimum level, returning their indexes', () => {
    expect(filterLogEntries(entries, 'trace', '')).toEqual([0, 1, 2, 3])
    expect(filterLogEntries(entries, 'info', '')).toEqual([1, 2, 3])
    expect(filterLogEntries(entries, 'error', '')).toEqual([2, 3])
  })

  it('applies a case-insensitive regex', () => {
    expect(filterLogEntries(entries, 'trace', 'REFUSED|failed')).toEqual([2, 3])
  })

  it('falls back to a literal match for an unfinished regex', () => {
    expect(filterLogEntries(entries, 'trace', 'match[1')).toEqual([0])
  })
})
