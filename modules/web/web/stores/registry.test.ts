// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { listRegisteredModelNames, registerStoreFactory } from './registry';

describe('listRegisteredModelNames', () => {
  it('returns registered model names including newly registered factories', () => {
    const marker = `__test__.listRegisteredModelNames.${Date.now()}`;
    registerStoreFactory(marker, () => ({}) as any);

    const names = listRegisteredModelNames();
    expect(Array.isArray(names)).toBe(true);
    expect(names).toContain(marker);
  });
});
