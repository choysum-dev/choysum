// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export type ClassValue = string | false | null | undefined | ClassValue[];

/**
 * Merges class name inputs into a single space-separated string.
 */
export function cn(...inputs: ClassValue[]): string {
  const out: string[] = [];
  const visit = (value: ClassValue): void => {
    if (!value) {
      return;
    }
    if (Array.isArray(value)) {
      for (const item of value) {
        visit(item);
      }
      return;
    }
    out.push(value);
  };
  for (const input of inputs) {
    visit(input);
  }
  return out.join(' ');
}
