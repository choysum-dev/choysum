// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { resolveTreeBoxStyle } from './treeBoxHeight';

test('resolveTreeBoxStyle: number → fixed px height', () => {
  expect(resolveTreeBoxStyle(180)).toEqual({ height: '180px' });
  expect(resolveTreeBoxStyle(360)).toEqual({ height: '360px' });
});

test('resolveTreeBoxStyle: auto → content height', () => {
  expect(resolveTreeBoxStyle('auto')).toEqual({ height: 'auto' });
});

test('resolveTreeBoxStyle: CSS length string passes through', () => {
  expect(resolveTreeBoxStyle('20rem')).toEqual({ height: '20rem' });
  expect(resolveTreeBoxStyle('50vh')).toEqual({ height: '50vh' });
});

test('resolveTreeBoxStyle: invalid / empty falls back to auto', () => {
  expect(resolveTreeBoxStyle(0)).toEqual({ height: 'auto' });
  expect(resolveTreeBoxStyle(-1)).toEqual({ height: 'auto' });
  expect(resolveTreeBoxStyle(undefined)).toEqual({ height: 'auto' });
  expect(resolveTreeBoxStyle('')).toEqual({ height: 'auto' });
});
