// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { syncFnRecorder } from '@/web/web/__tests__/mountApp';
import { canShowAction } from './actionVisibility';

describe('canShowAction', () => {
  test('returns true when action id is empty', () => {
    const checker = syncFnRecorder(() => false);
    expect(canShowAction(undefined, checker)).toBe(true);
    expect(canShowAction('', checker)).toBe(true);
    expect(checker.calls.length).toBe(0);
  });

  test('returns true when checker is missing', () => {
    expect(canShowAction('auth.action.user_create', undefined)).toBe(true);
  });

  test('delegates to checker when both are provided', () => {
    const checker = syncFnRecorder((id: string | undefined) => id === 'auth.action.user_create');
    expect(canShowAction('auth.action.user_create', checker)).toBe(true);
    expect(canShowAction('auth.action.user_delete', checker)).toBe(false);
    expect(checker.calls.length).toBe(2);
  });
});
