// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import {
  cloneRowDraft,
  collectRowDirtyPayload,
  hasHandleField,
  isListRecordRow,
  isRowDraftDirty,
  renumberSequence,
  unwrapListRecord,
  withEditingPayload,
} from '@/web/web/composables/listRowEdit';
import type { WebModelStore } from '@/web/web/stores/modelStore';

describe('listRowEdit helpers', () => {
  it('unwrapListRecord reads controller RecordRow payload', () => {
    const payload = { Id: '1', Name: 'A' };
    expect(unwrapListRecord({ kind: 'record', payload })).toEqual(payload);
  });

  it('isListRecordRow rejects group and more rows', () => {
    expect(isListRecordRow({ kind: 'group' })).toBe(false);
    expect(isListRecordRow({ kind: 'more' })).toBe(false);
    expect(isListRecordRow({ kind: 'record', payload: { Id: '1' } })).toBe(true);
  });

  it('cloneRowDraft deep-clones plain row data', () => {
    const src = { Id: '1', Name: 'A', nested: { x: 1 } };
    const draft = cloneRowDraft(src);
    expect(draft).toEqual(src);
    expect(draft).not.toBe(src);
    draft.nested.x = 2;
    expect(src.nested.x).toBe(1);
  });

  it('renumberSequence writes 1..n on handleField', () => {
    const rows = [{ Sequence: 10 }, { Sequence: 20 }, { Sequence: 30 }];
    const changed = renumberSequence(rows, 'Sequence');
    expect(rows.map(r => r.Sequence)).toEqual([1, 2, 3]);
    expect(changed).toHaveLength(3);
  });

  it('renumberSequence respects pagination start offset', () => {
    const rows = [{ Sequence: 1 }, { Sequence: 2 }];
    renumberSequence(rows, 'Sequence', 21);
    expect(rows.map(r => r.Sequence)).toEqual([21, 22]);
  });

  it('hasHandleField requires numeric writable metadata field', () => {
    const store = {
      fieldsMetadata: {
        Sequence: { id: '1', type: 'int', typeAnnotation: '', isReadonly: false },
        Name: { id: '2', type: 'string', typeAnnotation: '' },
      },
    } as unknown as WebModelStore<any>;
    expect(hasHandleField(store, 'Sequence')).toBe(true);
    expect(hasHandleField(store, 'Name')).toBe(false);
    expect(hasHandleField(store, 'Missing')).toBe(false);
  });

  it('collectRowDirtyPayload returns changed top-level writable fields only', () => {
    const original = { Id: '1', Name: 'A', Sequence: 1 };
    const draft = { Id: '1', Name: 'B', Sequence: 1 };
    const payload = collectRowDirtyPayload(original, draft, {
      Name: { id: '2', type: 'string', typeAnnotation: '' },
      Sequence: { id: '3', type: 'int', typeAnnotation: '', isReadonly: true },
    });
    expect(payload).toEqual({ Name: 'B' });
    expect(isRowDraftDirty(original, draft)).toBe(true);
  });

  it('withEditingPayload swaps payload for the active editing row id', () => {
    const row = { kind: 'record', key: '1', payload: { Id: '1', Name: 'Old' } };
    const draft = { Id: '1', Name: 'Draft' };
    const next = withEditingPayload(row, '1', draft);
    expect(next.payload).toBe(draft);
    expect(withEditingPayload(row, '2', draft)).toBe(row);
  });
});
