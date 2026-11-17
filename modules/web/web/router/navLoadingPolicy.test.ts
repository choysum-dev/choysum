// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { resolveNavLoadingPolicy } from './navLoadingPolicy';

test('resolveNavLoadingPolicy defaults omit and unknown to content', () => {
  expect(resolveNavLoadingPolicy({ meta: {} })).toBe('content');
  expect(resolveNavLoadingPolicy({ meta: { navLoading: undefined } })).toBe('content');
  expect(resolveNavLoadingPolicy({ meta: { navLoading: 'nope' as any } })).toBe('content');
});

test('resolveNavLoadingPolicy returns explicit content, route, and none', () => {
  expect(resolveNavLoadingPolicy({ meta: { navLoading: 'content' } })).toBe('content');
  expect(resolveNavLoadingPolicy({ meta: { navLoading: 'route' } })).toBe('route');
  expect(resolveNavLoadingPolicy({ meta: { navLoading: 'none' } })).toBe('none');
});
