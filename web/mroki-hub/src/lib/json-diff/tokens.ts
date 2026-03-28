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
