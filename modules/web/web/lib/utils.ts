// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { twMerge } from 'tailwind-merge';

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
 * Flattens class inputs with clsx-like semantics (falsy skip, arrays, truthy keys).
 */
function flatten(inputs: ClassValue[]): string {
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

/**
 * Merges class name inputs: flatten (clsx-like), then Tailwind conflict resolution
 * so later same-group utilities win (shadcn `twMerge(clsx(...))` shape).
 */
export function cn(...inputs: ClassValue[]): string {
  return twMerge(flatten(inputs));
}

/**
 * Shared focus chrome for form controls (Input, Selection, ManyToOne, Select, …):
 * thin border-ring + soft ring — the Varchar field look.
 */
export const controlFocusRing =
  'outline-none transition-[color,box-shadow] focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50';

/** Same chrome when the control is a composite that focuses an inner child (e.g. TagsInput). */
export const controlFocusWithinRing =
  'outline-none transition-[color,box-shadow] focus-within:border-ring focus-within:ring-3 focus-within:ring-ring/50';
