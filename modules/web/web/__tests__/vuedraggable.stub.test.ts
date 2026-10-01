// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Covers FE host stub branches for `vuedraggable` (aliased in unit host bundle).
 */
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { h } from 'vue';
import DraggableStub from 'vuedraggable';

describe('vuedraggable FE stub', () => {
  test('renders item slot from list with string itemKey', async () => {
    const mounted = mountApp(DraggableStub as any, {
      props: {
        list: [{ id: 'a', title: 'A' }, { id: 'b', title: 'B' }],
        itemKey: 'id',
        disabled: true,
      },
      slots: {
        item: ({ element }: { element: { title: string } }) =>
          h('span', { 'data-test': 'item' }, element.title),
      },
    });
    await flushPromises();
    expect(mounted.qa('[data-test=item]').length).toBe(2);
    expect(mounted.text()).toContain('A');
    expect(mounted.text()).toContain('B');
    expect(mounted.q('.fe-stub-draggable')?.getAttribute('data-disabled')).toBe('true');
    mounted.unmount();
  });

  test('falls back to modelValue and function itemKey; default slot when no item slot', async () => {
    const withModel = mountApp(DraggableStub as any, {
      props: {
        modelValue: [{ key: 'x' }, null],
        itemKey: (el: { key?: string } | null) => (el ? el.key : null),
      },
      slots: {
        item: ({ element }: { element: { key?: string } | null }) =>
          h('span', { 'data-test': 'mv' }, element?.key || 'empty'),
      },
    });
    await flushPromises();
    expect(withModel.qa('[data-test=mv]').length).toBe(2);
    expect(withModel.text()).toContain('x');
    expect(withModel.text()).toContain('empty');
    withModel.unmount();

    const withNullElement = mountApp(DraggableStub as any, {
      props: {
        modelValue: [null],
        itemKey: 'id',
      },
      slots: {
        item: () => h('span', { 'data-test': 'null-el' }, 'n'),
      },
    });
    await flushPromises();
    expect(withNullElement.q('[data-test=null-el]')).not.toBeNull();
    withNullElement.unmount();

    const withDefault = mountApp(DraggableStub as any, {
      props: { list: undefined, modelValue: undefined },
      slots: {
        default: () => h('span', { 'data-test': 'def' }, 'fallback'),
      },
    });
    await flushPromises();
    expect(withDefault.q('[data-test=def]')).not.toBeNull();
    expect(withDefault.text()).toContain('fallback');
    withDefault.unmount();
  });
});
