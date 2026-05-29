// Composable for reading/setting the active locale from components.
import { computed } from 'vue'
import { i18n, applyLocale, SUPPORTED_LOCALES, type Locale } from './index'

// Native display names — each language shown in its own script.
const LABELS: Record<Locale, string> = {
  zh: '中文',
  en: 'English',
}

// Short badge labels for the compact switcher.
const SHORT_LABELS: Record<Locale, string> = {
  zh: '中',
  en: 'EN',
}

export function setLocale(locale: Locale) {
  applyLocale(locale, true)
}

export function useLocale() {
  const locale = computed<Locale>({
    get: () => i18n.global.locale.value as Locale,
    set: (l) => setLocale(l),
  })

  const availableLocales = SUPPORTED_LOCALES.map((code) => ({
    code,
    label: LABELS[code],
  }))

  const shortLabel = computed(() => SHORT_LABELS[locale.value])

  const toggle = () => setLocale(locale.value === 'zh' ? 'en' : 'zh')

  return { locale, setLocale, toggle, availableLocales, shortLabel }
}
