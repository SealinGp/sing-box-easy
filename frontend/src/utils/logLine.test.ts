import { describe, expect, it } from 'bun:test'
import { parseLogLine, stripAnsi, isStartupFailure } from './logLine'

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
