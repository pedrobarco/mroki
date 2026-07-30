// Client-side mirror of Go's time.ParseDuration, used to validate per-gate
// retention against the server's global floor before the API round-trip. The
// server remains the source of truth; this only fails obviously-invalid input
// fast. Supported units match Go exactly: ns, us (µs), ms, s, m, h. Go itself
// does NOT support "d" (days) or "w" (weeks); the Hub accepts those as a
// convenience via normalizeToGoDuration, which expands them into hours before
// the value is validated or sent to the API.

const UNIT_NS: Record<string, number> = {
  ns: 1,
  us: 1e3,

  µs: 1e3,
  ms: 1e6,
  s: 1e9,
  m: 60 * 1e9,
  h: 3600 * 1e9,
}

// A signed decimal number followed by a required unit, one or more times, e.g.
// "168h", "1h30m", "300ms". Mirrors Go's grammar (leading sign allowed once).
// Go accepts a trailing decimal point with no fraction (e.g. "1.h"), so the
// integer/fraction parts are individually optional as long as at least one digit
// is present overall.
const NUM_UNIT = /^(\d+\.\d*|\d*\.\d+|\d+)(ns|us|µs|ms|s|m|h)/

// Go stores durations as int64 nanoseconds; anything at or past this overflows
// and Go rejects it. We mirror that so client validation matches the server.
// int64's max (2^63 - 1) is not exactly representable as a JS double, so we use
// 2^63 as the boundary — the smallest double strictly greater than max int64.
const MAX_INT64_NS = 2 ** 63

/**
 * Parse a Go duration string (e.g. "168h", "1h30m") into a total number of
 * nanoseconds. Returns null when the string is not a valid Go duration.
 *
 * "0" (and "0s", etc.) parse successfully to 0; callers decide whether zero is
 * acceptable.
 */
export function parseGoDuration(input: string): number | null {
  let s = input.trim()
  if (s === '') return null

  // Optional leading sign.
  let sign = 1
  if (s[0] === '+' || s[0] === '-') {
    if (s[0] === '-') sign = -1
    s = s.slice(1)
  }

  // Special case: a bare "0" with no unit is valid in Go.
  if (s === '0') return 0

  if (s === '') return null

  let total = 0
  let matchedAny = false
  while (s !== '') {
    const m = NUM_UNIT.exec(s)
    if (!m) return null
    const value = Number(m[1])
    const unitNs = UNIT_NS[m[2] as string]
    if (!Number.isFinite(value) || unitNs === undefined) return null
    total += value * unitNs
    // Reject values Go cannot store (int64 nanosecond overflow).
    if (total >= MAX_INT64_NS) return null
    s = s.slice(m[0].length)
    matchedAny = true
  }

  return matchedAny ? sign * total : null
}

// Human-friendly units Go's parser rejects. Days and weeks are convenience
// shorthands the Hub accepts and expands into hours (Go's largest unit) before
// sending to the API, so users can type "30d" or "2w" instead of "720h".
const HOURS_PER: Record<string, number> = {
  d: 24,
  w: 168,
}

// A signed decimal number followed by a day/week shorthand, one or more times,
// e.g. "30d", "2w", "1w3d". Same grammar as NUM_UNIT but for d/w only.
const NUM_DW = /^(\d+(?:\.\d+)?|\.\d+)(d|w)/

/**
 * Normalize a human duration string into a valid Go duration by expanding any
 * day ("d") and week ("w") shorthands into hours. Segments already using Go
 * units (h, m, s, …) are preserved verbatim. Returns null when the input mixes
 * unknown units or is otherwise unparseable.
 *
 * Examples: "30d" -> "720h", "2w" -> "336h", "1w3d12h" -> "168h72h12h",
 * "168h" -> "168h" (unchanged). The output is not minimized — it is only
 * guaranteed to be a string Go's ParseDuration accepts.
 */
export function normalizeToGoDuration(input: string): string | null {
  let s = input.trim()
  if (s === '') return null

  let prefix = ''
  if (s[0] === '+' || s[0] === '-') {
    prefix = s[0]
    s = s.slice(1)
  }

  if (s === '0') return input.trim()
  if (s === '') return null

  let out = ''
  while (s !== '') {
    const dw = NUM_DW.exec(s)
    if (dw) {
      const value = Number(dw[1])
      const hours = value * (HOURS_PER[dw[2] as string] as number)
      if (!Number.isFinite(hours)) return null
      out += `${hours}h`
      s = s.slice(dw[0].length)
      continue
    }
    const m = NUM_UNIT.exec(s)
    if (!m) return null
    out += m[0]
    s = s.slice(m[0].length)
  }

  const candidate = prefix + out
  // Final guard: the assembled string must be a real Go duration.
  return parseGoDuration(candidate) === null ? null : candidate
}

const NS_PER_HOUR = 3600 * 1e9

/**
 * Render a Go duration string in a friendlier, human-scaled form for display.
 * The raw value is machine-formatted (e.g. Go's "720h0m0s"), which is noisy for
 * round numbers of days. When the duration is a whole number of days (a
 * multiple of 24h) it is shown as "<n>d (<raw>)"; otherwise the trimmed raw
 * value is returned unchanged. Falls back to the input when it cannot be parsed.
 */
export function humanizeGoDuration(input: string): string {
  const raw = input.trim()
  const ns = parseGoDuration(raw)
  if (ns === null || ns <= 0) return raw

  const hours = ns / NS_PER_HOUR
  if (Number.isInteger(hours) && hours % 24 === 0) {
    const days = hours / 24
    return `${days}d (${raw})`
  }
  return raw
}
