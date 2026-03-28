import type { PatchOp } from '@/api'
import type { DiffLine } from './types'
import {
  valTok,
  keyTok,
  comma,
  br,
  appendComma,
  collapsedPreview,
  isCont,
  isObj,
  escPtr,
} from './tokens'
import { renderBlock } from './render'
import { countLines } from './render'

function hasOpsUnder(path: string, opMap: Map<string, PatchOp>): boolean {
  for (const p of opMap.keys()) if (p === path || p.startsWith(path + '/')) return true
  return false
}

export function buildDiffLines(live: unknown, shadow: unknown, ops: PatchOp[]): DiffLine[] {
  const opMap = new Map<string, PatchOp>()
  for (const op of ops) opMap.set(op.path, op)
  const lines: DiffLine[] = []
  walkMerged(live, shadow, '', 0, opMap, lines)
  return lines
}

function walkMerged(
  live: unknown,
  shadow: unknown,
  path: string,
  indent: number,
  opMap: Map<string, PatchOp>,
  lines: DiffLine[]
): void {
  const op = opMap.get(path)
  if (op && !isCont(live) && !isCont(shadow)) {
    if (op.op === 'replace') {
      lines.push({ tokens: valTok(live), type: 'replaced-old', indent, path })
      lines.push({ tokens: valTok(shadow), type: 'replaced-new', indent, path })
    } else if (op.op === 'remove') {
      lines.push({ tokens: valTok(live), type: 'removed', indent, path })
    } else if (op.op === 'add') {
      lines.push({ tokens: valTok(shadow), type: 'added', indent, path })
    }
    return
  }
  if (isObj(live) && isObj(shadow)) {
    walkObject(live, shadow, path, indent, opMap, lines)
    return
  }
  if (Array.isArray(live) && Array.isArray(shadow)) {
    walkArray(live, shadow, path, indent, opMap, lines)
    return
  }
  if (op) {
    renderBlock(live, indent, 'removed', path, lines)
    renderBlock(shadow, indent, 'added', path, lines)
  } else {
    lines.push({ tokens: valTok(live), type: 'normal', indent, path })
  }
}

