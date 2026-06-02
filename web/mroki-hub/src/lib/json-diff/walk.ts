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
import {
  buildContext,
  hasOpsUnder,
  leafOp,
  classifyArray,
  alignArray,
  type DiffContext,
} from './align'

export function buildDiffLines(live: unknown, shadow: unknown, ops: PatchOp[]): DiffLine[] {
  const ctx = buildContext(ops)
  const lines: DiffLine[] = []
  walkMerged(live, shadow, '', 0, ctx, lines)
  return lines
}

function walkMerged(
  live: unknown,
  shadow: unknown,
  path: string,
  indent: number,
  ctx: DiffContext,
  lines: DiffLine[]
): void {
  const op = leafOp(ctx, path)
  // The `!isCont` guard is load-bearing: an x-indexed `remove` and a y-indexed
  // op can share the same numeric pointer, so `leafOp` may return the remove for
  // an array slot we entered as a kept (container) element. Restricting the leaf
  // branch to primitives keeps containers on the recursive path below; preserve
  // this guard if `walkArray` is ever refactored.
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
    walkObject(live, shadow, path, indent, ctx, lines)
    return
  }
  if (Array.isArray(live) && Array.isArray(shadow)) {
    walkArray(live, shadow, path, indent, ctx, lines)
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
  ctx: DiffContext,
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
      if (hasOpsUnder(ctx, cp)) {
        if (isCont(live[key]) && isCont(shadow[key])) {
          lines.push({ tokens: keyTok(key), type: 'normal', indent: indent + 1, path: cp })
          walkMerged(live[key], shadow[key], cp, indent + 1, ctx, lines)
          if (!last) appendComma(lines)
        } else {
          const op = leafOp(ctx, cp)
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
            walkMerged(live[key], shadow[key], cp, indent + 1, ctx, lines)
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
  ctx: DiffContext,
  lines: DiffLine[]
): void {
  // Align live (x) and shadow (y) via the edit script reconstructed from the
  // ops, rather than walking by position — positional walking silently misses
  // reordered/inserted/removed elements (issue #114).
  const entries = alignArray(live.length, shadow.length, classifyArray(ctx, path))
  lines.push({ tokens: [br('[')], type: 'normal', indent, path })
  entries.forEach((e, i) => {
    const last = i === entries.length - 1
    if (e.kind === 'equal') {
      const v = live[e.liveIndex!]
      const cp = `${path}/${e.shadowIndex}`
      if (isCont(v)) {
        lines.push({
          tokens: collapsedPreview(v),
          type: 'collapsed',
          indent: indent + 1,
          path: cp,
          collapsedValue: v,
          collapsedLineCount: countLines(v),
          collapsedTrailingComma: !last,
        })
      } else {
        lines.push({
          tokens: [...valTok(v), ...(last ? [] : [comma()])],
          type: 'normal',
          indent: indent + 1,
          path: cp,
        })
      }
    } else if (e.kind === 'replace') {
      const lv = live[e.liveIndex!]
      const sv = shadow[e.shadowIndex!]
      const cp = `${path}/${e.shadowIndex}`
      if (isCont(lv) && isCont(sv)) {
        walkMerged(lv, sv, cp, indent + 1, ctx, lines)
        if (!last) appendComma(lines)
      } else if (isCont(lv) || isCont(sv)) {
        renderBlock(lv, indent + 1, 'replaced-old', cp, lines)
        renderBlock(sv, indent + 1, 'replaced-new', cp, lines)
        if (!last) appendComma(lines)
      } else {
        lines.push({ tokens: valTok(lv), type: 'replaced-old', indent: indent + 1, path: cp })
        lines.push({
          tokens: [...valTok(sv), ...(last ? [] : [comma()])],
          type: 'replaced-new',
          indent: indent + 1,
          path: cp,
        })
      }
    } else if (e.kind === 'remove') {
      const v = live[e.liveIndex!]
      const cp = `${path}/${e.liveIndex}`
      if (isCont(v)) {
        renderBlock(v, indent + 1, 'removed', cp, lines)
        if (!last) appendComma(lines)
      } else {
        lines.push({
          tokens: [...valTok(v), ...(last ? [] : [comma()])],
          type: 'removed',
          indent: indent + 1,
          path: cp,
        })
      }
    } else {
      const v = shadow[e.shadowIndex!]
      const cp = `${path}/${e.shadowIndex}`
      if (isCont(v)) {
        renderBlock(v, indent + 1, 'added', cp, lines)
        if (!last) appendComma(lines)
      } else {
        lines.push({
          tokens: [...valTok(v), ...(last ? [] : [comma()])],
          type: 'added',
          indent: indent + 1,
          path: cp,
        })
      }
    }
  })
  lines.push({ tokens: [br(']')], type: 'normal', indent, path })
}
