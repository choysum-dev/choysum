// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { createApp, defineComponent, h } from 'vue';
import { describe, expect, it, vi } from 'vitest';

import { useFilterEditorBindings } from './useFilterEditorBindings';

/**
 * Runs a composable inside a real setup() so onUnmounted (and friends) are valid.
 */
function runInSetup<T>(fn: () => T): T {
  let result!: T;
  const app = createApp(
    defineComponent({
      setup() {
        result = fn();
        return () => h('div');
      },
    }),
  );
  app.mount(document.createElement('div'));
  app.unmount();
  return result;
}

describe('useFilterEditorBindings static meta (T4.2)', () => {
  it('metaTypeOf reads only static fieldsMetadata.type', () => {
    const FieldsGet = vi.fn(async () => ({}));
    const ensureFieldsGet = vi.fn(async () => ({}));
    const store = {
      fieldsMetadata: {
        Status: { type: 'selection' },
        Name: { type: 'varchar' },
      },
      FieldsGet,
      ensureFieldsGet,
    } as any;

    const { metaTypeOf, getOperatorOptionsForField } = runInSetup(() => useFilterEditorBindings(store));
    expect(metaTypeOf('Status')).toBe('selection');
    expect(metaTypeOf('Name')).toBe('varchar');
    expect(getOperatorOptionsForField('Status').length).toBeGreaterThan(0);
    expect(FieldsGet).not.toHaveBeenCalled();
    expect(ensureFieldsGet).not.toHaveBeenCalled();
  });

  it('source does not await FieldsGet (D2 / D13)', () => {
    const src = readFileSync(resolve(__dirname, './useFilterEditorBindings.ts'), 'utf8');
    expect(src).not.toMatch(/\bensureFieldsGet\b/);
    expect(src).not.toMatch(/\bFieldsGet\b/);
  });

  it('adds child_of/parent_of for tree Id and tree manytoone', () => {
    const store = {
      storeId: 's1',
      fieldsMetadata: {
        Id: { type: 'char' },
        ParentPath: { type: 'varchar' },
        ParentId: {
          type: 'manytoone',
          relationModel: 'base.Company',
          relationModelParentField: 'ParentId',
        },
        PartnerId: { type: 'manytoone', relationModel: 'base.Partner' },
        TagIds: { type: 'manytooneref', relationModel: 'base.Tag' },
      },
    } as any;
    const api = runInSetup(() => useFilterEditorBindings(store));
    expect(api.isTreeModel()).toBe(true);
    expect(api.metaTypeOf('')).toBe('');
    expect(api.relationModelOf('ParentId')).toBe('base.Company');
    expect(api.isTreeManyToOne('ParentId')).toBe(true);
    expect(api.isTreeManyToOne('PartnerId')).toBe(false);
    expect(api.isMultiValueOperator('in')).toBe(true);
    expect(api.isMultiValueOperator('=')).toBe(false);

    const idOps = api.getOperatorOptionsForField('Id').map(o => o.value);
    expect(idOps).toEqual(expect.arrayContaining(['child_of', 'parent_of']));
    const parentOps = api.getOperatorOptionsForField('ParentId').map(o => o.value);
    expect(parentOps).toEqual(expect.arrayContaining(['child_of', 'parent_of']));
    // Base catalog always lists child_of/parent_of; tree enrichment is for Id / tree m2o only.
    const partnerOps = api.getOperatorOptionsForField('PartnerId').map(o => o.value);
    expect(partnerOps).toEqual(expect.arrayContaining(['=', 'in']));
    expect(api.getOperatorOptionsForField().length).toBeGreaterThan(0);
    expect(api.isNullOperator('is')).toBe(true);
    expect(api.requiresValue('=')).toBe(true);
  });

  it('caches relation stores from getRelationStore and destroys on unmount', () => {
    const destroy = vi.fn();
    const rel = { destroy };
    const store = {
      storeId: 's2',
      fieldsMetadata: {
        PartnerId: { type: 'manytoone', relationModel: 'base.Partner' },
      },
      getRelationStore: vi.fn(() => rel),
    } as any;

    let api!: ReturnType<typeof useFilterEditorBindings>;
    const app = createApp(
      defineComponent({
        setup() {
          api = useFilterEditorBindings(store);
          return () => h('div');
        },
      })
    );
    app.mount(document.createElement('div'));
    const a = api.relationStoreOf('PartnerId');
    const b = api.relationStoreOf('PartnerId');
    expect(a).toBe(rel);
    expect(b).toBe(rel);
    expect(store.getRelationStore).toHaveBeenCalledTimes(1);
    expect(api.relationStoreOf()).toBeUndefined();
    app.unmount();
    expect(destroy).toHaveBeenCalled();
  });
});
