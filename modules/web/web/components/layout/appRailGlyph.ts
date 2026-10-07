// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

type GraphemeClusterFn = (input: string) => string | undefined;

/**
 * First grapheme cluster of `input` when Intl.Segmenter is available.
 * Returns undefined when the host has no Segmenter or the call fails.
 */
export function readGraphemeCluster(input: string): string | undefined {
  try {
    const Segmenter = (Intl as unknown as {
      Segmenter?: new (
        locale?: string,
        opts?: { granularity: string },
      ) => { segment: (text: string) => Iterable<{ segment: string }> };
    }).Segmenter;
    if (!Segmenter) {
      return undefined;
    }
    const segments = new Segmenter(undefined, { granularity: 'grapheme' }).segment(input);
    return segments[Symbol.iterator]().next().value?.segment;
  } catch {
    return undefined;
  }
}

/**
 * First visible grapheme of an app title for the icon rail when no icon is set.
 * Latin letters are uppercased; CJK and other scripts keep the first character.
 */
export function firstGrapheme(title: string, clusterOf: GraphemeClusterFn = readGraphemeCluster): string {
  const trimmed = String(title || '').trim();
  if (!trimmed) return '?';
  const ch = clusterOf(trimmed) || Array.from(trimmed)[0] || '?';
  return ch.toUpperCase();
}
