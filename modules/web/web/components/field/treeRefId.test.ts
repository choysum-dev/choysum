// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { normalizeTreeRefId } from './treeRefId';

describe('normalizeTreeRefId', () => {
  it('normalizes plain ids and objects with Id', () => {
    expect(normalizeTreeRefId(null)).toBeNull();
    expect(normalizeTreeRefId(undefined)).toBeNull();
    expect(normalizeTreeRefId('  abc  ')).toBe('abc');
    expect(normalizeTreeRefId({ Id: ' node-1 ' })).toBe('node-1');
  });

  it('unwraps toEntity proxies before normalizing', () => {
    expect(
      normalizeTreeRefId({
        toEntity: () => ({ Id: ' from-entity ' }),
      })
    ).toBe('from-entity');
  });
});
