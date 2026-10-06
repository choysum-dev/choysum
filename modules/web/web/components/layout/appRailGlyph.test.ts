// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { firstGrapheme } from './appRailGlyph';

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
