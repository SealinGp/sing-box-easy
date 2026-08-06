import zashboardIcon from '../assets/zashboard.svg'
import yacdIcon from '../assets/yacd.ico'

/**
 * Known Clash-compatible web dashboards that sing-box can download into its
 * `external_ui` directory.
 *
 * Shared by the init wizard (ConfigureExperimental.vue, card grid) and the
 * Clash API settings page (ClashAPISettings.vue, dropdown) so both offer the
 * same list and the same URLs.
 *
 * `descKey` is an i18n key; `name`/`url` are technical values and stay literal.
 */
export interface DashboardOption {
  id: string
  name: string
  /** Release archive sing-box downloads and extracts. */
  url: string
  /** Project homepage, shown as a link. */
  link: string
  /** Bundled icon asset. */
  icon: string
  /** Screenshot URL, used by the wizard's preview grid. */
  preview: string
  descKey: string
}

export const DASHBOARD_OPTIONS: readonly DashboardOption[] = [
  {
    id: 'zashboard',
    name: 'Zashboard',
    url: 'https://github.com/Zephyruso/zashboard/archive/gh-pages.zip',
    link: 'https://github.com/Zephyruso/zashboard',
    icon: zashboardIcon,
    preview: 'https://raw.githubusercontent.com/Zephyruso/zashboard/refs/heads/main/readme/pc.png',
    descKey: 'init.experimental.dashboards.zashboard',
  },
  {
    id: 'yacd',
    name: 'Yacd',
    url: 'https://github.com/MetaCubeX/Yacd-meta/archive/gh-pages.zip',
    link: 'https://github.com/haishanh/yacd',
    icon: yacdIcon,
    preview:
      'https://user-images.githubusercontent.com/1166872/47954055-97e6cb80-dfc0-11e8-991f-230fd40481e5.png',
    descKey: 'init.experimental.dashboards.yacd',
  },
] as const

/** Sentinel id for "not one of the presets — the user typed their own URL". */
export const CUSTOM_DASHBOARD_ID = 'custom'

/**
 * Resolve a stored download URL back to the preset that produced it.
 * Returns null for an empty URL, or CUSTOM_DASHBOARD_ID when it matches no preset.
 */
export function dashboardIdForUrl(url: string | undefined | null): string | null {
  const trimmed = url?.trim()
  if (!trimmed) return null
  return DASHBOARD_OPTIONS.find((d) => d.url === trimmed)?.id ?? CUSTOM_DASHBOARD_ID
}
