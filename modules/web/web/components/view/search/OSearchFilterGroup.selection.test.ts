// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@vue/test-utils';
import { nextTick } from 'vue';
import { describe, expect, it, vi } from 'vitest';

import { createFieldsGetHelpers } from '@/web/web/stores/fieldsGet';
import type { WebFieldMetadata } from '@/web/web/stores/modelStore';
import OSearchFilterGroup from './OSearchFilterGroup.vue';

describe('OSearchFilterGroup selection filter (T4.5)', () => {
  it('renders selection dropdown options from ensureFieldsGet', async () => {
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
    const FieldsGet = vi.fn(async () => ({
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
    const wrapper = mount(OSearchFilterGroup as any, {
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
      global: {
        stubs: {
          'el-radio-group': true,
          'el-radio': true,
          'el-button': true,
          'el-divider': true,
          'el-select': { template: `<div class="el-select"><slot /></div>` },
          'el-option': {
            props: ['label', 'value'],
            template: `<div class="opt" :data-label="label" :data-value="value" />`,
          },
          'el-input': true,
          OFieldBase: {
            template: `<div class="ob"><slot name="edit" :fieldValue="() => ({ value: null })" :record="{}" /></div>`,
          },
        },
      },
    });

    await flushPromises();
    await nextTick();

    expect(FieldsGet).toHaveBeenCalled();
    const opts = wrapper.findAll('.opt').filter(o => o.attributes('data-value'));
    // Field/operator selects also render el-option stubs; selection values are active/archived.
    const selectionOpts = opts.filter(o => ['active', 'archived'].includes(String(o.attributes('data-value'))));
    expect(selectionOpts.map(o => o.attributes('data-label'))).toEqual(['启用', '归档']);
  });
});
