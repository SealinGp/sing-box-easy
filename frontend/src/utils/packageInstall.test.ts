import { describe, expect, test } from 'bun:test'
import type { PackageInstallResult } from '../types/version'
import { packageInstallFailureText, packageInstallVerdict } from './packageInstall'

const result = (over: Partial<PackageInstallResult>): PackageInstallResult => ({
  state: 'succeeded',
  from_version: 'v1.5.3',
  to_version: 'v1.5.4',
  exit_code: 0,
  started_at: '',
  finished_at: '',
  log_tail: '',
  ...over,
})

describe('packageInstallVerdict', () => {
  test('the helper result decides when it names this run', () => {
    expect(packageInstallVerdict({
      status: { current_version: 'v1.5.3', last_package_install: result({ state: 'failed', exit_code: 255 }) },
      target: 'v1.5.4',
      from: 'v1.5.3',
      sawOutage: true,
    })).toBe('failed')
    expect(packageInstallVerdict({
      status: { current_version: 'v1.5.4', last_package_install: result({}) },
      target: '1.5.4',
      from: 'v1.5.3',
      sawOutage: false,
    })).toBe('succeeded')
  })

  test('a helper that is still running is pending, whatever the version says', () => {
    expect(packageInstallVerdict({
      status: { current_version: 'v1.5.4', last_package_install: result({ state: 'running', exit_code: -1 }) },
      target: 'v1.5.4',
      from: 'v1.5.3',
      sawOutage: true,
    })).toBe('pending')
  })

  test('an interrupted helper is a failure, not an endless wait', () => {
    expect(packageInstallVerdict({
      status: { current_version: 'v1.5.3', last_package_install: result({ state: 'interrupted', exit_code: -1 }) },
      target: 'v1.5.4',
      from: 'v1.5.3',
      sawOutage: true,
    })).toBe('failed')
  })

  test('a result from an earlier install does not answer for this one', () => {
    expect(packageInstallVerdict({
      status: { current_version: 'v1.5.3', last_package_install: result({ to_version: 'v1.5.2' }) },
      target: 'v1.5.4',
      from: 'v1.5.3',
      sawOutage: false,
    })).toBe('pending')
  })

  test('a target that predates the feature is recognised by its version', () => {
    expect(packageInstallVerdict({
      status: { current_version: 'v1.5.4' },
      target: 'v1.5.4',
      from: 'v1.5.3',
      sawOutage: true,
    })).toBe('succeeded')
  })

  test('a version change is enough, even when the outage was never observed', () => {
    // The panel is down for ~2s and the probe runs every 2s, so the outage is
    // routinely missed. The version having changed is proof on its own.
    expect(packageInstallVerdict({
      status: { current_version: 'v1.5.4' },
      target: 'v1.5.4',
      from: 'v1.5.5',
      sawOutage: false,
    })).toBe('succeeded')
  })

  test('a reinstall of the running version needs the outage', () => {
    // The old panel answers with the target version before the helper has
    // done anything.
    expect(packageInstallVerdict({
      status: { current_version: 'v1.5.4' },
      target: 'v1.5.4',
      from: '1.5.4',
      sawOutage: false,
    })).toBe('pending')
    expect(packageInstallVerdict({
      status: { current_version: 'v1.5.4' },
      target: 'v1.5.4',
      from: '1.5.4',
      sawOutage: true,
    })).toBe('succeeded')
  })

  test('the old panel coming back without a result is still pending', () => {
    expect(packageInstallVerdict({
      status: { current_version: 'v1.5.3' },
      target: 'v1.5.4',
      from: 'v1.5.3',
      sawOutage: true,
    })).toBe('pending')
  })
})

describe('packageInstallFailureText', () => {
  test('carries opkg\'s own output', () => {
    expect(packageInstallFailureText(result({ state: 'failed', exit_code: 255, log_tail: 'Collected errors:\n * no space' })))
      .toBe('opkg exited with 255.\nCollected errors:\n * no space')
  })
  test('says so when the helper never finished', () => {
    expect(packageInstallFailureText(result({ state: 'interrupted', exit_code: -1 })))
      .toBe('The install did not finish.')
  })
  test('is empty without a result', () => {
    expect(packageInstallFailureText(null)).toBe('')
  })
})
