import { ApiError } from '../types/api'

/**
 * The most informative text available for a thrown API/transport failure.
 *
 * The shared axios interceptor (services/api.ts) throws an ApiError carrying
 * the backend's `msg`, which names the actual problem ("subscription server
 * returned http 404"); the caller's fallback is a generic category. Prefer the
 * former, and fall back only when it is missing or blank.
 *
 * Extracted so a toast and an inline message describe the same failure the same
 * way — a message that changes wording depending on where it is rendered makes
 * a bug report harder to match to a log line.
 */
export function apiErrorMessage(err: unknown, fallback: string): string {
  if ((err instanceof ApiError || err instanceof Error) && err.message.trim() !== '') {
    return err.message
  }
  return fallback
}
