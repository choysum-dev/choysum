// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { afterEach, describe, expect, it } from 'vitest';
import { nextLocalToken, resetLocalTokenCounterForTest } from './localToken';

describe('nextLocalToken', () => {
  afterEach(() => {
    resetLocalTokenCounterForTest();
  });

  it('uses randomUUID when available', () => {
    const token = nextLocalToken('o-form-view', {
      randomUUID: () => 'uuid-123',
    });

    expect(token).toBe('o-form-view:uuid-123');
  });

  it('falls back to time and incrementing counter when randomUUID is unavailable', () => {
    const first = nextLocalToken('FormView:new', { randomUUID: undefined, now: () => 36 });
    const second = nextLocalToken('FormView:new', { randomUUID: undefined, now: () => 36 });

    expect(first).toBe('FormView:new:10:1');
    expect(second).toBe('FormView:new:10:2');
  });
});
