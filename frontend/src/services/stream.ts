/**
 * SSE client.
 *
 * WHY NOT `EventSource`
 * ────────────────────
 * It cannot set request headers, and this API authenticates with
 * `Authorization: Bearer <token>` read from localStorage (see api.ts). The usual
 * workaround — putting the token in the query string — writes a live session
 * credential into every access log, proxy log and Referer header between the
 * browser and the panel. `fetch()` sends headers and exposes the body as a
 * stream, so it costs one small parser and keeps the credential in a header
 * where it belongs.
 *
 * The parser is small because the server writes a deliberately narrow subset of
 * SSE (sse.go): `event:` + one or more `data:` lines + a blank line, plus `:`
 * comment frames for keep-alive. Multi-line `data:` is the one part that must
 * not be simplified away — a JSON payload containing a newline is split across
 * several `data:` lines and has to be rejoined before parsing.
 */
import { ApiError, Code, type BasicResponse } from '../types/api'

const BASE_URL = '/api/1.12.12'

export interface StreamHandlers {
  /** Called per event. `name` is the server's `event:` field. */
  onEvent: (name: string, data: unknown) => void
  /** Transport or protocol failure. Not called on a clean close. */
  onError?: (error: Error) => void
  /** The stream ended — cleanly or otherwise. Always called exactly once. */
  onClose?: () => void
}

export interface StreamOptions extends StreamHandlers {
  /** POST body. Omit for a GET stream. */
  body?: unknown
  /** Aborts the request; ALWAYS wire this to component teardown. */
  signal: AbortSignal
}

/**
 * Opens an SSE stream.
 *
 * Resolves when the stream closes. The caller aborts via `options.signal` —
 * and must, because the server holds a child process open for the life of a log
 * follow. An abandoned stream is a `journalctl -f` that never exits.
 */
export async function openStream(path: string, options: StreamOptions): Promise<void> {
  const { body, signal, onEvent, onError, onClose } = options

  try {
    const token = localStorage.getItem('sb_token')
    const headers: Record<string, string> = { Accept: 'text/event-stream' }
    if (token) headers['Authorization'] = `Bearer ${token}`
    if (body !== undefined) headers['Content-Type'] = 'application/json'

    const response = await fetch(`${BASE_URL}${path}`, {
      method: body === undefined ? 'GET' : 'POST',
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
      signal,
    })

    if (!response.ok || !response.body) {
      throw new Error(`stream failed: HTTP ${response.status}`)
    }

    const reader = response.body.pipeThrough(new TextDecoderStream()).getReader()
    // Frames are separated by a blank line and can be split across reads, so a
    // partial tail is carried into the next chunk rather than parsed.
    let buffer = ''

    for (;;) {
      const { done, value } = await reader.read()
      if (done) break
      buffer += value

      let boundary = buffer.indexOf('\n\n')
      while (boundary !== -1) {
        const frame = buffer.slice(0, boundary)
        buffer = buffer.slice(boundary + 2)
        dispatchFrame(frame, onEvent)
        boundary = buffer.indexOf('\n\n')
      }
    }
  } catch (error) {
    // An abort is the caller closing the stream on purpose, not a failure.
    if (signal.aborted) return
    onError?.(error instanceof Error ? error : new Error(String(error)))
  } finally {
    onClose?.()
  }
}

/**
 * Parses one frame and hands it to the caller.
 *
 * A frame whose envelope reports failure throws, which unwinds into
 * `openStream`'s catch and reaches the caller as `onError(ApiError)` — so a
 * mid-stream failure carries the same `code` a unary call would have, which is
 * the whole reason the server wraps each event in the standard envelope rather
 * than sending bare payloads.
 */
function dispatchFrame(frame: string, onEvent: StreamHandlers['onEvent']): void {
  let name = 'message'
  const dataLines: string[] = []

  for (const line of frame.split('\n')) {
    // Comment frame — the keep-alive. Carries nothing by design.
    if (line.startsWith(':')) continue
    if (line.startsWith('event:')) {
      name = line.slice(6).trim()
      continue
    }
    if (line.startsWith('data:')) {
      // Exactly one leading space is the separator, per the SSE spec; anything
      // beyond it is payload and must survive.
      dataLines.push(line.slice(5).replace(/^ /, ''))
    }
  }

  if (dataLines.length === 0) return

  let envelope: BasicResponse<unknown>
  try {
    envelope = JSON.parse(dataLines.join('\n'))
  } catch {
    // A frame we cannot parse is not worth tearing the stream down for — the
    // next one is very likely fine, and a log viewer that dies on one bad line
    // is worse than one that drops it.
    return
  }

  if (envelope.code !== Code.Success) {
    throw new ApiError(envelope.code, envelope.msg || 'stream error')
  }
  onEvent(name, envelope.data)
}
