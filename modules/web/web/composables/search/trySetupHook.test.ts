// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { trySetupHook } from './trySetupHook';

describe('trySetupHook', () => {
  it('returns the hook result', () => {
    expect(trySetupHook(() => 'ok')).toBe('ok');
    expect(trySetupHook(() => 42)).toBe(42);
  });

  it('returns null when the hook throws', () => {
    expect(
      trySetupHook(() => {
        throw new Error('no inject');
      })
    ).toBeNull();
  });
});
