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
    fieldFallback: 'Field',
  };

  it('formats create, unlink, field, and action kinds', () => {
    expect(formatFieldChangeSummary({ kind: 'fieldChange', id: '1', at: 1, field: null, changeKind: ' create ', oldValue: null, newValue: null, actorUid: null }, labels)).toBe('Record created');
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

  it('uses the localized field fallback when the audit row has no field name', () => {
    expect(
      formatFieldChangeSummary(
        { kind: 'fieldChange', id: '5', at: 1, field: null, changeKind: 'field', oldValue: 'A', newValue: 'B', actorUid: null },
        { ...labels, fieldFallback: '字段' }
      )
    ).toBe('字段:A->B');
  });

  it('normalizes empty values and bare action kinds', () => {
    expect(
      formatFieldChangeSummary(
        { kind: 'fieldChange', id: '6', at: 1, field: 'Name', changeKind: 'field', oldValue: '', newValue: null, actorUid: null },
        labels
      )
    ).toBe('Name:—->—');
    expect(
      formatFieldChangeSummary(
        { kind: 'fieldChange', id: '7', at: 1, field: null, changeKind: 'action:', oldValue: null, newValue: null, actorUid: null },
        labels
      )
    ).toBe('Action:action:');
    expect(
      formatFieldChangeSummary(
        { kind: 'fieldChange', id: '8', at: 1, field: 'Status', changeKind: undefined as any, oldValue: 'open', newValue: '', actorUid: null },
        labels
      )
    ).toBe('Status:open->—');
    expect(
      formatFieldChangeSummary(
        { kind: 'fieldChange', id: '9', at: 1, field: '', changeKind: null as any, oldValue: null, newValue: 'done', actorUid: null },
        labels
      )
    ).toBe('Field:—->done');
  });
});
