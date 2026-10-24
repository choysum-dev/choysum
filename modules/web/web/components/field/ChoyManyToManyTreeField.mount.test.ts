// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Patch coverage for checkbox-click early returns on RelationTree handlers.
 */

import { computed, defineComponent, h, ref } from 'vue';

import type { UseField } from '@/web/web/composables/useField';
import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import ChoyFieldBase from './ChoyFieldBase.vue';
import ChoyManyToManyTreeField from './ChoyManyToManyTreeField.vue';
import RelationTree from '@/web/web/components/internal/RelationTree.vue';

const RelationTreeStub = defineComponent({
  name: 'RelationTreeStub',
  inheritAttrs: false,
  setup(_: any, { slots }: any) {
    return () => h('div', { class: 'relation-tree-stub' }, slots.default?.({ node: {}, data: {} }));
  },
});

function makeBinding(opts: { items?: any[]; relationStore?: any } = {}): UseField {
  const items = ref(
    (opts.items || []).map((x: any) =>
      x && typeof x === 'object' ? { ...x } : { Id: String(x), DisplayName: String(x) }
    )
  );
  return {
    env: { isForm: true, isEditMode: true, viewMode: 'edit', fieldPrefix: null },
    prop: 'CategoryIds',
    meta: { type: 'ManyToMany', relationModel: 'demo.Category' } as any,
    fieldRef: () => items as any,
    fieldRefOf: () => items as any,
    recordRef: () => computed(() => ({ Id: 'parent' })) as any,
    registerFields: () => {},
    relationStore: opts.relationStore,
    store: undefined,
    asMutableArray: () => ({
      getItems: () => items.value,
      insertItem: (row: any) => {
        items.value = [...items.value, row];
      },
      clearItems: () => {
        items.value = [];
      },
      removeItemAt: () => {},
    }),
    asView: () => ({ fieldValue: () => items }) as any,
  } as any;
}

function relationStoreStub(overrides: Record<string, unknown> = {}) {
  return {
    fullModelName: 'demo.Category',
    storeId: 'store-tree-1',
    Search: fnRecorder(async () => [{ Id: 'c1', DisplayName: 'Cat', ParentId: null }]),
    ...overrides,
  };
}

