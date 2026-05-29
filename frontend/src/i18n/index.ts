// vue-i18n setup: locale detection, persistence, and the app plugin loader.
//
// Locale is a device/UI preference, so it lives in localStorage (no backend).
// On first load we auto-detect from navigator.language; anything that isn't
// clearly English falls back to Chinese (this project's primary audience).
import { createI18n } from 'vue-i18n'
import type { App } from 'vue'
import dayjs from 'dayjs'
import en from './locales/en'
import zh from './locales/zh'

export const SUPPORTED_LOCALES = ['zh', 'en'] as const
export type Locale = (typeof SUPPORTED_LOCALES)[number]

export const STORAGE_KEY = 'sbe-locale'
export const DEFAULT_LOCALE: Locale = 'zh'

// Normalize a raw language tag (e.g. "zh-CN", "en_US") to a supported locale,
// or null when it maps to neither.
export function normalizeLocale(raw: string | null | undefined): Locale | null {
  if (!raw) return null
  const lower = raw.toLowerCase()
  if (lower.startsWith('zh')) return 'zh'
  if (lower.startsWith('en')) return 'en'
  return null
}

// Resolve the initial locale: saved choice → browser language → fallback.
export function detectLocale(): Locale {
  try {
    const saved = normalizeLocale(localStorage.getItem(STORAGE_KEY))
    if (saved) return saved
  } catch {
    // localStorage may be unavailable (private mode / SSR) — ignore.
  }
  const fromNav = normalizeLocale(
    typeof navigator !== 'undefined' ? navigator.language : null,
  )
  return fromNav ?? DEFAULT_LOCALE
}

export const i18n = createI18n({
  legacy: false,
  globalInjection: true,
  locale: detectLocale(),
  fallbackLocale: DEFAULT_LOCALE,
  messages: { en, zh },
})

// Apply a locale across vue-i18n, the document, and dayjs. `persist` controls
// whether the choice is written to localStorage (true for explicit user
// actions, false for the initial auto-detected apply).
export function applyLocale(locale: Locale, persist: boolean) {
  i18n.global.locale.value = locale
  if (persist) {
    try {
      localStorage.setItem(STORAGE_KEY, locale)
    } catch {
      // ignore write failures
    }
  }
  try {
    document.documentElement.lang = locale
  } catch {
    // ignore (non-DOM env)
  }
  dayjs.locale(locale === 'zh' ? 'zh-cn' : 'en')
}

export function loadI18n(app: App<Element>) {
  app.use(i18n)
  // Sync <html lang> + dayjs with the auto-detected locale without persisting.
  applyLocale(i18n.global.locale.value as Locale, false)
}
