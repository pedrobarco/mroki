import type { Token, DiffLine } from './types'

export function valTok(v: unknown): Token[] {
  if (v === null) return [{ text: 'null', type: 'null' }]
  if (v === undefined) return [{ text: 'undefined', type: 'null' }]
  if (typeof v === 'string') return [{ text: `"${v}"`, type: 'string' }]
  if (typeof v === 'number') return [{ text: String(v), type: 'number' }]
  if (typeof v === 'boolean') return [{ text: String(v), type: 'boolean' }]
  return [{ text: JSON.stringify(v), type: 'string' }]
}

export function keyTok(k: string): Token[] {
  return [
    { text: `"${k}"`, type: 'key' },
    { text: ': ', type: 'punctuation' },
  ]
}

export function comma(): Token {
  return { text: ',', type: 'punctuation' }
}

export function br(t: string): Token {
  return { text: t, type: 'bracket' }
}

export function appendComma(lines: DiffLine[]): void {
  const last = lines[lines.length - 1]
  if (last) last.tokens = [...last.tokens, comma()]
}

/** Produce a short preview token list like `{ 3 fields }` or `[ 5 items ]` for collapsed containers. */
export function collapsedPreview(v: unknown): Token[] {
  if (isObj(v)) {
    const n = Object.keys(v).length
    return [br('{'), { text: ` ${n} field${n !== 1 ? 's' : ''} `, type: 'punctuation' }, br('}')]
  }
  if (Array.isArray(v)) {
    const n = v.length
    return [br('['), { text: ` ${n} item${n !== 1 ? 's' : ''} `, type: 'punctuation' }, br(']')]
  }
  return valTok(v)
}

// --- Structural helpers ---

export function isCont(v: unknown): boolean {
  return isObj(v) || Array.isArray(v)
}

export function isObj(v: unknown): v is Record<string, unknown> {
  return typeof v === 'object' && v !== null && !Array.isArray(v)
}

export function escPtr(t: string): string {
  return t.replace(/~/g, '~0').replace(/\//g, '~1')
}

/**
 * Tokenize an arbitrary JSON value into syntax-highlighted tokens.
 * When pretty is true the output is multi-line (2-space indented) with newline
 * tokens, suitable for rendering inside a <pre>; otherwise it is a single
 * compact line. Token types mirror the colours used by the diff viewer.
 */
export function tokenizeJson(v: unknown, pretty = false, indent = 0): Token[] {
  if (v === null) return [{ text: 'null', type: 'null' }]
  if (v === undefined) return [{ text: 'undefined', type: 'null' }]
  if (typeof v === 'string') return [{ text: JSON.stringify(v), type: 'string' }]
  if (typeof v === 'number') return [{ text: String(v), type: 'number' }]
  if (typeof v === 'boolean') return [{ text: String(v), type: 'boolean' }]

  const nl = (lvl: number): Token[] =>
    pretty ? [{ text: '\n' + '  '.repeat(lvl), type: 'punctuation' }] : []
  const sep: Token = { text: pretty ? ',' : ', ', type: 'punctuation' }

  if (Array.isArray(v)) {
    if (v.length === 0) return [br('['), br(']')]
    const toks: Token[] = [br('[')]
    v.forEach((item, i) => {
      toks.push(...nl(indent + 1), ...tokenizeJson(item, pretty, indent + 1))
      if (i < v.length - 1) toks.push(sep)
    })
    toks.push(...nl(indent), br(']'))
    return toks
  }
  if (isObj(v)) {
    const entries = Object.entries(v)
    if (entries.length === 0) return [br('{'), br('}')]
    const toks: Token[] = [br('{')]
    entries.forEach(([k, val], i) => {
      toks.push(...nl(indent + 1), ...keyTok(k), ...tokenizeJson(val, pretty, indent + 1))
      if (i < entries.length - 1) toks.push(sep)
    })
    toks.push(...nl(indent), br('}'))
    return toks
  }
  return [{ text: String(v), type: 'string' }]
}
