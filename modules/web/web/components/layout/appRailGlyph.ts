// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * First visible grapheme of an app title for the icon rail when no icon is set.
 * Latin letters are uppercased; CJK and other scripts keep the first character.
 */
export function firstGrapheme(title: string): string {
  const trimmed = String(title || '').trim();
  if (!trimmed) return '?';
  let ch: string | undefined;
  try {
    const Segmenter = (Intl as unknown as {
      Segmenter?: new (
        locale?: string,
        opts?: { granularity: string },
      ) => { segment: (input: string) => Iterable<{ segment: string }> };
    }).Segmenter;
    if (Segmenter) {
      ch = Array.from(new Segmenter(undefined, { granularity: 'grapheme' }).segment(trimmed))[0]
        ?.segment;
    }
  } catch {
    ch = undefined;
  }
  ch = ch || Array.from(trimmed)[0] || '?';
  return /[a-z]/i.test(ch) ? ch.toUpperCase() : ch;
}
