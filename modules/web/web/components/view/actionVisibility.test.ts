// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';
import { canShowAction } from './actionVisibility';

describe('canShowAction', () => {
  it('returns true when action id is empty', () => {
    const checker = vi.fn(() => false);
    expect(canShowAction(undefined, checker)).toBe(true);
    expect(canShowAction('', checker)).toBe(true);
    expect(checker).not.toHaveBeenCalled();
  });

  it('returns true when checker is missing', () => {
    expect(canShowAction('auth.action.user_create', undefined)).toBe(true);
  });

  it('delegates to checker when both are provided', () => {
    const checker = vi.fn((id: string | undefined) => id === 'auth.action.user_create');
    expect(canShowAction('auth.action.user_create', checker)).toBe(true);
    expect(canShowAction('auth.action.user_delete', checker)).toBe(false);
    expect(checker).toHaveBeenCalledTimes(2);
  });
});
