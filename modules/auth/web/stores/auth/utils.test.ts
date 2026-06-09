// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';

import { hashPasswordClient } from './utils';

describe('hashPasswordClient', () => {
  it('returns raw password on non-client runtime', async () => {
    const got = await hashPasswordClient('plain-secret', 'admin');
    expect(got).toBe('plain-secret');
  });
});
