// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, h, nextTick, ref } from 'vue';
import { ElOption, ElSelect } from 'element-plus';

import type { UseField } from '@/web/web/composables/useField';
import { createFieldsGetHelpers } from '@/web/web/stores/fieldsGet';
import type { WebFieldMetadata } from '@/web/web/stores/modelStore';
import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import OFieldBase from './OFieldBase.vue';
import OSelectionField from './OSelectionField.vue';

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

const epSfcs = [ElSelect, ElOption];

function installEpStubs() {
  stubSfc(ElSelect as any, {
    name: 'ElSelect',
    props: ['loading', 'disabled', 'modelValue'],
    setup(props: any, { slots }: any) {
      return () =>
        h(
          'div',
          {
            class: 'sel',
            'data-loading': String(!!props.loading),
            'data-disabled': String(!!props.disabled),
          },
          slots.default?.()
        );
    },
  });
  stubSfc(ElOption as any, {
    name: 'ElOption',
    props: ['label', 'value', 'disabled'],
    setup(props: any) {
      return () =>
        h('div', {
          class: 'opt',
          'data-label': props.label,
          'data-value': props.value,
          'data-disabled': props.disabled ? '1' : '0',
        });
    },
  });
}

function installFieldBaseStub(slot: 'edit' | 'both' = 'edit') {
  stubSfc(OFieldBase as any, {
    name: 'OFieldBase',
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

describe('OSelectionField FieldsGet wiring', () => {
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
    restoreSfc(OFieldBase as any);
    for (const Comp of epSfcs) restoreSfc(Comp as any);
  });

  test('edit onMounted calls ensureFieldsGet and shows loading', async () => {
    installEpStubs();
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

    const m = mountApp(OSelectionField as any, {
      props: {
        binding: makeBinding({ meta: staticMeta, store, isEditMode: true }),
        renderMode: 'form',
      },
    });

    await nextTick();
    expect(ensureCalls.length).toBe(1);
    expect(ensureCalls[0]![0]).toEqual(['Status']);
    expect(m.q('.sel')?.getAttribute('data-loading')).toBe('true');

    resolveEnsure();
    await flushPromises();
    await nextTick();
    expect(m.q('.sel')?.getAttribute('data-loading')).toBe('false');
    expect(FieldsGet.calls.length).toBe(1);
    expect(helpers.getFieldMeta('Status')?.selection?.[0]?.label).toBe('启用');
    m.unmount();
  });

  test('two instances share one RPC via ensureFieldsGet cache', async () => {
    installEpStubs();
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
      mountApp(OSelectionField as any, {
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
    installEpStubs();
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

    const m = mountApp(OSelectionField as any, {
      props: {
        binding: makeBinding({ meta: staticMeta, store, isEditMode: true }),
        selection: ['archived'],
        renderMode: 'form',
      },
    });

    await flushPromises();
    await nextTick();
    const opts = m.qa('.opt');
    expect(opts).toHaveLength(1);
    expect(opts[0]!.getAttribute('data-value')).toBe('archived');
    expect(opts[0]!.getAttribute('data-label')).toBe('归档');
    m.unmount();
  });

  test('dynamic selectionKind loads options via ensureFieldsGet on mount', async () => {
    installEpStubs();
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

    const m = mountApp(OSelectionField as any, {
      props: {
        binding: makeBinding({ meta: dynamicMeta, store, isEditMode: true }),
        renderMode: 'form',
      },
    });

    await flushPromises();
    await nextTick();
    expect(FieldsGet.calls.length).toBeGreaterThan(0);
    const opts = m.qa('.opt');
    expect(opts.length).toBe(2);
    expect(opts[0]!.getAttribute('data-label')).toBe('启用');
    m.unmount();
  });

  test('onchange selection narrows FieldsGet baseline options', async () => {
    installEpStubs();
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

    const m = mountApp(OSelectionField as any, {
      props: {
        binding: makeBinding({ meta: staticMeta, store, isEditMode: true }),
        renderMode: 'form',
      },
      provide: { lastOnchangeResult },
    });

    await flushPromises();
    await nextTick();
    const opts = m.qa('.opt');
    expect(opts.map(o => o.getAttribute('data-value'))).toEqual(['archived', 'draft']);
    expect(opts[0]!.getAttribute('data-label')).toBe('归档');
    expect(opts[1]!.getAttribute('data-disabled')).toBe('1');
    m.unmount();
  });
});
