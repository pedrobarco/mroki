import type { PatchOp } from '@/api'

export function stripPathPrefix(ops: PatchOp[], prefix: string): PatchOp[] {
  return ops
    .filter((op) => op.path.startsWith(prefix + '/') || op.path === prefix)
    .map((op) => ({ ...op, path: op.path === prefix ? '/' : op.path.slice(prefix.length) }))
}
