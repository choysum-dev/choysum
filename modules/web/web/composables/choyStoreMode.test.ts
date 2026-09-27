// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  hasChoyStoreEngine,
  isChoyStoreFieldBinding,
  splitChoyAttrsListeners,
} from './choyStoreMode';

describe('choyStoreMode', () => {
  test('isChoyStoreFieldBinding requires store+prop or binding', () => {
    expect(isChoyStoreFieldBinding({})).toBe(false);
    expect(isChoyStoreFieldBinding({ store: {} })).toBe(false);
    expect(isChoyStoreFieldBinding({ prop: 'Name' })).toBe(false);
    expect(isChoyStoreFieldBinding({ store: {}, prop: '' })).toBe(false);
    expect(isChoyStoreFieldBinding({ store: {}, prop: 'Name' })).toBe(true);
    expect(isChoyStoreFieldBinding({ binding: {} })).toBe(true);
  });

  test('isChoyStoreFieldBinding accepts page store with prop', () => {
    expect(isChoyStoreFieldBinding({ prop: 'Name' }, { model: 'x' })).toBe(true);
    expect(isChoyStoreFieldBinding({ prop: '' }, { model: 'x' })).toBe(false);
    expect(isChoyStoreFieldBinding({ store: { a: 1 }, prop: 'Name' }, { model: 'x' })).toBe(
      true,
    );
  });

  test('hasChoyStoreEngine accepts prop or page store', () => {
    expect(hasChoyStoreEngine(undefined, undefined)).toBe(false);
    expect(hasChoyStoreEngine(null, null)).toBe(false);
    expect(hasChoyStoreEngine({ model: 'x' }, null)).toBe(true);
    expect(hasChoyStoreEngine(undefined, { model: 'x' })).toBe(true);
  });

  test('splitChoyAttrsListeners separates on* functions from bind attrs', () => {
    const onFoo = () => undefined;
    const { bind, listeners } = splitChoyAttrsListeners({
      class: 'x',
      'data-test': 't',
      onFoo,
      onBar: 'not-a-function',
    });
    expect(bind).toEqual({ class: 'x', 'data-test': 't', onBar: 'not-a-function' });
    expect(listeners).toEqual({ onFoo });
  });
});