describe('ChoyManyToManyTreeField patch coverage', () => {
  beforeEach(() => {
    stubSfc(ChoyFieldBase as any, {
      name: 'ChoyFieldBase',
      inheritAttrs: false,
      props: {
        binding: { type: Object, required: true },
        label: { type: String, default: undefined },
        rules: { type: Array, default: undefined },
        formItemProps: { type: Object, default: undefined },
        required: { type: [Boolean, Function, Object], default: undefined },
        readonly: { type: [Boolean, Function, Object], default: undefined },
        visible: { type: [Boolean, Function, Object], default: undefined },
        cellVisible: { type: [Boolean, Function, Object], default: undefined },
        renderMode: { type: String, default: undefined },
        showInlineError: { type: Boolean, default: undefined },
        preserveModeSlot: { type: Boolean, default: undefined },
      },
      setup(_p: any, { slots }: any) {
        return () => h('div', { class: 'field-base-stub' }, [slots.edit?.({}), slots.display?.({})]);
      },
    });
    stubSfc(RelationTree as any, RelationTreeStub as any);
  });

  afterEach(() => {
    restoreSfc(ChoyFieldBase as any);
    restoreSfc(RelationTree as any);
  });

  test('applySelectedIds writes ClientModel objects not bare ids', async () => {
    const items = ref([{ Id: 'c1', DisplayName: 'Cat' }]);
    const binding = {
      env: { isForm: true, isEditMode: true, viewMode: 'edit', fieldPrefix: null },
      prop: 'CategoryIds',
      meta: { type: 'ManyToMany', relationModel: 'demo.Category' },
      fieldRef: () => items as any,
      fieldRefOf: () => items as any,
      recordRef: () => computed(() => ({ Id: 'parent' })) as any,
      registerFields: () => {},
      relationStore: relationStoreStub({
        Search: fnRecorder(async () => [{ Id: 'c2', DisplayName: 'Other', Sequence: 1 }]),
      }),
      store: undefined,
      asMutableArray: () => ({
        getItems: () => items.value,
        insertItem: (row: any) => {
          items.value = [...items.value, row];
        },
        clearItems: () => {
          items.value = [];
        },
        removeItemAt: () => {},
      }),
      asView: () => ({ fieldValue: () => items }) as any,
    } as any;
    const m = mountApp(ChoyManyToManyTreeField as any, {
      props: { binding, renderMode: 'form', lazy: true },
    });
    await flushPromises();
    const ss = m.setupState() as any;
    ss.treeData = [
      {
        __id: 'c2',
        __label: 'Other',
        __order: 1,
        __depth: 1,
        raw: { Id: 'c2', DisplayName: 'Other' },
      },
    ];
    ss.applySelectedIds(['c2']);
    expect(items.value).toHaveLength(1);
    expect(items.value[0]).toMatchObject({ Id: 'c2', DisplayName: 'Other' });
    expect(typeof items.value[0]).toBe('object');
    m.unmount();
  });

  test('child preload Search passes relationConditionSource from host field', async () => {
    const Search = fnRecorder(async (_cond: unknown, opts?: any) => {
      if (opts?.fields?.includes?.('Children') || opts?.fields?.includes?.('Childs')) {
        return [{ Id: 'root', Children: [{ Id: 'child', Name: 'Child', Code: 'C' }] }];
      }
      return [{ Id: 'root', Name: 'Root', Code: 'R', Sequence: 1 }];
    });
    const hostStore = { fullModelName: 'demo.Record', storeId: 'host-1' };
    const binding = makeBinding({
      items: [],
      relationStore: relationStoreStub({ Search }),
    });
    const m = mountApp(ChoyManyToManyTreeField as any, {
      props: {
        binding,
        store: hostStore,
        renderMode: 'form',
        lazy: false,
        childrenField: 'Children',
        rootCondition: ['ParentId', 'is', null],
      },
    });
    await flushPromises();
    const withSource = Search.calls.filter((c: any[]) => c[1]?.relationConditionSource);
    expect(withSource.length).toBeGreaterThanOrEqual(1);
    expect(withSource.some((c: any[]) => c[1]?.relationConditionSource?.field === 'CategoryIds')).toBe(true);
    m.unmount();
  });

  test('checkbox closest skips expand and stops display capture', async () => {
    const binding = makeBinding({
      items: ['c1'],
      relationStore: relationStoreStub(),
    });
    const m = mountApp(ChoyManyToManyTreeField as any, {
      props: { binding, renderMode: 'form', expandOnClickNode: true },
    });
    await flushPromises();
    const ss = m.setupState() as any;

    const expand = fnRecorder();
    const collapse = fnRecorder();
    const checkboxTarget = {
      closest: (sel: string) => (sel === '.choy-relation-tree__checkbox' ? {} : null),
    } as any;
    const plainTarget = {
      closest: () => null,
    } as any;

    // Line: skip toggle when click originates from checkbox.
    ss.onNodeClick({}, { expand, collapse, expanded: false }, {}, { target: checkboxTarget });
    expect(expand.calls.length).toBe(0);

    // Non-checkbox click still expands.
    ss.onNodeClick({}, { expand, collapse, expanded: false }, {}, { target: plainTarget });
    expect(expand.calls.length).toBe(1);

    const stop = fnRecorder();
    const prevent = fnRecorder();
    // Display capture: only checkbox hits prevent/stop.
    ss.onDisplayTreeClickCapture({
      target: plainTarget,
      preventDefault: prevent,
      stopPropagation: stop,
    });
    expect(prevent.calls.length).toBe(0);

    ss.onDisplayTreeClickCapture({
      target: checkboxTarget,
      preventDefault: prevent,
      stopPropagation: stop,
    });
    expect(prevent.calls.length).toBe(1);
    expect(stop.calls.length).toBe(1);

    m.unmount();
  });
});
