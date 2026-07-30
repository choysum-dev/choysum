// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';
import {
  cloneRowDraft,
  collectRowDirtyPayload,
  getDraftField,
  hasHandleField,
  isListRecordRow,
  isNumericHandleField,
  isRowDraftDirty,
  listRecordId,
  renumberSequence,
  setDraftField,
  unwrapListRecord,
  withEditingPayload,
} from '@/web/web/composables/listRowEdit';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import * as diff from '@/core/utils/diff';

describe('listRowEdit helpers', () => {
  it('unwrapListRecord covers wrappers and passthrough', () => {
    expect(unwrapListRecord(null)).toBeNull();
    expect(unwrapListRecord(undefined)).toBeUndefined();
    const payload = { Id: '1', Name: 'A' };
    expect(unwrapListRecord({ kind: 'record', payload })).toEqual(payload);
    expect(unwrapListRecord({ type: 'record', record: payload })).toEqual(payload);
    expect(unwrapListRecord({ payload })).toEqual(payload);
    expect(unwrapListRecord(payload)).toEqual(payload);
  });

  it('isListRecordRow rejects invalid and accepts record shapes', () => {
    expect(isListRecordRow(null)).toBe(false);
    expect(isListRecordRow('x')).toBe(false);
    expect(isListRecordRow({ kind: 'group' })).toBe(false);
    expect(isListRecordRow({ kind: 'more' })).toBe(false);
    expect(isListRecordRow({ kind: 'record', payload: { Id: '1' } })).toBe(true);
    expect(isListRecordRow({ type: 'record', record: { id: '2' } })).toBe(true);
    expect(isListRecordRow({ Id: '3' })).toBe(true);
    expect(isListRecordRow({ Name: 'no-id' })).toBe(false);
  });

  it('listRecordId reads Id/id and returns empty when missing', () => {
    expect(listRecordId({ Id: 7 })).toBe('7');
    expect(listRecordId({ kind: 'record', payload: { id: 'x' } })).toBe('x');
    expect(listRecordId({ Name: 'n' })).toBe('');
    expect(listRecordId(null)).toBe('');
  });

  it('cloneRowDraft deep-clones plain row data', () => {
    const src = { Id: '1', Name: 'A', nested: { x: 1 } };
    const draft = cloneRowDraft(src);
    expect(draft).toEqual(src);
    expect(draft).not.toBe(src);
    draft.nested.x = 2;
    expect(src.nested.x).toBe(1);
  });

  it('isNumericHandleField accepts numeric types only', () => {
    expect(isNumericHandleField(undefined)).toBe(false);
    expect(isNumericHandleField({ id: '1', type: '' } as any)).toBe(false);
    expect(isNumericHandleField({ id: '1', type: undefined } as any)).toBe(false);
    expect(isNumericHandleField({ id: '1', type: 'int', typeAnnotation: '' } as any)).toBe(true);
    expect(isNumericHandleField({ id: '1', type: 'Integer', typeAnnotation: '' } as any)).toBe(true);
    expect(isNumericHandleField({ id: '1', type: 'number', typeAnnotation: '' } as any)).toBe(true);
    expect(isNumericHandleField({ id: '1', type: 'decimal', typeAnnotation: '' } as any)).toBe(true);
    expect(isNumericHandleField({ id: '1', type: 'float', typeAnnotation: '' } as any)).toBe(true);
    expect(isNumericHandleField({ id: '1', type: 'varchar', typeAnnotation: '' } as any)).toBe(false);
  });

  it('hasHandleField requires store, field, writable numeric meta', () => {
    expect(hasHandleField(undefined)).toBe(false);
    expect(hasHandleField({ fieldsMetadata: {} } as any, '')).toBe(false);
    expect(hasHandleField({ fieldsMetadata: null } as any, 'Sequence')).toBe(false);
    expect(hasHandleField({} as any, 'Sequence')).toBe(false);
    const store = {
      fieldsMetadata: {
        Sequence: { id: '1', type: 'int', typeAnnotation: '', isReadonly: false },
        Locked: { id: '2', type: 'int', typeAnnotation: '', isReadonly: true },
        Name: { id: '3', type: 'string', typeAnnotation: '' },
      },
    } as unknown as WebModelStore<any>;
    expect(hasHandleField(store, 'Sequence')).toBe(true);
    expect(hasHandleField(store, 'Locked')).toBe(false);
    expect(hasHandleField(store, 'Name')).toBe(false);
    expect(hasHandleField(store, 'Missing')).toBe(false);
  });

  it('renumberSequence writes sequences and skips unchanged', () => {
    const rows = [{ Sequence: 10 }, { Sequence: 20 }, { Sequence: 30 }];
    expect(renumberSequence(rows, 'Sequence')).toHaveLength(3);
    expect(rows.map(r => r.Sequence)).toEqual([1, 2, 3]);
    expect(renumberSequence(rows, 'Sequence')).toHaveLength(0);

    const page = [{ Sequence: 1 }, { Sequence: 2 }];
    renumberSequence(page, 'Sequence', 21);
    expect(page.map(r => r.Sequence)).toEqual([21, 22]);

    const badStart = [{ Sequence: 5 }];
    renumberSequence(badStart, 'Sequence', Number.NaN);
    expect(badStart[0].Sequence).toBe(1);
    renumberSequence(badStart, 'Sequence', 0);
    expect(badStart[0].Sequence).toBe(1);
  });

  it('collectRowDirtyPayload skips dotted paths and readonly fields', () => {
    const original = { Id: '1', Name: 'A', Sequence: 1, Nested: { x: 1 } };
    const draft = { Id: '1', Name: 'B', Sequence: 2, Nested: { x: 1 } };
    const payload = collectRowDirtyPayload(original, draft, {
      Name: { id: '2', type: 'string', typeAnnotation: '' },
      Sequence: { id: '3', type: 'int', typeAnnotation: '', isReadonly: true },
    });
    expect(payload).toEqual({ Name: 'B' });
    expect(isRowDraftDirty(original, draft)).toBe(true);
    expect(isRowDraftDirty(null, draft)).toBe(false);
    expect(isRowDraftDirty(original, null)).toBe(false);
  });

  it('collectRowDirtyPayload skips empty paths from collectChangedPaths', () => {
    const spy = vi.spyOn(diff, 'collectChangedPaths').mockReturnValue(new Set(['', 'Name']));
    const payload = collectRowDirtyPayload({ Name: 'A' }, { Name: 'B' }, {
      Name: { id: '1', type: 'string', typeAnnotation: '' },
    });
    expect(payload).toEqual({ Name: 'B' });
    spy.mockRestore();
  });

  it('withEditingPayload swaps kind/type/plain rows', () => {
    const draft = { Id: '1', Name: 'Draft' };
    expect(withEditingPayload({ kind: 'record', payload: { Id: '1' } }, null, draft)).toEqual({
      kind: 'record',
      payload: { Id: '1' },
    });
    expect(withEditingPayload({ kind: 'record', payload: { Id: '1' } }, '1', null)).toEqual({
      kind: 'record',
      payload: { Id: '1' },
    });
    expect(withEditingPayload({ kind: 'record', key: '1', payload: { Id: '1', Name: 'Old' } }, '1', draft).payload).toBe(
      draft
    );
    expect(withEditingPayload({ type: 'record', record: { Id: '1' } }, '1', draft).record).toBe(draft);
    expect(withEditingPayload({ Id: '1', Name: 'Old' }, '1', draft)).toBe(draft);
    expect(withEditingPayload({ kind: 'record', payload: { Id: '1' } }, '2', draft).payload.Id).toBe('1');
  });

  it('getDraftField and setDraftField handle nested paths', () => {
    expect(getDraftField(null, 'a.b')).toBeNull();
    const draft: any = {};
    setDraftField(null, 'a', 1);
    setDraftField(draft, 'a.b.c', 3);
    expect(getDraftField(draft, 'a.b.c')).toBe(3);
    setDraftField(draft, 'x', 9);
    expect(draft.x).toBe(9);
    draft.a = null;
    setDraftField(draft, 'a.b', 1);
    expect(draft.a.b).toBe(1);
  });

  it('setDraftField and collectRowDirtyPayload reject prototype-pollution keys', () => {
    const draft: any = { Name: 'A' };
    setDraftField(draft, '__proto__.polluted', true);
    setDraftField(draft, 'constructor.prototype.x', 1);
    expect(({} as any).polluted).toBeUndefined();
    expect(draft.Name).toBe('A');
    expect(Object.prototype.hasOwnProperty.call(draft, '__proto__')).toBe(false);

    const spy = vi.spyOn(diff, 'collectChangedPaths').mockReturnValue(new Set(['__proto__', 'Name']));
    const payload = collectRowDirtyPayload({ Name: 'A' }, { Name: 'B', __proto__: { x: 1 } } as any);
    expect(payload).toEqual({ Name: 'B' });
    expect(Object.getPrototypeOf(payload)).toBe(Object.prototype);
    spy.mockRestore();
  });
});
