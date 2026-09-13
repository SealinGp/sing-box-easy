export interface ClipboardWriters {
  modern?: (text: string) => Promise<void>
  legacy: (text: string) => boolean
}

function legacyCopy(text: string): boolean {
  if (typeof document === 'undefined') return false

  const previouslyFocused = document.activeElement
  const textarea = document.createElement('textarea')
  textarea.value = text
  textarea.readOnly = true
  textarea.setAttribute('aria-hidden', 'true')
  textarea.style.cssText = 'position:fixed;top:0;left:0;opacity:0;pointer-events:none'
  document.body.appendChild(textarea)

  try {
    textarea.focus()
    textarea.select()
    textarea.setSelectionRange(0, text.length)
    return document.execCommand('copy')
  } finally {
    document.body.removeChild(textarea)
    if (previouslyFocused instanceof HTMLElement) {
      previouslyFocused.focus({ preventScroll: true })
    }
  }
}

function browserClipboardWriters(): ClipboardWriters {
  const clipboard = typeof navigator === 'undefined' ? undefined : navigator.clipboard

  return {
    modern: clipboard?.writeText
      ? (text) => clipboard.writeText(text)
      : undefined,
    legacy: legacyCopy,
  }
}

/**
 * Copy text in secure and insecure browser contexts.
 *
 * The Clipboard API may be absent on LAN HTTP pages, or present but reject
 * because of browser permissions. Both cases fall back to the synchronous
 * copy command while the click still provides the required user gesture.
 */
export async function writeTextToClipboard(
  text: string,
  writers: ClipboardWriters = browserClipboardWriters(),
): Promise<void> {
  let modernError: unknown

  if (writers.modern) {
    try {
      await writers.modern(text)
      return
    } catch (error) {
      modernError = error
    }
  }

  try {
    if (writers.legacy(text)) return
  } catch (error) {
    throw error
  }

  if (modernError instanceof Error) throw modernError
  throw new Error('Clipboard access is unavailable')
}
