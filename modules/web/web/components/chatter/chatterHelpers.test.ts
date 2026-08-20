// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { formatFieldChangeSummary } from './chatterHelpers';

describe('formatFieldChangeSummary', () => {
  const labels = {
    created: 'Record created',
    unlinked: 'Record removed',
    changed: (field: string, oldValue: string, newValue: string) => `${field}:${oldValue}->${newValue}`,
    action: (name: string) => `Action:${name}`,
  };

  it('formats create, unlink, field, and action kinds', () => {
    expect(formatFieldChangeSummary({ kind: 'fieldChange', id: '1', at: 1, field: null, changeKind: 'create', oldValue: null, newValue: null, actorUid: null }, labels)).toBe('Record created');
    expect(formatFieldChangeSummary({ kind: 'fieldChange', id: '2', at: 1, field: null, changeKind: 'unlink', oldValue: null, newValue: null, actorUid: null }, labels)).toBe('Record removed');
    expect(
      formatFieldChangeSummary(
        { kind: 'fieldChange', id: '3', at: 1, field: 'Name', changeKind: 'field', oldValue: 'A', newValue: 'B', actorUid: 'u1' },
        labels
      )
    ).toBe('Name:A->B');
    expect(
      formatFieldChangeSummary(
        { kind: 'fieldChange', id: '4', at: 1, field: null, changeKind: 'action:confirm', oldValue: null, newValue: null, actorUid: 'u1' },
        labels
      )
    ).toBe('Action:confirm');
  });
});
