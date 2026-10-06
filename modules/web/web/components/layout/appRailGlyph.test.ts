// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { firstGrapheme, readGraphemeCluster } from './appRailGlyph';

test('firstGrapheme uppercases latin and keeps the first CJK character', () => {
  expect(firstGrapheme('Master Data')).toBe('M');
  expect(firstGrapheme('  company')).toBe('C');
  expect(firstGrapheme('公司管理')).toBe('公');
  expect(firstGrapheme('')).toBe('?');
  expect(firstGrapheme('   ')).toBe('?');
  const emoji = firstGrapheme('👨‍👩‍👧 team');
  expect(emoji.length).toBeGreaterThan(0);
  expect(emoji).not.toBe('?');
});

test('firstGrapheme uses injected cluster readers and code-point fallback', () => {
  expect(firstGrapheme('ab', () => 'z')).toBe('Z');
  expect(firstGrapheme('ab', () => undefined)).toBe('A');
  expect(firstGrapheme('ab', () => '')).toBe('A');
});

test('readGraphemeCluster tolerates missing or throwing Segmenter', () => {
  const intl = Intl as unknown as { Segmenter?: unknown };
  const prev = intl.Segmenter;
  try {
    intl.Segmenter = undefined;
    expect(readGraphemeCluster('ab')).toBeUndefined();
    intl.Segmenter = class {
      constructor() {
        throw new Error('segmenter unavailable');
      }
    };
    expect(readGraphemeCluster('ab')).toBeUndefined();
    intl.Segmenter = class {
      segment() {
        return [];
      }
    };
    expect(readGraphemeCluster('ab')).toBeUndefined();
  } finally {
    intl.Segmenter = prev;
  }
});
