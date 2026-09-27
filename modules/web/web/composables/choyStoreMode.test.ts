// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { hasChoyStoreEngine, isChoyStoreFieldBinding } from './choyStoreMode';

describe('choyStoreMode', () => {
  test('isChoyStoreFieldBinding requires store+prop or binding', () => {
    expect(isChoyStoreFieldBinding({})).toBe(false);
    expect(isChoyStoreFieldBinding({ store: {} })).toBe(false);
    expect(isChoyStoreFieldBinding({ prop: 'Name' })).toBe(false);
    expect(isChoyStoreFieldBinding({ store: {}, prop: '' })).toBe(false);
    expect(isChoyStoreFieldBinding({ store: {}, prop: 'Name' })).toBe(true);
    expect(isChoyStoreFieldBinding({ binding: {} })).toBe(true);
  });

  test('hasChoyStoreEngine accepts prop or page store', () => {
    expect(hasChoyStoreEngine(undefined, undefined)).toBe(false);
    expect(hasChoyStoreEngine(null, null)).toBe(false);
    expect(hasChoyStoreEngine({ model: 'x' }, null)).toBe(true);
    expect(hasChoyStoreEngine(undefined, { model: 'x' })).toBe(true);
  });
});
