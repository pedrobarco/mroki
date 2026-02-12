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
