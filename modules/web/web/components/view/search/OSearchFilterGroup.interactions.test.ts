// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';

import { fnRecorder, mountApp, stub, stubSfc, restoreSfc } from '@/web/web/__tests__/mountApp';
import OSearchFilterGroup from './OSearchFilterGroup.vue';
import OSearchFilterCondition from './OSearchFilterCondition.vue';

describe('OSearchFilterGroup interactions', () => {
  afterEach(() => {
    restoreSfc(OSearchFilterCondition);
  });

  test('emits logic / add / remove via header controls and nests groups', async () => {
    stubSfc(OSearchFilterCondition, {
      name: 'OSearchFilterCondition',
      setup() {
        return () => h('div', { class: 'cond' });
      },
    });

    const onSetLogic = fnRecorder();
    const onAddGroup = fnRecorder();
    const onRemoveGroup = fnRecorder();
    const onAddCondition = fnRecorder();
    const store = { fieldsMetadata: { Name: { type: 'varchar' } } } as any;

    const { unmount, qa, q, click } = mountApp(OSearchFilterGroup as any, {
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
        onUpdateCondition: fnRecorder(),
        onRemoveCondition: fnRecorder(),
      },
      stubs: {
        ElRadioGroup: {
          name: 'ElRadioGroup',
          props: ['modelValue'],
          emits: ['update:modelValue'],
          setup(props: any, { slots, emit }: any) {
            return () =>
              h('div', { class: 'rg' }, [
                slots.default?.(),
                h('button', {
                  class: 'to-or',
                  type: 'button',
                  onClick: () => emit('update:modelValue', 'Or'),
                }),
              ]);
          },
        },
        ElRadio: {
          name: 'ElRadio',
          props: ['value', 'label'],
          setup(props: any) {
            return () => h('div', { class: 'radio', 'data-value': props.value ?? props.label });
          },
        },
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
        ElDivider: stub('ElDivider'),
      },
    });

    const firstRg = qa('.rg')[0]!;
    expect(
      Array.from(firstRg.querySelectorAll('.radio')).map(r => r.getAttribute('data-value'))
    ).toEqual(['And', 'Or']);

    expect(qa('.cond').length).toBeGreaterThan(0);
    expect(q('.osf-group--or')).toBeTruthy();

    click('.to-or');
    expect(onSetLogic.calls[0]).toEqual(['Or', 'g1']);

    const buttons = qa('.btn');
    (buttons[0] as HTMLElement).click();
    expect(onAddCondition.calls[0]).toEqual(['g1']);
    (buttons[1] as HTMLElement).click();
    expect(onAddGroup.calls[0]).toEqual(['g1']);
    (buttons[2] as HTMLElement).click();
    expect(onRemoveGroup.calls[0]).toEqual(['g1']);

    unmount();
  });
});
