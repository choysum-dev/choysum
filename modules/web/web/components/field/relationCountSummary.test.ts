// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { formatRelationCountSummary } from './relationCountSummary';

function fakeT(msg: string, ...args: unknown[]): string {
  if (!args.length) return msg;
  return args.reduce<string>((s, a) => s.replace('%s', String(a)), msg);
}

test('formatRelationCountSummary: empty uses em-dash', () => {
  expect(formatRelationCountSummary(0, fakeT)).toBe('—');
  expect(formatRelationCountSummary(-1, fakeT)).toBe('—');
  expect(formatRelationCountSummary(Number.NaN, fakeT)).toBe('—');
});

test('formatRelationCountSummary: positive count', () => {
  expect(formatRelationCountSummary(1, fakeT)).toBe('1 records');
  expect(formatRelationCountSummary(3, fakeT)).toBe('3 records');
});
