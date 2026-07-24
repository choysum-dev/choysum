// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';

import OSearchFilterGroup from './OSearchFilterGroup.vue';

vi.mock('@/web/web/i18n', async () => {
  const actual = await vi.importActual<typeof import('@/web/web/i18n')>('@/web/web/i18n');
  return {
    ...actual,
    createTranslate: () => ({ _t: (msg: string) => msg }),
  };
});

describe('OSearchFilterGroup interactions', () => {
  it('emits logic / add / remove via header controls and nests groups', async () => {
    const onSetLogic = vi.fn();
    const onAddGroup = vi.fn();
    const onRemoveGroup = vi.fn();
    const onAddCondition = vi.fn();
    const store = { fieldsMetadata: { Name: { type: 'varchar' } } } as any;

    const wrapper = mount(OSearchFilterGroup as any, {
      props: {
        group: {
          id: 'g1',
          tempId: 'g1',
          logic: 'And',
          children: [
            { id: 'c1', field: 'Name', operator: '=', value: 'a' },
            {
              id: 'g2',
              tempId: 'g2',
              logic: 'Or',
              children: [{ id: 'c2', field: 'Name', operator: '!=', value: 'b' }],
            },
          ],
        },
        isRoot: false,
        fields: [{ prop: 'Name', label: 'Name' }],
        store,
        onSetLogic,
        onAddGroup,
        onRemoveGroup,
        onAddCondition,
        onUpdateCondition: vi.fn(),
        onRemoveCondition: vi.fn(),
      },
      global: {
        stubs: {
          'el-radio-group': {
            props: ['modelValue'],
            emits: ['update:modelValue'],
            template: `<div class="rg"><button class="to-or" @click="$emit('update:modelValue', 'Or')" /></div>`,
          },
          'el-radio': true,
          'el-button': {
            template: `<button class="btn" @click="$emit('click')"><slot /></button>`,
          },
          'el-divider': true,
          OSearchFilterCondition: { template: `<div class="cond" />` },
        },
      },
    });

    expect(wrapper.findAll('.cond').length).toBeGreaterThan(0);
    expect(wrapper.find('.osf-group--or').exists()).toBe(true);

    await wrapper.find('.to-or').trigger('click');
    expect(onSetLogic).toHaveBeenCalledWith('Or', 'g1');

    const buttons = wrapper.findAll('.btn');
    await buttons[0]!.trigger('click');
    expect(onAddCondition).toHaveBeenCalledWith('g1');
    await buttons[1]!.trigger('click');
    expect(onAddGroup).toHaveBeenCalledWith('g1');
    await buttons[2]!.trigger('click');
    expect(onRemoveGroup).toHaveBeenCalledWith('g1');
  });
});
