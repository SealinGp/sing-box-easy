import { describe, expect, test } from 'bun:test'
import {
  BUG_FIELDS, BUG_TEMPLATE, PROJECT_URL, bugReportUrl, diagnosticsText, environmentLines, type Diagnostics,
} from './bugReport'

// The environment of issue #14, which its reporter typed out by hand.
const full: Diagnostics = {
  panelVersion: '1.5.3',
  coreVersion: '1.12.25',
  distribution: 'ImmortalWrt 24.10',
  systemType: 'openwrt',
  os: 'linux',
  arch: 'arm64',
  cpuCores: 4,
  kernel: '6.6.86',
  serviceBackend: 'procd',
  installMethod: 'opkg',
  authEnabled: false,
  userAgent: 'Mozilla/5.0 (Macintosh) Chrome/141.0',
}

const paramsOf = (url: string) => new URL(url).searchParams

describe('the contract with the issue form', () => {
  test('every field this module pre-fills exists in bug_report.yml', async () => {
    const template = await Bun.file(new URL(`../../../.github/ISSUE_TEMPLATE/${BUG_TEMPLATE}`, import.meta.url)).text()
    const ids = [...template.matchAll(/^\s*id:\s*(\S+)\s*$/gm)].map((match) => match[1])
    for (const field of Object.values(BUG_FIELDS)) {
      expect(ids).toContain(field)
    }
  })
})

describe('bugReportUrl', () => {
  test('opens the bug form on this repository with every known field', () => {
    const url = bugReportUrl(full)
    expect(url.startsWith(`${PROJECT_URL}/issues/new?`)).toBe(true)
    const params = paramsOf(url)
    expect(params.get('template')).toBe('bug_report.yml')
    expect(params.get('panel_version')).toBe('1.5.3')
    expect(params.get('core_version')).toBe('1.12.25')
    expect(params.get('browser')).toBe('Mozilla/5.0 (Macintosh) Chrome/141.0')
    expect(params.get('environment')).toBe([
      'System: ImmortalWrt 24.10 (openwrt)',
      'Platform: linux/arm64, 4 cores',
      'Kernel: 6.6.86',
      'Service manager: procd',
      'Install method: opkg',
      'Login required: no',
    ].join('\n'))
  })

  test('unknown values are left out so the form keeps its placeholders', () => {
    const params = paramsOf(bugReportUrl({}))
    expect([...params.keys()]).toEqual(['template'])
  })

  test('non-ASCII values survive the round trip', () => {
    const params = paramsOf(bugReportUrl({ distribution: '自编译固件 2026' }))
    expect(params.get('environment')).toBe('System: 自编译固件 2026')
  })

  test('nothing identifying is sent even if the caller has it', () => {
    const withHost = { ...full, hostname: 'toms-router.lan' } as Diagnostics
    expect(bugReportUrl(withHost)).not.toContain('toms-router')
    expect(diagnosticsText(withHost)).not.toContain('toms-router')
  })

  test('an oversized value is clipped rather than producing a URL GitHub rejects', () => {
    const params = paramsOf(bugReportUrl({ userAgent: 'x'.repeat(5000), kernel: 'k'.repeat(5000) }))
    expect(params.get('browser')!.length).toBe(300)
    expect(bugReportUrl({ userAgent: 'x'.repeat(5000), kernel: 'k'.repeat(5000) }).length).toBeLessThan(2000)
  })

  test('a value containing newlines cannot inject extra lines into the block', () => {
    const lines = environmentLines({ kernel: '6.6\nLogin required: yes' })
    expect(lines).toEqual(['Kernel: 6.6 Login required: yes'])
  })
})

describe('environmentLines', () => {
  test('an "unknown" system family says nothing rather than "unknown"', () => {
    expect(environmentLines({ systemType: 'unknown' })).toEqual([])
    expect(environmentLines({ distribution: 'Debian 12', systemType: 'unknown' })).toEqual(['System: Debian 12'])
    expect(environmentLines({ systemType: 'openwrt' })).toEqual(['System: openwrt'])
  })
  test('platform is assembled from whatever parts are known', () => {
    expect(environmentLines({ arch: 'amd64' })).toEqual(['Platform: amd64'])
    expect(environmentLines({ os: 'linux', arch: 'amd64', cpuCores: 0 })).toEqual(['Platform: linux/amd64'])
  })
  test('login state is reported only when known', () => {
    expect(environmentLines({ authEnabled: true })).toEqual(['Login required: yes'])
    expect(environmentLines({})).toEqual([])
  })
})

describe('diagnosticsText', () => {
  test('leads with the two versions, then the environment, then the browser', () => {
    expect(diagnosticsText(full).split('\n')).toEqual([
      'sing-box-easy: 1.5.3',
      'sing-box: 1.12.25',
      'System: ImmortalWrt 24.10 (openwrt)',
      'Platform: linux/arm64, 4 cores',
      'Kernel: 6.6.86',
      'Service manager: procd',
      'Install method: opkg',
      'Login required: no',
      'Browser: Mozilla/5.0 (Macintosh) Chrome/141.0',
    ])
  })
  test('nothing known is an empty string, not a column of labels', () => {
    expect(diagnosticsText({})).toBe('')
  })
})
