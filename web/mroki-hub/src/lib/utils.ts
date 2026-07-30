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
