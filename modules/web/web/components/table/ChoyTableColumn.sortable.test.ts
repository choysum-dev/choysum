// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h, nextTick, ref } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyFieldBase from '@/web/web/components/field/ChoyFieldBase.vue';
import ChoyTableColumn from '@/web/web/components/table/ChoyTableColumn.vue';
import { useTableProvideColumnRegistry, useTableProvideBuildContext } from '@/web/web/composables/useTable';

function mountWithColumn(columnVNode: unknown) {
  return mountApp(
    {
      setup() {
        const { columns } = useTableProvideColumnRegistry();
        useTableProvideBuildContext({
          getRows: () => [],
          baseIndex: 1,
          store: null as any,
          selectionApi: null as any,
        });
        return () =>
          h('div', [
            columnVNode as any,
            h(
              'pre',
              { class: 'col-dump' },
              JSON.stringify(
                (columns.value || []).map((c: any) => ({
                  key: c.key,
                  sortable: c.sortable,
                }))
              )
            ),
          ]);
      },
    } as any,
    {}
  );
}

test('ChoyTableColumn honors vColumnProps.sortable when top-level sortable is omitted', async () => {
  const m = mountWithColumn(
    h(ChoyTableColumn as any, {
      prop: 'Name',
      label: 'Name',
      vColumnProps: { sortable: true },
    })
  );
  await flushPromises();
  await nextTick();
  expect(m.q('.col-dump')?.textContent).toBe('[{"key":"Name","sortable":true}]');
  m.unmount();
});

test('ChoyFieldBase table mode defaults varchar columns to sortable', async () => {
  const meta = { type: 'varchar', string: 'Name', id: '1', typeAnnotation: '' };
  const binding = {
    prop: 'Name',
    meta,
    store: {
      metadata: { fields: { Name: meta } },
      getFieldMeta: (n: string) => (n === 'Name' ? meta : undefined),
    },
    env: {
      isForm: false,
      fieldPresentation: 'cell',
      isEditMode: false,
      viewMode: 'list',
      fieldPrefix: null,
    },
    fieldRef: () => ref('x'),
    fieldRefOf: () => ref('x'),
    recordRef: () => ref({ Id: '1' }),
    registerFields: () => {},
    asView: (v: any) => v,
    asMutableArray: () => [],
    emitOnchange: async () => null,
  };
  const m = mountWithColumn(
    h(ChoyFieldBase as any, { binding, renderMode: 'table' }, { display: () => h('span', 'cell') })
  );
  await flushPromises();
  await nextTick();
  expect(m.q('.col-dump')?.textContent).toBe('[{"key":"Name","sortable":true}]');
  m.unmount();
});
