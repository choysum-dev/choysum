// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h, nextTick } from 'vue';

import { mountApp, stubSfc, restoreSfc } from '@/web/web/__tests__/mountApp';
import OSearchFilter from './OSearchFilter.vue';
import OSearchFilterGroup from './OSearchFilterGroup.vue';

function stubGroupEmpty() {
  stubSfc(OSearchFilterGroup, {
    name: 'OSearchFilterGroup',
    setup() {
      return () => h('div', { 'data-stub': 'OSearchFilterGroup' });
    },
  });
}

function stubGroupInteractive() {
  stubSfc(OSearchFilterGroup, {
    name: 'OSearchFilterGroup',
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
    setup(props: any) {
      return () =>
        h('div', { class: 'stub-group' }, [
          h('button', {
            class: 'logic',
            type: 'button',
            onClick: () => props.onSetLogic('Or', props.group.id),
          }),
          h('button', {
            class: 'add-c',
            type: 'button',
            onClick: () => props.onAddCondition(props.group.id),
          }),
          h('button', {
            class: 'add-g',
            type: 'button',
            onClick: () => props.onAddGroup(props.group.id),
          }),
          h('button', {
            class: 'rm-g',
            type: 'button',
            onClick: () => props.onRemoveGroup(props.group.id),
          }),
          h('button', {
            class: 'upd',
            type: 'button',
            onClick: () => props.onUpdateCondition('c1', { value: 1 }),
          }),
          h('button', {
            class: 'rm-c',
            type: 'button',
            onClick: () => props.onRemoveCondition('c1'),
          }),
        ]);
    },
  });
}

describe('OSearchFilter preview labels', () => {
  afterEach(() => {
    restoreSfc(OSearchFilterGroup);
  });

  test('renders field labels in preview for equals / in / is operators', () => {
    stubGroupEmpty();
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
    const { unmount, q } = mountApp(OSearchFilter as any, {
      props: {
        store,
        draft,
        fields: [
          { prop: 'Name', label: '名称' },
          { prop: 'PartnerId', label: '合作伙伴' },
          { prop: 'Active', label: '启用' },
        ],
      },
      stubs: {
        ElButton: true,
      },
    });
    const expr = q('.expr')?.textContent ?? '';
    expect(expr).toContain('名称');
    expect(expr).toContain('合作伙伴');
    expect(expr).toContain('启用');
    expect(expr).toContain('(incomplete)');
    expect(q('.label')?.textContent ?? '').toContain('4');
    unmount();
  });

  test('formats nested Or groups and empty roots', async () => {
    stubGroupEmpty();
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
    const m = mountApp(OSearchFilter as any, {
      reactiveProps: true,
      props: {
        store,
        draft,
        fields: [
          { prop: 'Name', label: '名称' },
          { prop: 'PartnerId', label: '合作伙伴' },
        ],
      },
      stubs: { ElButton: true },
    });
    const expr = m.q('.expr')?.textContent ?? '';
    expect(expr).toContain('名称');
    expect(expr).toContain('合作伙伴');
    expect(expr).toContain('AND');
    expect(m.q('.label')?.textContent ?? '').toContain('3');

    m.props.draft = { root: { id: 'empty', logic: 'And', children: [] } };
    await nextTick();
    expect(m.q('.expr')?.textContent ?? '').toContain('(empty)');
    m.unmount();
  });

  test('forwards footer and group events', async () => {
    stubGroupInteractive();
    const store = { fieldsMetadata: {} } as any;
    const emitted: Record<string, any[][]> = {};
    const track = (name: string) => (...args: any[]) => {
      (emitted[name] ||= []).push(args);
    };
    const { unmount, click, qa } = mountApp(OSearchFilter as any, {
      props: {
        store,
        draft: { root: { id: 'g', logic: 'And', children: [] } },
        fields: [],
      },
      on: {
        onLogicChange: track('logic-change'),
        onAddCondition: track('add-condition'),
        onAddGroup: track('add-group'),
        onRemoveGroup: track('remove-group'),
        onUpdateCondition: track('update-condition'),
        onRemoveCondition: track('remove-condition'),
        onCancel: track('cancel'),
        onConfirm: track('confirm'),
      },
      stubs: {
        ElButton: {
          name: 'ElButton',
          emits: ['click'],
          setup(_, { slots, emit }: any) {
            return () =>
              h(
                'button',
                { class: 'btn', type: 'button', onClick: () => emit('click') },
                slots.default?.()
              );
          },
        },
      },
    });
    click('.logic');
    click('.add-c');
    click('.add-g');
    click('.rm-g');
    click('.upd');
    click('.rm-c');
    const buttons = qa('.btn');
    (buttons[0] as HTMLElement).click();
    (buttons[1] as HTMLElement).click();
    expect(emitted['logic-change']?.[0]).toEqual(['Or', 'g']);
    expect(emitted['add-condition']?.[0]).toEqual(['g']);
    expect(emitted['add-group']?.[0]).toEqual(['g']);
    expect(emitted['remove-group']?.[0]).toEqual(['g']);
    expect(emitted['update-condition']?.[0]).toEqual(['c1', { value: 1 }]);
    expect(emitted['remove-condition']?.[0]).toEqual(['c1']);
    expect(emitted['cancel']).toBeTruthy();
    expect(emitted['confirm']).toBeTruthy();
    unmount();
  });
});
