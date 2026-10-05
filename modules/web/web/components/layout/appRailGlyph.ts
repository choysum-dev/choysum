// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * First visible grapheme of an app title for the icon rail when no icon is set.
 * Latin letters are uppercased; CJK and other scripts keep the first character.
 */
export function firstGrapheme(title: string): string {
  const trimmed = String(title || '').trim();
  if (!trimmed) return '?';
  const ch = Array.from(trimmed)[0] || '?';
  return /[a-z]/i.test(ch) ? ch.toUpperCase() : ch;
}
