import type { DiffLine, LineType } from './types'
import { valTok, keyTok, comma, br, appendComma, isCont, isObj, escPtr } from './tokens'

/** Render a value (primitive or container) as diff lines with a fixed line type. */
export function renderBlock(
  value: unknown,
  indent: number,
  type: LineType,
  path: string,
  lines: DiffLine[]
): void {
  if (isCont(value)) {
    renderContainerBlock(value, indent, type, path, lines)
  } else {
    lines.push({ tokens: valTok(value), type, indent, path })
  }
}

/** Recursively render a container (object/array) with proper tokens and a fixed line type. */
function renderContainerBlock(
  value: unknown,
  indent: number,
  type: LineType,
  path: string,
  lines: DiffLine[]
): void {
  if (isObj(value)) {
    const keys = Object.keys(value)
    lines.push({ tokens: [br('{')], type, indent, path })
    keys.forEach((key, i) => {
      const cp = `${path}/${escPtr(key)}`
      const last = i === keys.length - 1
      if (isCont(value[key])) {
        lines.push({ tokens: keyTok(key), type, indent: indent + 1, path: cp })
        renderContainerBlock(value[key], indent + 1, type, cp, lines)
        if (!last) appendComma(lines)
      } else {
        lines.push({
          tokens: [...keyTok(key), ...valTok(value[key]), ...(last ? [] : [comma()])],
          type,
          indent: indent + 1,
          path: cp,
        })
      }
    })
    lines.push({ tokens: [br('}')], type, indent, path })
  } else if (Array.isArray(value)) {
    lines.push({ tokens: [br('[')], type, indent, path })
    value.forEach((item, i) => {
      const cp = `${path}/${i}`
      const last = i === value.length - 1
      if (isCont(item)) {
        renderContainerBlock(item, indent + 1, type, cp, lines)
        if (!last) appendComma(lines)
      } else {
        lines.push({
          tokens: [...valTok(item), ...(last ? [] : [comma()])],
          type,
          indent: indent + 1,
          path: cp,
        })
      }
    })
    lines.push({ tokens: [br(']')], type, indent, path })
  }
}

/**
 * Recursively render a value as 'normal' lines with proper syntax tokens.
 * Used for lazy-expanding collapsed nodes.
 */
export function renderNormal(
  value: unknown,
  indent: number,
  path: string,
  lines: DiffLine[]
): void {
  if (isObj(value)) {
    const keys = Object.keys(value)
    lines.push({ tokens: [br('{')], type: 'normal', indent, path })
    keys.forEach((key, i) => {
      const cp = `${path}/${escPtr(key)}`
      const last = i === keys.length - 1
      if (isCont(value[key])) {
        lines.push({ tokens: keyTok(key), type: 'normal', indent: indent + 1, path: cp })
        renderNormal(value[key], indent + 1, cp, lines)
        if (!last) appendComma(lines)
      } else {
        lines.push({
          tokens: [...keyTok(key), ...valTok(value[key]), ...(last ? [] : [comma()])],
          type: 'normal',
          indent: indent + 1,
          path: cp,
        })
      }
    })
    lines.push({ tokens: [br('}')], type: 'normal', indent, path })
  } else if (Array.isArray(value)) {
    lines.push({ tokens: [br('[')], type: 'normal', indent, path })
    value.forEach((item, i) => {
      const cp = `${path}/${i}`
      const last = i === value.length - 1
      if (isCont(item)) {
        renderNormal(item, indent + 1, cp, lines)
        if (!last) appendComma(lines)
      } else {
        lines.push({
          tokens: [...valTok(item), ...(last ? [] : [comma()])],
          type: 'normal',
          indent: indent + 1,
          path: cp,
        })
      }
    })
    lines.push({ tokens: [br(']')], type: 'normal', indent, path })
  } else {
    lines.push({ tokens: valTok(value), type: 'normal', indent, path })
  }
}

/** Count how many lines a value would produce when fully rendered. */
export function countLines(value: unknown): number {
  if (!isCont(value)) return 1
  const tmp: DiffLine[] = []
  renderNormal(value, 0, '', tmp)
  return tmp.length
}
