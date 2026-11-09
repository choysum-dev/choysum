// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h, nextTick, ref } from 'vue';
import type { UseField } from '@/web/web/composables/useField';
import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import ChoyFieldBase from './ChoyFieldBase.vue';
import ChoyManyToManyKanbanField from './ChoyManyToManyKanbanField.vue';
import ChoyViewScope from '@/web/web/components/view/ChoyViewScope.vue';

function makeBinding(opts: { items?: any[]; isForm?: boolean } = {}) {
  const items = ref(opts.items ?? []);
  const binding: UseField = {
    env: {
      isForm: opts.isForm ?? true,
      isEditMode: false,
      viewMode: 'display',
      fieldPrefix: null,
    },
    prop: 'Depends',
    meta: { type: 'ManyToMany', relationModel: 'meta.MetaModuleCatalog' } as any,
    fieldRef: () => ({ value: items.value }) as any,
    registerFields: () => {},
    asMutableArray: () => ({
      getItems: () => items.value,
      insertItem: () => {},
      removeItemAt: () => {},
    }),
  } as any;
  return { binding, items };
}

function installStubs() {
  stubSfc(ChoyFieldBase as any, {
    name: 'ChoyFieldBase',
    props: { binding: { type: Object, required: true } },
    setup(_: any, { slots }: any) {
      return () => h('div', { class: 'field-base-stub' }, [slots.display?.({})]);
    },
  });
  stubSfc(ChoyViewScope as any, {
    name: 'ChoyViewScope',
    setup(_: any, { slots }: any) {
      return () => h('div', { class: 'view-scope-stub' }, slots.default?.());
    },
  });
}

describe('ChoyManyToManyKanbanField mount', () => {
  afterEach(() => {
    restoreSfc(ChoyFieldBase as any);
    restoreSfc(ChoyViewScope as any);
  });

  test('renders empty board and cards for related rows', async () => {
    installStubs();
    const empty = mountApp(ChoyManyToManyKanbanField as any, {
      props: {
        binding: makeBinding({ items: [] }).binding,
        cardTitleField: 'ModuleName',
        readonly: true,
      },
    });
    await nextTick();
    await flushPromises();
    expect(empty.q('[data-testid=choy-m2m-kanban]')).not.toBeNull();
    expect(empty.q('[data-testid=choy-m2m-kanban-card]')).toBeNull();
    empty.unmount();

    const filled = mountApp(ChoyManyToManyKanbanField as any, {
      props: {
        binding: makeBinding({
          items: [
            { Id: '1', ModuleName: 'core', Summary: 'Core platform' },
            { Id: '2', ModuleName: 'base', Summary: 'Base' },
          ],
        }).binding,
        cardTitleField: 'ModuleName',
        cardSubtitleField: 'Summary',
        readonly: true,
      },
    });
    await nextTick();
    await flushPromises();
    const cards = filled.qa('[data-testid=choy-m2m-kanban-card]');
    expect(cards.length).toBe(2);
    expect(String(cards[0]?.textContent || '')).toContain('core');
    filled.unmount();
  });

  test('emits card-click with the related row', async () => {
    installStubs();
    const clicks: any[] = [];
    const m = mountApp(ChoyManyToManyKanbanField as any, {
      props: {
        binding: makeBinding({ items: [{ Id: '1', ModuleName: 'auth' }] }).binding,
        cardTitleField: 'ModuleName',
        readonly: true,
        onCardClick: (payload: unknown) => clicks.push(payload),
      },
    });
    await nextTick();
    await flushPromises();
    const card = m.q('[data-testid=choy-m2m-kanban-card]') as HTMLElement | null;
    expect(card).not.toBeNull();
    card?.click();
    await flushPromises();
    expect(clicks.length).toBe(1);
    expect(String((clicks[0] as any)?.item?.ModuleName || '')).toBe('auth');
    m.unmount();
  });
});
