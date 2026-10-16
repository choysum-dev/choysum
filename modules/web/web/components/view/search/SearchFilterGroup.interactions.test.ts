// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h, reactive } from 'vue';

import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import ToggleGroup from '@/web/web/components/vendor/ui/toggle-group/ToggleGroup.vue';
import ToggleGroupItem from '@/web/web/components/vendor/ui/toggle-group/ToggleGroupItem.vue';
import SearchFilterCondition from './SearchFilterCondition.vue';
import SearchFilterGroup from './SearchFilterGroup.vue';

describe('SearchFilterGroup interactions', () => {
  afterEach(() => {
    restoreSfc(SearchFilterCondition);
    restoreSfc(ChoyButton as any);
    restoreSfc(ToggleGroup as any);
    restoreSfc(ToggleGroupItem as any);
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
    stubSfc(ToggleGroup as any, {
      name: 'ToggleGroup',
      props: { modelValue: { default: undefined }, class: { default: undefined } },
      emits: ['update:modelValue'],
      setup(props: any, { slots, attrs }: any) {
        return () =>
          h('div', { class: ['rg', props.class, attrs.class], 'data-slot': 'toggle-group' }, slots.default?.());
      },
    });
    stubSfc(ToggleGroupItem as any, {
      name: 'ToggleGroupItem',
      inheritAttrs: false,
      props: { value: { type: String, required: true }, class: { default: undefined } },
      setup(props: any, { slots, attrs }: any) {
        return () =>
          h(
            'button',
            {
              type: 'button',
              class: ['toggle-item', props.class, attrs.class],
              'data-value': props.value,
              'data-logic-active': attrs['data-logic-active'],
              style: attrs.style,
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

    const m = mountApp(SearchFilterGroup as any, {
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

    // Minimal DOM qa lacks descendant selectors; first .rg is the root group.
    const rootRg = m.qa('.rg')[0] as HTMLElement;
    expect(rootRg).toBeTruthy();
    const items = Array.from(rootRg.querySelectorAll('.toggle-item'));
    expect(items.map(r => r.getAttribute('data-value'))).toEqual(['And', 'Or']);
    expect(m.qa('.cond').length).toBeGreaterThan(0);
    expect(m.qa('.osf-group--or').length).toBeGreaterThan(0);

    m.setupState().onLogicToggle('Or');
    expect(onSetLogic.calls[0]).toEqual(['Or', 'g1']);

    const rootOps = m.qa('.osf-group__ops')[0] as HTMLElement;
    const buttons = Array.from(rootOps.querySelectorAll('button.btn'));
    expect(buttons.length).toBe(3);
    // Nested headers are icon-only; labels stay on aria-label / title.
    expect(buttons[0].getAttribute('aria-label')).toBe('Add condition');
    expect(buttons[1].getAttribute('aria-label')).toBe('Add group');
    expect(buttons[2].getAttribute('aria-label')).toBe('Remove group');
    (buttons[0] as HTMLElement).click();
    expect(onAddCondition.calls[0]).toEqual(['g1']);
    (buttons[1] as HTMLElement).click();
    expect(onAddGroup.calls[0]).toEqual(['g1']);
    (buttons[2] as HTMLElement).click();
    expect(onRemoveGroup.calls[0]).toEqual(['g1']);
    m.unmount();
  });

  test('joiner label and toggle active styles follow group.logic', async () => {
    stubSfc(SearchFilterCondition, {
      name: 'SearchFilterCondition',
      setup() {
        return () => h('div', { class: 'cond' });
      },
    });
    stubSfc(ChoyButton as any, {
      name: 'ChoyButton',
      setup(_: any, { slots }: any) {
        return () => h('button', { type: 'button', class: 'btn' }, slots.default?.());
      },
    });
    stubSfc(ToggleGroup as any, {
      name: 'ToggleGroup',
      props: { modelValue: { default: undefined }, class: { default: undefined } },
      setup(props: any, { slots, attrs }: any) {
        return () =>
          h(
            'div',
            {
              class: ['rg', props.class],
              'data-slot': 'toggle-group',
              'data-logic': attrs['data-logic'],
            },
            slots.default?.(),
          );
      },
    });
    stubSfc(ToggleGroupItem as any, {
      name: 'ToggleGroupItem',
      inheritAttrs: false,
      props: { value: { type: String, required: true }, class: { default: undefined } },
      setup(props: any, { slots, attrs }: any) {
        return () =>
          h(
            'button',
            {
              type: 'button',
              class: ['toggle-item', props.class],
              'data-value': props.value,
              'data-logic-active': attrs['data-logic-active'],
              style: attrs.style,
            },
            slots.default?.(),
          );
      },
    });

    const group = reactive({
      id: 'g1',
      tempId: 'g1',
      logic: 'And' as 'And' | 'Or',
      children: [
        { id: 'c1', field: 'Name', operator: '=', value: 'a' },
        { id: 'c2', field: 'Name', operator: '=', value: 'b' },
      ],
    });
    const onSetLogic = fnRecorder((logic: 'And' | 'Or') => {
      group.logic = logic;
    });

    const m = mountApp(SearchFilterGroup as any, {
      props: {
        group,
        isRoot: true,
        fields: [{ prop: 'Name', label: 'Name' }],
        store: { fieldsMetadata: { Name: { type: 'varchar' } } },
        onSetLogic,
        onAddGroup: () => {},
        onRemoveGroup: () => {},
        onAddCondition: () => {},
        onUpdateCondition: () => {},
        onRemoveCondition: () => {},
      },
    });
    await flushPromises();

    const joiner = m.q('.osf-group__joiner-label') as HTMLElement;
    expect(joiner).toBeTruthy();
    expect(joiner.getAttribute('data-logic')).toBe('And');
    expect(joiner.textContent?.trim()).toBe('AND');

    const andBtn = m.qa('.toggle-item').find(el => el.getAttribute('data-value') === 'And') as HTMLElement;
    const orBtn = m.qa('.toggle-item').find(el => el.getAttribute('data-value') === 'Or') as HTMLElement;
    expect(andBtn.getAttribute('data-logic-active')).toBe('true');
    expect(orBtn.getAttribute('data-logic-active')).toBe('false');
    expect(m.setupState().logicItemStyle('And')).toEqual({
      backgroundColor: 'var(--color-success)',
      color: 'var(--color-primary-foreground)',
    });
    expect(m.setupState().logicItemStyle('Or')).toBeUndefined();

    m.setupState().onLogicToggle('Or');
    await flushPromises();
    expect(onSetLogic.calls[0]).toEqual(['Or', 'g1']);
    expect(group.logic).toBe('Or');
    expect(joiner.getAttribute('data-logic')).toBe('Or');
    expect(joiner.textContent?.trim()).toBe('OR');
    expect(andBtn.getAttribute('data-logic-active')).toBe('false');
    expect(orBtn.getAttribute('data-logic-active')).toBe('true');
    expect(m.setupState().logicItemStyle('Or')).toEqual({
      backgroundColor: 'var(--color-warning)',
      color: 'var(--color-primary-foreground)',
    });
    expect(m.setupState().logicItemStyle('And')).toBeUndefined();

    // Same-value toggle must not emit again.
    m.setupState().onLogicToggle('Or');
    expect(onSetLogic.calls.length).toBe(1);
    m.unmount();
  });
});
