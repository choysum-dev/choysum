// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { nextTick, h, onMounted, ref } from 'vue';

import { createFieldsGetHelpers, FIELD_PRESENTATION_FIELDS_GET_ATTRS } from '@/web/web/stores/fieldsGet';
import type { WebFieldMetadata } from '@/web/web/stores/modelStore';
import { flushPromises, fnRecorder, mountApp, stub, stubSfc, restoreSfc } from '@/web/web/__tests__/mountApp';
import SearchFilterGroup from './SearchFilterGroup.vue';
import SelectionField from '@/web/web/components/field/ChoySelectionField.vue';

describe('SearchFilterGroup selection filter (T4.5)', () => {
  afterEach(() => {
    restoreSfc(SelectionField);
  });

  test('renders selection dropdown options from ensureFieldsGet', async () => {
    stubSfc(SelectionField, {
      name: 'SelectionField',
      props: ['store', 'binding'],
      setup(props: any) {
        const opts = ref<Array<{ label: string; value: string }>>([]);
        onMounted(async () => {
          const leaf = String(props.binding?.prop || 'Status');
          const res = await props.store?.ensureFieldsGet?.([leaf], [...FIELD_PRESENTATION_FIELDS_GET_ATTRS]);
          const sel = res?.[leaf]?.selection || [];
          opts.value = sel.map((s: any) => ({
            value: String(s.value ?? s[0] ?? ''),
            label: String(s.label ?? s[1] ?? s.value ?? ''),
          }));
        });
        return () =>
          h(
            'div',
            { class: 'sel-stub' },
            opts.value.map(o => h('div', { class: 'opt', 'data-label': o.label, 'data-value': o.value }))
          );
      },
    });

    const statusMeta: WebFieldMetadata = {
      id: '1',
      type: 'selection',
      typeAnnotation: 'string',
      string: 'Status',
      selection: [
        { value: 'active', label: 'Active' },
        { value: 'archived', label: 'Archived' },
      ],
    };
    const FieldsGet = fnRecorder(async () => ({
      Status: {
        ...statusMeta,
        selection: [
          { value: 'active', label: '启用' },
          { value: 'archived', label: '归档' },
        ],
      },
    }));
    const helpers = createFieldsGetHelpers(
      { fieldsMetadata: { Status: statusMeta }, FieldsGet },
      { getLang: () => 'zh_CN' }
    );
    const store = {
      fieldsMetadata: { Status: statusMeta },
      FieldsGet,
      ...helpers,
    };

    const noop = () => {};
    const { unmount, qa } = mountApp(SearchFilterGroup as any, {
      props: {
        group: {
          id: 'g1',
          tempId: 'g1',
          logic: 'And',
          children: [
            {
              id: 'c1',
              tempId: 'c1',
              field: 'Status',
              operator: '=',
              value: null,
            },
          ],
        },
        fields: [{ prop: 'Status', label: '状态' }],
        store,
        onSetLogic: noop,
        onAddGroup: noop,
        onRemoveGroup: noop,
        onAddCondition: noop,
        onUpdateCondition: noop,
        onRemoveCondition: noop,
      },
      stubs: {
        ElRadioGroup: stub('ElRadioGroup'),
        ElRadio: stub('ElRadio'),
        ElButton: stub('ElButton'),
        ElDivider: stub('ElDivider'),
        ElSelect: {
          name: 'ElSelect',
          setup(_, { slots }) {
            return () => h('div', { class: 'el-select' }, slots.default?.());
          },
        },
        ElOption: {
          name: 'ElOption',
          props: ['label', 'value'],
          setup(props: any) {
            return () => h('div', { class: 'opt', 'data-label': props.label, 'data-value': props.value });
          },
        },
        ElInput: stub('ElInput'),
        FieldBase: {
          name: 'FieldBase',
          setup(_, { slots }) {
            return () =>
              h(
                'div',
                { class: 'ob' },
                slots.edit?.({ fieldValue: () => ({ value: null }), record: {} })
              );
          },
        },
      },
    });

    await flushPromises();
    await nextTick();

    expect(FieldsGet.calls.length).toBeGreaterThan(0);
    const opts = qa('.opt').filter(o => o.getAttribute('data-value'));
    const selectionOpts = opts.filter(o => ['active', 'archived'].includes(String(o.getAttribute('data-value'))));
    expect(selectionOpts.map(o => o.getAttribute('data-label'))).toEqual(['启用', '归档']);
    unmount();
  });
});