function walkObject(
  live: Record<string, unknown>,
  shadow: Record<string, unknown>,
  path: string,
  indent: number,
  opMap: Map<string, PatchOp>,
  lines: DiffLine[]
): void {
  const allKeys = [...new Set([...Object.keys(live), ...Object.keys(shadow)])]
  lines.push({ tokens: [br('{')], type: 'normal', indent, path })
  allKeys.forEach((key, i) => {
    const cp = `${path}/${escPtr(key)}`
    const last = i === allKeys.length - 1
    const inL = key in live,
      inS = key in shadow
    if (inL && inS) {
      if (hasOpsUnder(cp, opMap)) {
        if (isCont(live[key]) && isCont(shadow[key])) {
          lines.push({ tokens: keyTok(key), type: 'normal', indent: indent + 1, path: cp })
          walkMerged(live[key], shadow[key], cp, indent + 1, opMap, lines)
          if (!last) appendComma(lines)
        } else {
          const op = opMap.get(cp)
          if (op && op.op === 'replace') {
            if (isCont(live[key]) || isCont(shadow[key])) {
              lines.push({
                tokens: keyTok(key),
                type: 'replaced-old',
                indent: indent + 1,
                path: cp,
              })
              renderBlock(live[key], indent + 1, 'replaced-old', cp, lines)
              lines.push({
                tokens: keyTok(key),
                type: 'replaced-new',
                indent: indent + 1,
                path: cp,
              })
              renderBlock(shadow[key], indent + 1, 'replaced-new', cp, lines)
              if (!last) appendComma(lines)
            } else {
              lines.push({
                tokens: [...keyTok(key), ...valTok(live[key])],
                type: 'replaced-old',
                indent: indent + 1,
                path: cp,
              })
              lines.push({
                tokens: [...keyTok(key), ...valTok(shadow[key]), ...(last ? [] : [comma()])],
                type: 'replaced-new',
                indent: indent + 1,
                path: cp,
              })
            }
          } else {
            lines.push({ tokens: keyTok(key), type: 'normal', indent: indent + 1, path: cp })
            walkMerged(live[key], shadow[key], cp, indent + 1, opMap, lines)
            if (!last) appendComma(lines)
          }
        }
      } else if (isCont(live[key])) {
        // Unchanged container → emit collapsed placeholder
        lines.push({
          tokens: [...keyTok(key), ...collapsedPreview(live[key])],
          type: 'collapsed',
          indent: indent + 1,
          path: cp,
          collapsedValue: live[key],
          collapsedLineCount: countLines(live[key]),
          collapsedTrailingComma: !last,
        })
      } else {
        lines.push({
          tokens: [...keyTok(key), ...valTok(live[key]), ...(last ? [] : [comma()])],
          type: 'normal',
          indent: indent + 1,
          path: cp,
        })
      }
    } else if (inL) {
      if (isCont(live[key])) {
        lines.push({ tokens: keyTok(key), type: 'removed', indent: indent + 1, path: cp })
        renderBlock(live[key], indent + 1, 'removed', cp, lines)
        if (!last) appendComma(lines)
      } else {
        lines.push({
          tokens: [...keyTok(key), ...valTok(live[key]), ...(last ? [] : [comma()])],
          type: 'removed',
          indent: indent + 1,
          path: cp,
        })
      }
    } else {
      if (isCont(shadow[key])) {
        lines.push({ tokens: keyTok(key), type: 'added', indent: indent + 1, path: cp })
        renderBlock(shadow[key], indent + 1, 'added', cp, lines)
        if (!last) appendComma(lines)
      } else {
        lines.push({
          tokens: [...keyTok(key), ...valTok(shadow[key]), ...(last ? [] : [comma()])],
          type: 'added',
          indent: indent + 1,
          path: cp,
        })
      }
    }
  })
  lines.push({ tokens: [br('}')], type: 'normal', indent, path })
}

function walkArray(
  live: unknown[],
  shadow: unknown[],
  path: string,
  indent: number,
  opMap: Map<string, PatchOp>,
  lines: DiffLine[]
): void {
  const max = Math.max(live.length, shadow.length)
  lines.push({ tokens: [br('[')], type: 'normal', indent, path })
  for (let i = 0; i < max; i++) {
    const cp = `${path}/${i}`
    const last = i === max - 1
    const inL = i < live.length,
      inS = i < shadow.length
    if (inL && inS) {
      if (hasOpsUnder(cp, opMap)) {
        walkMerged(live[i], shadow[i], cp, indent + 1, opMap, lines)
        if (!last) appendComma(lines)
      } else if (isCont(live[i])) {
        // Unchanged container element → collapsed
        lines.push({
          tokens: collapsedPreview(live[i]),
          type: 'collapsed',
          indent: indent + 1,
          path: cp,
          collapsedValue: live[i],
          collapsedLineCount: countLines(live[i]),
          collapsedTrailingComma: !last,
        })
      } else {
        lines.push({
          tokens: [...valTok(live[i]), ...(last ? [] : [comma()])],
          type: 'normal',
          indent: indent + 1,
          path: cp,
        })
      }
    } else if (inL) {
      if (isCont(live[i])) {
        renderBlock(live[i], indent + 1, 'removed', cp, lines)
        if (!last) appendComma(lines)
      } else {
        lines.push({
          tokens: [...valTok(live[i]), ...(last ? [] : [comma()])],
          type: 'removed',
          indent: indent + 1,
          path: cp,
        })
      }
    } else {
      if (isCont(shadow[i])) {
        renderBlock(shadow[i], indent + 1, 'added', cp, lines)
        if (!last) appendComma(lines)
      } else {
        lines.push({
          tokens: [...valTok(shadow[i]), ...(last ? [] : [comma()])],
          type: 'added',
          indent: indent + 1,
          path: cp,
        })
      }
    }
  }
  lines.push({ tokens: [br(']')], type: 'normal', indent, path })
}
