// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { resolveFieldErrorSpace } from './fieldErrorSpace';

test('resolveFieldErrorSpace: prop wins over inject', () => {
  expect(resolveFieldErrorSpace('reserve', 'overlay')).toBe('reserve');
  expect(resolveFieldErrorSpace('collapse', 'overlay')).toBe('collapse');
  expect(resolveFieldErrorSpace('overlay', 'reserve')).toBe('overlay');
});

test('resolveFieldErrorSpace: inject when prop unset', () => {
  expect(resolveFieldErrorSpace(undefined, 'overlay')).toBe('overlay');
  expect(resolveFieldErrorSpace(null, 'reserve')).toBe('reserve');
});

test('resolveFieldErrorSpace: collapse when both unset', () => {
  expect(resolveFieldErrorSpace(undefined, null)).toBe('collapse');
  expect(resolveFieldErrorSpace(undefined, undefined)).toBe('collapse');
  expect(resolveFieldErrorSpace('nope' as any, 'also' as any)).toBe('collapse');
});
