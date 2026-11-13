// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { useDetachedField } from './useDetachedField.ts';

describe('useDetachedField', () => {
  test('returns a store-less binding in form edit mode', () => {
    const binding = useDetachedField();
    expect(binding.store).toBeUndefined();
    expect(binding.prop).toBe('_slot');
    expect(binding.env.isEditMode).toBe(true);
    expect(binding.env.isForm).toBe(true);
    expect(binding.env.fieldPresentation).toBe('form');
    expect(binding.fieldRef().value).toBe(null);
  });

  test('accepts custom prop name', () => {
    const binding = useDetachedField({ prop: 'PasswordAction' });
    expect(binding.prop).toBe('PasswordAction');
  });
});
