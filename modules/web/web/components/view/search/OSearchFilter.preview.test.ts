// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';

import OSearchFilter from './OSearchFilter.vue';

vi.mock('@/web/web/i18n', async () => {
  const actual = await vi.importActual<typeof import('@/web/web/i18n')>('@/web/web/i18n');
  return {
    ...actual,
    createTranslate: () => ({
      _t: (msg: string, ...args: unknown[]) => {
        if (args.length) return `${msg}:${args.join(',')}`;
        return msg;
      },
    }),
  };
});

describe('OSearchFilter preview labels', () => {
  it('renders field labels in preview for equals / in / is operators', () => {
    const store = {
      fieldsMetadata: {
        Name: { type: 'varchar' },
        PartnerId: { type: 'manytoone', relationModel: 'base.Partner' },
        Active: { type: 'boolean' },
      },
    } as any;
    const draft = {
      root: {
        id: 'g1',
        logic: 'And',
        children: [
          { id: 'c1', field: 'Name', operator: '=', value: 'Alice' },
          { id: 'c2', field: 'PartnerId', operator: 'in', value: [{ Id: 'p1' }, 'p2'] },
          { id: 'c3', field: 'Active', operator: 'is', value: null },
          { id: 'c4', field: '', operator: '=', value: '' },
        ],
      },
    };
    const wrapper = mount(OSearchFilter as any, {
      props: {
        store,
        draft,
        fields: [
          { prop: 'Name', label: '名称' },
          { prop: 'PartnerId', label: '合作伙伴' },
          { prop: 'Active', label: '启用' },
        ],
      },
      global: {
        stubs: {
          OSearchFilterGroup: true,
          'el-button': true,
        },
      },
    });
    const expr = wrapper.find('.expr').text();
    expect(expr).toContain('名称');
    expect(expr).toContain('合作伙伴');
    expect(expr).toContain('启用');
    expect(expr).toContain('(incomplete)');
    expect(wrapper.find('.label').text()).toContain('4');
  });

  it('formats nested Or groups and empty roots', async () => {
    const store = { fieldsMetadata: { Name: { type: 'varchar' }, PartnerId: { type: 'manytoone' } } } as any;
    const draft = {
      root: {
        id: 'g1',
        logic: 'Or',
        children: [
          {
            id: 'g2',
            logic: 'And',
            children: [
              { id: 'c1', field: 'Name', operator: 'is not', value: null },
              { id: 'c2', field: 'Name', operator: 'not in', value: 'solo' },
              { id: 'c3', field: 'PartnerId', operator: '=', value: { Id: 'p1' } },
            ],
          },
        ],
      },
    };
    const wrapper = mount(OSearchFilter as any, {
      props: {
        store,
        draft,
        fields: [
          { prop: 'Name', label: '名称' },
          { prop: 'PartnerId', label: '合作伙伴' },
        ],
      },
      global: { stubs: { OSearchFilterGroup: true, 'el-button': true } },
    });
    const expr = wrapper.find('.expr').text();
    expect(expr).toContain('名称');
    expect(expr).toContain('合作伙伴');
    expect(expr).toContain('AND');
    expect(wrapper.find('.label').text()).toContain('3');

    await wrapper.setProps({
      draft: { root: { id: 'empty', logic: 'And', children: [] } },
    });
    expect(wrapper.find('.expr').text()).toContain('(empty)');
  });

  it('forwards footer and group events', async () => {
    const store = { fieldsMetadata: {} } as any;
    const wrapper = mount(OSearchFilter as any, {
      props: {
        store,
        draft: { root: { id: 'g', logic: 'And', children: [] } },
        fields: [],
      },
      global: {
        stubs: {
          OSearchFilterGroup: {
            props: [
              'group',
              'isRoot',
              'fields',
              'store',
              'onSetLogic',
              'onAddCondition',
              'onUpdateCondition',
              'onRemoveCondition',
              'onAddGroup',
              'onRemoveGroup',
            ],
            template: `<div class="stub-group">
              <button class="logic" @click="onSetLogic('Or', group.id)" />
              <button class="add-c" @click="onAddCondition(group.id)" />
              <button class="add-g" @click="onAddGroup(group.id)" />
              <button class="rm-g" @click="onRemoveGroup(group.id)" />
              <button class="upd" @click="onUpdateCondition('c1', { value: 1 })" />
              <button class="rm-c" @click="onRemoveCondition('c1')" />
            </div>`,
          },
          'el-button': {
            template: `<button class="btn" @click="$emit('click')"><slot /></button>`,
          },
        },
      },
    });
    await wrapper.find('.logic').trigger('click');
    await wrapper.find('.add-c').trigger('click');
    await wrapper.find('.add-g').trigger('click');
    await wrapper.find('.rm-g').trigger('click');
    await wrapper.find('.upd').trigger('click');
    await wrapper.find('.rm-c').trigger('click');
    const buttons = wrapper.findAll('.btn');
    await buttons[0]!.trigger('click');
    await buttons[1]!.trigger('click');
    expect(wrapper.emitted('logic-change')?.[0]).toEqual(['Or', 'g']);
    expect(wrapper.emitted('add-condition')?.[0]).toEqual(['g']);
    expect(wrapper.emitted('add-group')?.[0]).toEqual(['g']);
    expect(wrapper.emitted('remove-group')?.[0]).toEqual(['g']);
    expect(wrapper.emitted('update-condition')?.[0]).toEqual(['c1', { value: 1 }]);
    expect(wrapper.emitted('remove-condition')?.[0]).toEqual(['c1']);
    expect(wrapper.emitted('cancel')).toBeTruthy();
    expect(wrapper.emitted('save')).toBeTruthy();
  });
});
