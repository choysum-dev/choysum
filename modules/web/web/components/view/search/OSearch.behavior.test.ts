// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@vue/test-utils';
import { nextTick } from 'vue';
import { describe, expect, it, vi } from 'vitest';

import OSearch from './OSearch.vue';

vi.mock('@/web/web/i18n', async () => {
  const actual = await vi.importActual<typeof import('@/web/web/i18n')>('@/web/web/i18n');
  return {
    ...actual,
    createTranslate: () => ({
      _t: (msg: string, ...args: unknown[]) => (args.length ? `${msg}:${args.join(',')}` : msg),
    }),
  };
});

vi.mock('element-plus', async () => {
  const actual = await vi.importActual<any>('element-plus');
  return {
    ...actual,
    ElMessage: { warning: vi.fn() },
  };
});

function makeStore() {
  return {
    storeId: 'demo.Widget',
    fieldsMetadata: {
      Name: { id: '1', type: 'varchar', string: 'Name' },
      CreatedAt: { id: '2', type: 'datetime', string: 'Created At' },
      Status: { id: '3', type: 'selection', string: 'Status' },
    },
    state: { queryState: { defaultFilters: [{ name: 'Active', query: ['Active', '=', true] }] } },
    getFieldsGetTranslatedString: () => undefined,
  } as any;
}

const elementStubs = {
  'el-tooltip': { template: `<div><slot /></div>` },
  'el-button': {
    emits: ['click'],
    template: `<button type="button" class="el-btn" @click="$emit('click', $event)"><slot /></button>`,
  },
  'el-tag': {
    props: { closable: { type: Boolean, default: false } },
    emits: ['close', 'click'],
    template: `<span class="el-tag" @click="$emit('click', $event)">
      <slot />
      <button v-if="closable" type="button" class="tag-close" @click.stop="$emit('close', $event)" />
    </span>`,
  },
  'el-popover': {
    props: ['visible'],
    emits: ['update:visible'],
    template: `<div class="el-popover"><slot name="reference" /><div class="pop"><slot /></div></div>`,
  },
  'el-dialog': {
    props: ['modelValue'],
    emits: ['update:modelValue', 'close'],
    template: `<div v-if="modelValue" class="el-dialog"><slot /></div>`,
  },
  'el-divider': true,
  'el-icon': { template: `<i><slot /></i>` },
  'el-tree-select': {
    emits: ['change', 'update:modelValue'],
    template: `<button type="button" class="tree" @click="$emit('change', 'f:Status')" />`,
  },
  OSearchFilter: {
    props: ['store', 'draft', 'fields'],
    emits: ['cancel', 'save'],
    template: `<div class="filter-editor">
      <button type="button" class="save" @click="$emit('save')" />
      <button type="button" class="cancel" @click="$emit('cancel')" />
    </div>`,
  },
};

describe('OSearch behavior', () => {
  function mountSearch(props: Record<string, any> = {}) {
    return mount(OSearch as any, {
      props: {
        store: makeStore(),
        placeholder: 'Find…',
        defaultFilters: [{ name: 'Active', query: ['Active', '=', true] }],
        ...props,
      },
      global: { stubs: elementStubs },
    });
  }

  it('emits query-update on enter / search icon and syncs controlled keyword', async () => {
    const wrapper = mountSearch({ currentKeyword: 'hello' });
    await flushPromises();
    const input = wrapper.find('input.o-search__input');
    expect((input.element as HTMLInputElement).value).toBe('hello');

    await input.trigger('keydown.enter');
    expect(wrapper.emitted('query-update')?.length).toBeGreaterThan(0);

    const before = wrapper.emitted('query-update')!.length;
    await wrapper.findAll('.el-btn')[0]!.trigger('click');
    expect(wrapper.emitted('query-update')!.length).toBeGreaterThan(before);
  });

  it('shows grouping tag and clears grouping', async () => {
    const wrapper = mountSearch({
      currentAppliedGroups: [{ field: 'Status' }, { field: 'CreatedAt', granularity: 'month' }],
    });
    await nextTick();
    expect(wrapper.find('.o-search__grouptag').exists()).toBe(true);
    await wrapper.find('.o-search__grouptag .tag-close').trigger('click');
    const payload = wrapper.emitted('query-update')!.at(-1)![0] as any;
    expect(payload.appliedGroups).toEqual([]);
  });

  it('opens custom filter editor, warns on incomplete save, and cancels', async () => {
    const { ElMessage } = await import('element-plus');
    const wrapper = mountSearch();
    await nextTick();
    const custom = wrapper.findAll('.el-btn').find(b => b.text().includes('Custom filter'));
    expect(custom).toBeTruthy();
    await custom!.trigger('click');
    await nextTick();
    expect(wrapper.find('.filter-editor').exists()).toBe(true);

    await wrapper.find('.filter-editor .save').trigger('click');
    expect(ElMessage.warning).toHaveBeenCalled();

    await wrapper.find('.filter-editor .cancel').trigger('click');
    await nextTick();
    expect(wrapper.find('.filter-editor').exists()).toBe(false);
  });

  it('applies controlled filters and supports tag close / backspace delete', async () => {
    const filters = [
      {
        id: 'f1',
        name: 'Active',
        logic: 'And' as const,
        children: [{ id: 'c1', field: 'Active', operator: '=', value: true }],
      },
    ];
    const wrapper = mountSearch({ currentAppliedFilters: filters });
    await flushPromises();
    expect(wrapper.find('.o-search__tag').exists()).toBe(true);

    await wrapper.find('.o-search__tag').trigger('click');
    await nextTick();
    expect(wrapper.find('.filter-editor').exists()).toBe(true);
    await wrapper.find('.filter-editor .cancel').trigger('click');
    await nextTick();

    await wrapper.find('.o-search__tag .tag-close').trigger('click');
    expect(wrapper.emitted('query-update')?.length).toBeGreaterThan(0);

    // Acknowledge the clear echo, then push a new controlled snapshot.
    await wrapper.setProps({ currentAppliedFilters: [] });
    await flushPromises();
    await wrapper.setProps({
      currentAppliedFilters: [
        {
          id: 'f2',
          name: 'X',
          logic: 'And',
          children: [{ id: 'c2', field: 'Name', operator: '=', value: 'a' }],
        },
      ],
    });
    await flushPromises();
    expect(wrapper.find('.o-search__tag').text()).toContain('X');
    await wrapper.find('.o-search__tag').trigger('click');
    await nextTick();
    expect(wrapper.find('.filter-editor').exists()).toBe(true);
  });

  it('toggles default filter from menu', async () => {
    const wrapper = mountSearch();
    await nextTick();
    const item = wrapper.findAll('.el-btn').find(b => b.text().includes('Active'));
    expect(item).toBeTruthy();
    await item!.trigger('click');
    expect(wrapper.emitted('query-update')?.length).toBeGreaterThan(0);
  });

  it('applies tree select change for grouping', async () => {
    const wrapper = mountSearch();
    await nextTick();
    await wrapper.find('.tree').trigger('click');
    expect(wrapper.emitted('query-update')?.length).toBeGreaterThan(0);
    const payload = wrapper.emitted('query-update')!.at(-1)![0] as any;
    expect(payload.appliedGroups?.some((g: any) => g.field === 'Status' || g === 'Status')).toBe(true);
  });
});
