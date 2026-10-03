// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Class-name helper for L2 / L1. Accepts string trees and Vue `ClassValue`
 * (incl. number / record forms from `HTMLAttributes["class"]`).
 */
export type ClassValue =
  | string
  | number
  | boolean
  | null
  | undefined
  | ClassValue[]
  | Record<string, unknown>;

/**
 * Merges class name inputs into a single space-separated string.
 */
export function cn(...inputs: ClassValue[]): string {
  const out: string[] = [];
  const visit = (value: ClassValue): void => {
    if (value === false || value === null || value === undefined || value === '') {
      return;
    }
    if (typeof value === 'number') {
      out.push(String(value));
      return;
    }
    if (typeof value === 'string') {
      out.push(value);
      return;
    }
    if (Array.isArray(value)) {
      for (const item of value) {
        visit(item);
      }
      return;
    }
    if (typeof value === 'object') {
      for (const [key, on] of Object.entries(value)) {
        if (on) {
          out.push(key);
        }
      }
    }
  };
  for (const input of inputs) {
    visit(input);
  }
  return out.join(' ');
}
