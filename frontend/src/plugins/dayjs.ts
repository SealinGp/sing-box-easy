import type { App } from 'vue'
import dayjs from 'dayjs'
import duration from 'dayjs/plugin/duration'
import relativeTime from 'dayjs/plugin/relativeTime'
import customParseFormat from 'dayjs/plugin/customParseFormat'
// Register the Chinese locale so the i18n layer can switch dayjs to it.
import 'dayjs/locale/zh-cn'

// Initialize dayjs with plugins
export function loadDayjs(app: App<Element>) {
  // Extend dayjs with plugins
  dayjs.extend(duration)
  dayjs.extend(relativeTime)
  dayjs.extend(customParseFormat)

  // Make dayjs available globally in Vue components
  app.config.globalProperties.$dayjs = dayjs

  // Optional: Also make it available as a provide/inject
  app.provide('dayjs', dayjs)
}

// Parse duration string like "24h", "30d", "60min", "2w" to hours
export const parseDurationToHours = (durationStr: string | undefined): number | null => {
  if (!durationStr) return null

  const patterns = [
    { regex: /^(\d+)\s*h(our)?s?$/i, unit: 'hours', factor: 1 },
    { regex: /^(\d+)\s*d(ay)?s?$/i, unit: 'days', factor: 24 },
    { regex: /^(\d+)\s*w(eek)?s?$/i, unit: 'weeks', factor: 168 },
    { regex: /^(\d+)\s*m(in)?(ute)?s?$/i, unit: 'minutes', factor: 1 / 60 },
    { regex: /^(\d+)\s*s(ec)?(ond)?s?$/i, unit: 'seconds', factor: 1 / 3600 },
    { regex: /^(\d+)\s*mo(nth)?s?$/i, unit: 'months', factor: 720 }, // Approximate: 30 days
  ]

  for (const pattern of patterns) {
    const match = durationStr.match(pattern.regex)
    if (match && match[1]) {
      const value = parseInt(match[1])
      return Math.round(value * pattern.factor)
    }
  }

  // Try to parse as just a number (assume hours)
  const numMatch = durationStr.match(/^(\d+)$/)
  if (numMatch && numMatch[1]) {
    return parseInt(numMatch[1])
  }

  return null
}

// Validate if a string is a valid duration
export const isValidDuration = (durationStr: string | undefined): boolean => {
  return parseDurationToHours(durationStr) !== null
}

// Format hours to a readable duration string
export const formatHoursToDuration = (hours: number): string => {
  if (hours < 1) return `${Math.round(hours * 60)}min`
  if (hours < 24) return `${hours}h`
  if (hours < 168) return `${Math.round(hours / 24)}d`
  if (hours < 720) return `${Math.round(hours / 168)}w`
  return `${Math.round(hours / 720)}mo`
}

export default dayjs