// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, h, nextTick, ref } from 'vue';

import type { UseField } from '@/web/web/composables/useField';
import { createFieldsGetHelpers } from '@/web/web/stores/fieldsGet';
import type { WebFieldMetadata } from '@/web/web/stores/modelStore';
import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import ChoyFieldBase from './ChoyFieldBase.vue';
import SelectionField from './ChoySelectionField.vue';

function makeBinding(opts: {
  prop?: string;
  isEditMode?: boolean;
  meta?: WebFieldMetadata;
  store?: any;
}): UseField {
  const value = ref<string | null>('active');
  const record = ref({ Id: '1', Status: 'active' });
  return {
    env: {
      isForm: true,
      isEditMode: opts.isEditMode !== false,
      viewMode: opts.isEditMode === false ? 'display' : 'edit',
      fieldPrefix: null,
    },
    prop: opts.prop || 'Status',
    meta: opts.meta as any,
    fieldRef: () => value as any,
    fieldRefOf: () => value as any,
    recordRef: () => computed(() => record.value) as any,
    registerFields: () => {},
    store: opts.store,
    asView: () => ({ fieldValue: () => value }) as any,
  } as any;
}

function installFieldBaseStub(slot: 'edit' | 'both' = 'edit') {
  stubSfc(ChoyFieldBase as any, {
    name: 'ChoyFieldBase',
    props: ['binding'],
    setup(_: any, { slots }: any) {
      const editSlot = () =>
        slots.edit?.({ fieldValue: () => ({ value: 'active' }), record: { Id: '1' } });
      const displaySlot = () =>
        slots.display?.({ fieldValue: () => ({ value: 'active' }), record: {} });
      return () =>
        h('div', { class: 'ob' }, slot === 'both' ? [editSlot(), displaySlot()] : [editSlot()]);
    },
  });
}

function valuedOptions(setupState: () => any, record?: unknown): Array<{ value: string; label: string; disabled: boolean }> {
  const fn = setupState()?.optionsFor as ((row?: unknown) => Array<{ value: string; label: string; disabled: boolean }>) | undefined;
  if (typeof fn !== 'function') return [];
  return fn(record).filter(o => o.value !== '');
}

function isDisabled(opt: { disabled?: boolean } | null | undefined): boolean {
  return !!opt?.disabled;
}

