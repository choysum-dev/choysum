// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createApp, defineComponent, h } from 'vue';
import { useFilterEditorBindings } from './useFilterEditorBindings';

type CallRecorder = { calls: unknown[][] };

function fnRecorder<T = undefined, A extends unknown[] = unknown[]>(
  impl?: (...args: A) => T | Promise<T>
): CallRecorder & ((...args: A) => T | Promise<T>) {
  const rec: CallRecorder & ((...args: A) => T | Promise<T>) = Object.assign(
    (...args: A) => {
      rec.calls.push(args);
      return impl ? impl(...args) : (undefined as T);
    },
    { calls: [] as unknown[][] }
  );
  return rec;
}

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
    })
  );
  app.mount(document.createElement('div'));
  app.unmount();
  return result;
}

describe('useFilterEditorBindings static meta (T4.2)', () => {
  test('metaTypeOf reads only static fieldsMetadata.type', () => {
    const FieldsGet = fnRecorder(async () => ({}));
    const ensureFieldsGet = fnRecorder(async () => ({}));
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
    expect(FieldsGet.calls.length).toBe(0);
    expect(ensureFieldsGet.calls.length).toBe(0);
  });

  test('adds child_of/parent_of for tree Id and tree manytoone', () => {
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
    expect(idOps.includes('child_of')).toBe(true);
    expect(idOps.includes('parent_of')).toBe(true);
    const parentOps = api.getOperatorOptionsForField('ParentId').map(o => o.value);
    expect(parentOps.includes('child_of')).toBe(true);
    expect(parentOps.includes('parent_of')).toBe(true);
    // Base catalog always lists child_of/parent_of; tree enrichment is for Id / tree m2o only.
    const partnerOps = api.getOperatorOptionsForField('PartnerId').map(o => o.value);
    expect(partnerOps.includes('=')).toBe(true);
    expect(partnerOps.includes('in')).toBe(true);
    expect(api.getOperatorOptionsForField().length).toBeGreaterThan(0);
    expect(api.isNullOperator('is')).toBe(true);
    expect(api.requiresValue('=')).toBe(true);
  });

  test('caches relation stores from getRelationStore and destroys on unmount', () => {
    const destroy = fnRecorder();
    const rel = { destroy };
    const getRelationStore = fnRecorder(() => rel);
    const store = {
      storeId: 's2',
      fieldsMetadata: {
        PartnerId: { type: 'manytoone', relationModel: 'base.Partner' },
      },
      getRelationStore,
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
    expect(getRelationStore.calls.length).toBe(1);
    expect(api.relationStoreOf()).toBeUndefined();
    app.unmount();
    expect(destroy.calls.length).toBe(1);
  });
});
