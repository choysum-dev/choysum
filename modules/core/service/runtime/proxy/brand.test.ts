// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getProxyKind, isBrandedProxy, markProxyKind } from './brand';

test('markProxyKind stores kind on proxy object via WeakMap', () => {
  const target = { Name: 'Alice' };
  const proxy = new Proxy(target, {});
  markProxyKind(proxy, 'onchange-preview');

  expect(getProxyKind(proxy)).toBe('onchange-preview');
  expect(isBrandedProxy(proxy)).toBe(true);
  expect(getProxyKind(target)).toBe(undefined);
});

test('getProxyKind returns undefined for plain objects', () => {
  expect(getProxyKind({})).toBe(undefined);
  expect(getProxyKind(null)).toBe(undefined);
  expect(isBrandedProxy(undefined)).toBe(false);
});