describe('SelectionField FieldsGet wiring', () => {
  const staticMeta: WebFieldMetadata = {
    id: '1',
    type: 'selection',
    typeAnnotation: 'string',
    string: 'Status',
    selection: [
      { value: 'active', label: 'Active' },
      { value: 'archived', label: 'Archived' },
    ],
  };

  afterEach(() => {
    restoreSfc(ChoyFieldBase as any);
  });

  test('edit onMounted calls ensureFieldsGet and shows loading', async () => {
    installFieldBaseStub('edit');

    const FieldsGet = fnRecorder(
      async () =>
        ({
          Status: {
            ...staticMeta,
            string: '状态',
            selection: [
              { value: 'active', label: '启用' },
              { value: 'archived', label: '归档' },
            ],
          },
        }) as any
    );
    let resolveEnsure!: () => void;
    const gate = new Promise<void>(r => {
      resolveEnsure = r;
    });
    const helpers = createFieldsGetHelpers(
      {
        fieldsMetadata: { Status: staticMeta },
        FieldsGet: async (...args: any[]) => {
          await gate;
          return FieldsGet(...(args as []));
        },
      },
      { getLang: () => 'zh_CN' }
    );
    const ensureCalls: unknown[][] = [];
    const ensureFieldsGet = async (...args: any[]) => {
      ensureCalls.push(args);
      return helpers.ensureFieldsGet(...(args as [string[], string[]?]));
    };
    const store = {
      fieldsMetadata: { Status: staticMeta },
      FieldsGet,
      ensureFieldsGet,
      getFieldMeta: helpers.getFieldMeta,
      getFieldsGetTranslatedString: helpers.getFieldsGetTranslatedString,
      clearFieldsGetCache: helpers.clearFieldsGetCache,
    };

    const m = mountApp(SelectionField as any, {
      props: {
        binding: makeBinding({ meta: staticMeta, store, isEditMode: true }),
        renderMode: 'form',
      },
    });

    await nextTick();
    expect(ensureCalls.length).toBe(1);
    expect(ensureCalls[0]![0]).toEqual(['Status']);
    expect(m.q('.choy-selection-field')?.getAttribute('data-loading')).toBe('true');

    resolveEnsure();
    await flushPromises();
    await nextTick();
    expect(m.q('.choy-selection-field')?.getAttribute('data-loading')).toBe('false');
    expect(FieldsGet.calls.length).toBe(1);
    expect(helpers.getFieldMeta('Status')?.selection?.[0]?.label).toBe('启用');
    m.unmount();
  });

  test('two instances share one RPC via ensureFieldsGet cache', async () => {
    installFieldBaseStub('both');

    const FieldsGet = fnRecorder(async () => ({
      Status: {
        ...staticMeta,
        selection: [{ value: 'active', label: '启用' }],
      },
    }));
    const helpers = createFieldsGetHelpers(
      { fieldsMetadata: { Status: staticMeta }, FieldsGet },
      { getLang: () => 'zh_CN' }
    );
    const store = { fieldsMetadata: { Status: staticMeta }, FieldsGet, ...helpers };

    const mountOne = (isEditMode: boolean) =>
      mountApp(SelectionField as any, {
        props: {
          binding: makeBinding({ meta: staticMeta, store, isEditMode }),
          renderMode: isEditMode ? 'form' : 'table',
        },
      });

    const a = mountOne(true);
    const b = mountOne(false);
    const c = mountOne(false);
    await flushPromises();
    expect(FieldsGet.calls.length).toBe(1);
    a.unmount();
    b.unmount();
    c.unmount();
  });

  test('merges FieldsGet options with props.selection filter', async () => {
    installFieldBaseStub('edit');

    const FieldsGet = fnRecorder(async () => ({
      Status: {
        ...staticMeta,
        selection: [
          { value: 'active', label: '启用' },
          { value: 'archived', label: '归档' },
        ],
      },
    }));
    const helpers = createFieldsGetHelpers(
      { fieldsMetadata: { Status: staticMeta }, FieldsGet },
      { getLang: () => 'zh_CN' }
    );
    const store = { fieldsMetadata: { Status: staticMeta }, FieldsGet, ...helpers };

    const m = mountApp(SelectionField as any, {
      props: {
        binding: makeBinding({ meta: staticMeta, store, isEditMode: true }),
        selection: ['archived'],
        renderMode: 'form',
      },
    });

    await flushPromises();
    await nextTick();
    const opts = valuedOptions(() => m.setupState());
    expect(opts).toHaveLength(1);
    expect(opts[0]!.value).toBe('archived');
    expect(opts[0]!.label).toBe('归档');
    m.unmount();
  });

  test('dynamic selectionKind loads options via ensureFieldsGet on mount', async () => {
    installFieldBaseStub('edit');

    const dynamicMeta: WebFieldMetadata = {
      id: '2',
      type: 'selection',
      typeAnnotation: 'string',
      selectionKind: 'dynamic',
    };
    const FieldsGet = fnRecorder(async () => ({
      Status: {
        ...dynamicMeta,
        selectionKind: 'dynamic' as const,
        selection: [
          { value: 'active', label: '启用' },
          { value: 'archived', label: '归档' },
        ],
      },
    }));
    const helpers = createFieldsGetHelpers(
      { fieldsMetadata: { Status: dynamicMeta }, FieldsGet },
      { getLang: () => 'zh_CN' }
    );
    const store = { fieldsMetadata: { Status: dynamicMeta }, FieldsGet, ...helpers };

    const m = mountApp(SelectionField as any, {
      props: {
        binding: makeBinding({ meta: dynamicMeta, store, isEditMode: true }),
        renderMode: 'form',
      },
    });

    await flushPromises();
    await nextTick();
    expect(FieldsGet.calls.length).toBeGreaterThan(0);
    const opts = valuedOptions(() => m.setupState());
    expect(opts.length).toBe(2);
    expect(opts[0]!.label).toBe('启用');
    m.unmount();
  });

  test('onchange selection narrows FieldsGet baseline options', async () => {
    installFieldBaseStub('edit');

    const FieldsGet = fnRecorder(async () => ({
      Status: {
        ...staticMeta,
        selection: [
          { value: 'active', label: '启用' },
          { value: 'archived', label: '归档' },
          { value: 'draft', label: '草稿' },
        ],
      },
    }));
    const helpers = createFieldsGetHelpers(
      { fieldsMetadata: { Status: staticMeta }, FieldsGet },
      { getLang: () => 'zh_CN' }
    );
    const store = { fieldsMetadata: { Status: staticMeta }, FieldsGet, ...helpers };
    const lastOnchangeResult = ref({
      selection: [{ field: 'Status', selection: ['archived', 'draft'], disabled: ['draft'] }],
    });

    const m = mountApp(SelectionField as any, {
      props: {
        binding: makeBinding({ meta: staticMeta, store, isEditMode: true }),
        renderMode: 'form',
      },
      provide: { lastOnchangeResult },
    });

    await flushPromises();
    await nextTick();
    const opts = valuedOptions(() => m.setupState());
    expect(opts.map(o => o.value)).toEqual(['archived', 'draft']);
    expect(opts[0]!.label).toBe('归档');
    expect(isDisabled(opts[1]!)).toBe(true);
    m.unmount();
  });

  // FW12: sample real ChoyFieldBase chrome (no Base stub) + Combobox.
  test('SelectionField real-chrome mounts FieldBase and combobox trigger', async () => {
    const value = ref<string | null>('active');
    const binding = {
      env: {
        isForm: true,
        fieldPresentation: 'form',
        isEditMode: true,
        viewMode: 'edit',
        fieldPrefix: null,
      },
      prop: 'Status',
      meta: staticMeta,
      fieldRef: () => value as any,
      fieldRefOf: () => value as any,
      recordRef: () => computed(() => ({ Id: '1', Status: 'active' })) as any,
      registerFields: () => {},
      store: { fieldsMetadata: { Status: staticMeta } },
      asView: () => ({ fieldValue: () => value }) as any,
    };
    const m = mountApp(SelectionField as any, {
      props: { binding, label: 'Status', renderMode: 'form' },
    });
    await flushPromises();
    await nextTick();
    expect(m.q('[data-anchor="choy.field-base"]')).toBeTruthy();
    expect(m.q('[data-slot="combobox-trigger"]') || m.q('button')).toBeTruthy();
    m.unmount();
  });
});
