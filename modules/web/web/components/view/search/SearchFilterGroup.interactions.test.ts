// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';

import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import SearchFilterCondition from './SearchFilterCondition.vue';
import SearchFilterGroup from './SearchFilterGroup.vue';

describe('SearchFilterGroup interactions', () => {
  afterEach(() => {
    restoreSfc(SearchFilterCondition);
    restoreSfc(ChoyButton as any);
  });

  test('emits logic / add / remove via header controls and nests groups', async () => {
    stubSfc(SearchFilterCondition, {
      name: 'SearchFilterCondition',
      setup() {
        return () => h('div', { class: 'cond' });
      },
    });
    stubSfc(ChoyButton as any, {
      name: 'ChoyButton',
      inheritAttrs: false,
      props: {
        variant: { type: String, default: 'default' },
        size: { type: String, default: 'default' },
        disabled: { type: Boolean, default: false },
        type: { type: String, default: 'button' },
        class: { default: undefined },
      },
      emits: ['click'],
      setup(props: any, { slots, emit, attrs }: any) {
        return () =>
          h(
            'button',
            {
              type: props.type || 'button',
              ...attrs,
              class: ['btn', props.class, attrs.class],
              disabled: props.disabled || undefined,
              onClick: (e: Event) => emit('click', e),
            },
            slots.default?.(),
          );
      },
    });

    const onSetLogic = fnRecorder();
    const onAddGroup = fnRecorder();
    const onRemoveGroup = fnRecorder();
    const onAddCondition = fnRecorder();
    const store = { fieldsMetadata: { Name: { type: 'varchar' } } } as any;

    const { unmount, qa, click } = mountApp(SearchFilterGroup as any, {
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
        onUpdateCondition: () => {},
        onRemoveCondition: () => {},
      },
    });

    await flushPromises();

    const radios = qa('.rg')[0] ? Array.from(qa('.rg')[0]!.querySelectorAll('.radio')) : [];
    expect(radios.map(r => r.getAttribute('data-value'))).toEqual(['And', 'Or']);

    expect(qa('.cond').length).toBeGreaterThan(0);
    expect(qa('.osf-group--or').length).toBeGreaterThan(0);

    click('.to-or');
    expect(onSetLogic.calls[0]).toEqual(['Or', 'g1']);

    const rootOps = qa('.osf-group__ops')[0] as HTMLElement;
    const buttons = Array.from(rootOps.querySelectorAll('button.btn'));
    expect(buttons.length).toBe(3);
    (buttons[0] as HTMLElement).click();
    expect(onAddCondition.calls[0]).toEqual(['g1']);
    (buttons[1] as HTMLElement).click();
    expect(onAddGroup.calls[0]).toEqual(['g1']);
    (buttons[2] as HTMLElement).click();
    expect(onRemoveGroup.calls[0]).toEqual(['g1']);
    unmount();
  });
});
