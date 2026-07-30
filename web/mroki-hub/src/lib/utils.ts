import type { ClassValue } from 'clsx'
import { clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

/**
 * Truncate a UUID or long ID to a shorter display format
 * @param id - The ID to truncate
 * @param length - Number of characters to keep (default: 8)
 * @returns Truncated ID
 */
export function truncateId(id: string, length = 8): string {
  return id.substring(0, length)
}

/**
 * Map a diff rate (percentage, 0–100) to a semantic text color class.
 * A low rate means live and shadow responses match, so it reads as healthy;
 * a high rate signals significant divergence.
 * @param rate - Diff rate as a percentage
 * @returns A semantic `text-*` utility class
 */
export function diffRateColorClass(rate: number): string {
  if (rate >= 10) return 'text-danger'
  if (rate >= 1) return 'text-warning'
  return 'text-success'
}

/**
 * Convert an RFC 6901 JSON Pointer (as used by diff patch rows) into the
 * gjson dot-notation path used by a gate's `ignored_fields`.
 *
 * Example: `/body/items/0/price` -> `body.items.0.price`,
 *          `/headers/X-Api-Key`  -> `headers.X-Api-Key`.
 *
 * Pointer escapes are decoded per RFC 6901 (`~1` -> `/`, `~0` -> `~`).
 * @param pointer - An RFC 6901 JSON Pointer, optionally leading with `/`
 * @returns The equivalent gjson dot-notation path
 */
export function pointerToGjson(pointer: string): string {
  return pointer
    .replace(/^\//, '')
    .split('/')
    .map((s) => s.replace(/~1/g, '/').replace(/~0/g, '~'))
    .join('.')
}

/**
 * Map an HTTP method to its tinted badge classes. GET reads as informational,
 * POST as a successful create, PUT/PATCH as a warning-toned mutation, DELETE as
 * destructive; unknown verbs fall back to a muted neutral.
 * @param method - An HTTP method (case-insensitive)
 * @returns Tailwind classes for the method badge background and text
 */
export function methodColorClass(method: string): string {
  switch (method.toUpperCase()) {
    case 'GET':
      return 'bg-info/15 text-info'
    case 'POST':
      return 'bg-success/15 text-success'
    case 'PUT':
    case 'PATCH':
      return 'bg-warning/15 text-warning'
    case 'DELETE':
      return 'bg-danger/15 text-danger'
    default:
      return 'bg-muted text-muted-foreground'
  }
}
