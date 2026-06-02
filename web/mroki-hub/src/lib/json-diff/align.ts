import type { PatchOp } from '@/api'

/**
 * Shared array-alignment helpers for the JSON diff views.
 *
 * Patch ops address array elements asymmetrically: `remove` uses the live (x)
 * index, while `add`/`replace` and any deeper modification use the shadow (y)
 * index (see pkg/diff/reporter.go). Keying ops in a flat `Map<path, op>` is
 * therefore lossy — an x-indexed remove and a y-indexed add can collide on the
 * same pointer string. These helpers pre-group the raw op list once and expose
 * an ordered, non-lossy alignment between the live and shadow arrays.
 */

/** One aligned slot between the live (x) and shadow (y) arrays. */
export interface AlignEntry {
  kind: 'equal' | 'replace' | 'add' | 'remove'
  /** Index in the live array (x). Null for `add`. */
  liveIndex: number | null
  /** Index in the shadow array (y). Null for `remove`. */
  shadowIndex: number | null
}

export interface DiffContext {
  /** Ops grouped by their immediate parent pointer (the container's path). */
  byParent: Map<string, PatchOp[]>
  /** Every ancestor-or-self pointer that has an op at or beneath it. */
  prefixSet: Set<string>
  /** Pointers that have an op *strictly* beneath them (proper ancestors only). */
  deepPrefixSet: Set<string>
}

/** Pre-group ops by parent pointer and record ancestor prefixes in one O(ops) pass. */
export function buildContext(ops: PatchOp[]): DiffContext {
  const byParent = new Map<string, PatchOp[]>()
  const prefixSet = new Set<string>()
  const deepPrefixSet = new Set<string>()
  for (const op of ops) {
    const parent = op.path.slice(0, op.path.lastIndexOf('/'))
    const bucket = byParent.get(parent)
    if (bucket) bucket.push(op)
    else byParent.set(parent, [op])

    const segs = op.path.split('/')
    let acc = ''
    for (let k = 1; k < segs.length; k++) {
      acc += '/' + segs[k]
      prefixSet.add(acc)
      // Proper ancestors (everything but the op's own leaf) have an op strictly below.
      if (k < segs.length - 1) deepPrefixSet.add(acc)
    }
  }
  return { byParent, prefixSet, deepPrefixSet }
}

/** True when an op exists at `path` or anywhere beneath it. */
export function hasOpsUnder(ctx: DiffContext, path: string): boolean {
  return ctx.prefixSet.has(path)
}

/** Find the op addressed at exactly `path` (preferring a replace on collision). */
export function leafOp(ctx: DiffContext, path: string): PatchOp | undefined {
  const parent = path.slice(0, path.lastIndexOf('/'))
  const bucket = ctx.byParent.get(parent)
  if (!bucket) return undefined
  return (
    bucket.find((o) => o.path === path && o.op === 'replace') ?? bucket.find((o) => o.path === path)
  )
}

export interface ArrayClassification {
  /** Live (x) indices that were removed. */
  removeX: Set<number>
  /** Shadow (y) indices that were added. */
  addY: Set<number>
  /** Whether the kept shadow element at index y changed (replace or deeper op). */
  isChangedY: (y: number) => boolean
}

/** Classify the direct element ops of the array at `path` into remove/add/change. */
export function classifyArray(ctx: DiffContext, path: string): ArrayClassification {
  const direct = ctx.byParent.get(path) ?? []
  const removeX = new Set<number>()
  const addY = new Set<number>()
  const replaceY = new Set<number>()
  for (const op of direct) {
    const idx = Number(op.path.slice(path.length + 1))
    if (!Number.isInteger(idx)) continue
    if (op.op === 'remove') removeX.add(idx)
    else if (op.op === 'add') addY.add(idx)
    else if (op.op === 'replace') replaceY.add(idx)
  }
  // A kept (non-added) shadow element changed iff it has a replace at its own
  // pointer or any op strictly beneath it. A bare `remove` at the same numeric
  // string is x-indexed (a different element) and must not count here.
  const isChangedY = (y: number): boolean =>
    replaceY.has(y) || ctx.deepPrefixSet.has(`${path}/${y}`)
  return { removeX, addY, isChangedY }
}

/**
 * Reconstruct go-cmp's array edit script from the classified ops.
 *
 * Dropping the removed x-indices from the live array and the added y-indices
 * from the shadow array leaves two equal-length sequences (the common
 * subsequence) that pair up positionally; each pair is `equal` or `replace`.
 * Removes and adds are interleaved by a two-pointer merge (removes first when a
 * remove and an add fall at the same gap).
 *
 * The reconstruction emits entries in document order. This matches the Patch
 * view only on the *set and counts* of changes, not necessarily the line order
 * (the Patch view walks ops in go-cmp emission order). The #3 consistency
 * guarantee is therefore "same changes", not "same line-for-line ordering".
 *
 * Precondition: `liveKept.length === shadowKept.length` — i.e. the classified
 * ops are self-consistent (every kept live element pairs with a kept shadow
 * element), which always holds for real backend output where exactly the
 * removed and added indices are dropped. If the ops are ever inconsistent
 * (hand-built or garbled), `pairCount` clamps to the shorter side and the
 * surplus kept elements degrade gracefully into `remove`/`add` entries rather
 * than throwing — a robustness fallback, not an expected path.
 */
export function alignArray(
  liveLen: number,
  shadowLen: number,
  { removeX, addY, isChangedY }: ArrayClassification
): AlignEntry[] {
  const liveKept: number[] = []
  for (let i = 0; i < liveLen; i++) if (!removeX.has(i)) liveKept.push(i)
  const shadowKept: number[] = []
  for (let j = 0; j < shadowLen; j++) if (!addY.has(j)) shadowKept.push(j)

  const pairCount = Math.min(liveKept.length, shadowKept.length)
  const entries: AlignEntry[] = []
  let xi = 0
  let yi = 0
  let t = 0
  while (xi < liveLen || yi < shadowLen) {
    const px = t < pairCount ? liveKept[t]! : -1
    const py = t < pairCount ? shadowKept[t]! : -1
    if (t < pairCount && xi === px && yi === py) {
      entries.push({ kind: isChangedY(py) ? 'replace' : 'equal', liveIndex: px, shadowIndex: py })
      xi++
      yi++
      t++
    } else if (xi < liveLen && (t >= pairCount || xi < px)) {
      entries.push({ kind: 'remove', liveIndex: xi, shadowIndex: null })
      xi++
    } else {
      entries.push({ kind: 'add', liveIndex: null, shadowIndex: yi })
      yi++
    }
  }
  return entries
}
