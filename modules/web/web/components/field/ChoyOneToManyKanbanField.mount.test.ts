// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, h, nextTick, ref } from 'vue';

import type { UseField } from '@/web/web/composables/useField';
import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import ChoyFieldBase from './ChoyFieldBase.vue';
import ChoyOneToManyKanbanField from './ChoyOneToManyKanbanField.vue';
import ChoyViewScope from '@/web/web/components/view/ChoyViewScope.vue';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import { ChoyDialog, ChoyDialogContent, ChoyDialogTitle } from '@/web/web/components/layout/choyDialog';

function makeBinding(opts: { items?: any[]; isEditMode?: boolean; isForm?: boolean }) {
  const items = ref(opts.items ?? [{ Id: '1', Name: 'A', Email: 'a@x' }]);
  const fieldValue = ref(items.value.slice());
  const registerFields = fnRecorder();
  const binding: UseField = {
    env: {
      isForm: opts.isForm ?? true,
      isEditMode: opts.isEditMode ?? true,
      viewMode: opts.isEditMode === false ? 'display' : 'edit',
      fieldPrefix: null,
    },
    prop: 'Lines',
    meta: { type: 'oneToMany', typeAnnotation: '' } as any,
    fieldRef: () => fieldValue as any,
    fieldRefOf: () => fieldValue as any,
    recordRef: () => computed(() => ({ Id: 'parent' })) as any,
    registerFields,
    relationStore: { fieldsMetadata: {} } as any,
    asMutableArray: () => ({
      getItems: () => items.value,
      insertItem: fnRecorder(),
      removeItemAt: fnRecorder(),
    }),
    store: undefined,
    asView: () => ({ fieldValue: () => fieldValue }) as any,
  } as any;
  return { binding, items, fieldValue, registerFields };
}

function installStubs(mode: 'edit' | 'display') {
  stubSfc(ChoyFieldBase as any, {
    name: 'ChoyFieldBase',
    props: { binding: { type: Object, required: true } },
    setup(_: any, { slots }: any) {
      return () =>
        h('div', { class: 'field-base-stub' }, [
          mode === 'edit' ? slots.edit?.({}) : slots.display?.({}),
        ]);
    },
  });
  stubSfc(ChoyViewScope as any, {
    name: 'ChoyViewScope',
    setup(_: any, { slots }: any) {
      return () => h('div', { class: 'view-scope-stub' }, slots.default?.());
    },
  });
  stubSfc(ChoyButton as any, {
    name: 'ChoyButton',
    emits: ['click'],
    setup(_: any, { slots, attrs, emit }: any) {
      return () =>
        h(
          'button',
          {
            ...attrs,
            type: 'button',
            onClick: (e: MouseEvent) => emit('click', e),
          },
          slots.default?.()
        );
    },
  });
  for (const Comp of [ChoyDialog, ChoyDialogContent, ChoyDialogTitle] as any[]) {
    stubSfc(Comp, {
      name: Comp.name || 'DialogStub',
      setup(_: any, { slots }: any) {
        return () => h('div', { class: 'dialog-stub' }, slots.default?.());
      },
    });
  }
}

describe('ChoyOneToManyKanbanField mount coverage', () => {
  afterEach(() => {
    restoreSfc(ChoyFieldBase as any);
    restoreSfc(ChoyViewScope as any);
    restoreSfc(ChoyButton as any);
    restoreSfc(ChoyDialog as any);
    restoreSfc(ChoyDialogContent as any);
    restoreSfc(ChoyDialogTitle as any);
  });

  test('long edit board marks virtualized when card count hits auto threshold', async () => {
    installStubs('edit');
    const items = Array.from({ length: 40 }, (_, i) => ({
      Id: String(i + 1),
      Name: `Card ${i + 1}`,
    }));
    const { binding } = makeBinding({ items });
    const m = mountApp(ChoyOneToManyKanbanField as any, {
      props: { binding, cardTitleField: 'Name', cardSubtitleField: '' },
    });
    await nextTick();
    const board = m.q('[data-testid="choy-o2m-kanban-board"]');
    expect(board).toBeTruthy();
    expect(board?.getAttribute('data-virtualized')).toBe('true');
    expect(m.q('[data-testid="choy-o2m-kanban-virtual-spacer"]')).toBeTruthy();
    expect(m.q('[data-testid="choy-o2m-kanban-add"]')).toBeTruthy();
    m.unmount();
  });

  test('edit board mounts cards and add tile without empty No-data body', async () => {
    installStubs('edit');
    const { binding } = makeBinding({ items: [{ Id: '1', Name: 'Alpha' }] });
    const m = mountApp(ChoyOneToManyKanbanField as any, {
      props: { binding, cardTitleField: 'Name', cardSubtitleField: '' },
    });
    await nextTick();
    expect(m.q('[data-testid="choy-o2m-kanban-board"]')).toBeTruthy();
    expect(m.q('[data-testid="choy-o2m-kanban-card"]')?.textContent).toContain('Alpha');
    expect(m.q('[data-testid="choy-o2m-kanban-add"]')).toBeTruthy();
    expect(m.q('[data-testid="choy-o2m-kanban-summary"]')).toBeFalsy();
    m.unmount();

    const empty = makeBinding({ items: [] });
    const em = mountApp(ChoyOneToManyKanbanField as any, { props: { binding: empty.binding } });
    await nextTick();
    expect(em.q('[data-testid="choy-o2m-kanban-add"]')).toBeTruthy();
    expect((em.q('[data-testid="choy-o2m-kanban-board"]')?.textContent || '').includes('No data')).toBe(false);
    em.unmount();
  });

  test('list-cell display shows count summary instead of board', async () => {
    installStubs('display');
    const { binding } = makeBinding({
      items: [{ Id: '1' }, { Id: '2' }],
      isEditMode: false,
      isForm: false,
    });
    const m = mountApp(ChoyOneToManyKanbanField as any, { props: { binding } });
    await nextTick();
    const summary = m.q('[data-testid="choy-o2m-kanban-summary"]');
    expect(summary).toBeTruthy();
    expect(summary?.textContent?.trim()).toBe('2 records');
    expect(m.q('[data-testid="choy-o2m-kanban-board"]')).toBeNull();
    m.unmount();
  });

  test('display empty stays blank without default No data', async () => {
    installStubs('display');
    const { binding } = makeBinding({ items: [], isEditMode: false });
    const m = mountApp(ChoyOneToManyKanbanField as any, { props: { binding } });
    await nextTick();
    const board = m.q('[data-testid="choy-o2m-kanban-board"]');
    expect(board).toBeTruthy();
    expect((board?.textContent || '').includes('No data')).toBe(false);
    m.unmount();
  });

  test('registers title, subtitle and fields for Form returnFields', async () => {
    installStubs('edit');
    const { binding, registerFields } = makeBinding({ items: [{ Id: '1', Name: 'Alpha' }] });
    const m = mountApp(ChoyOneToManyKanbanField as any, {
      props: {
        binding,
        cardTitleField: 'Name',
        cardSubtitleField: '',
        fields: ['Quantity', 'State', 'Lines.Sequence'],
      },
    });
    await nextTick();
    const registered = (registerFields.calls || []).flatMap((args: unknown[]) => {
      const first = args?.[0];
      return Array.isArray(first) ? first : first != null ? [first] : [];
    });
    expect(registered).toContain('Lines.DisplayName');
    expect(registered).toContain('Lines.Name');
    expect(registered).toContain('Lines.Quantity');
    expect(registered).toContain('Lines.State');
    expect(registered).toContain('Lines.Sequence');
    expect(registered.some((p: string) => p === 'Lines.Email')).toBe(false);
    m.unmount();
  });

  test('host card chrome keeps delete outside custom #card slot', async () => {
    installStubs('edit');
    const { binding } = makeBinding({ items: [{ Id: '1', Name: 'Alpha', Sequence: 10 }] });
    binding.relationStore = {
      fieldsMetadata: { Sequence: { type: 'int' } },
    } as any;
    const m = mountApp(ChoyOneToManyKanbanField as any, {
      props: { binding, confirmOnRemove: false },
      slots: {
        card: (slotProps: { item?: { Name?: string } }) =>
          h('div', { 'data-testid': 'custom-card-body' }, slotProps?.item?.Name || ''),
      },
    });
    await nextTick();
    const body = m.q('[data-testid="custom-card-body"]');
    expect(body).toBeTruthy();
    expect(body?.textContent || '').toContain('Alpha');
    expect(m.q('[data-testid="choy-o2m-kanban-delete"]')).toBeTruthy();
    expect(m.q('[data-testid="choy-o2m-kanban-handle"]')).toBeTruthy();
    m.unmount();

    const hidden = makeBinding({ items: [{ Id: '1', Name: 'Alpha' }] });
    const hm = mountApp(ChoyOneToManyKanbanField as any, {
      props: { binding: hidden.binding, removable: false, showHandle: false },
    });
    await nextTick();
    expect(hm.q('[data-testid="choy-o2m-kanban-delete"]')).toBeNull();
    expect(hm.q('[data-testid="choy-o2m-kanban-handle"]')).toBeNull();
    hm.unmount();
  });

  test('edit dialog footer exposes secondary delete; card chrome delete removes row', async () => {
    installStubs('edit');
    const removeItemAt = fnRecorder();
    const items = ref([{ Id: '1', Name: 'Alpha' }]);
    const fieldValue = ref(items.value.slice());
    const base = makeBinding({ items: items.value });
    const binding: UseField = {
      ...base.binding,
      fieldRef: () => fieldValue as any,
      asMutableArray: () => ({
        getItems: () => items.value,
        insertItem: fnRecorder(),
        removeItemAt,
      }),
    } as any;
    const m = mountApp(ChoyOneToManyKanbanField as any, {
      props: { binding, confirmOnRemove: false },
    });
    await nextTick();
    expect(m.q('[data-testid="choy-o2m-kanban-dialog-delete"]')).toBeNull();
    m.click('[data-testid="choy-o2m-kanban-card"]');
    await nextTick();
    await flushPromises();
    expect(m.q('[data-testid="choy-o2m-kanban-dialog-delete"]')).toBeTruthy();

    m.click('[data-testid="choy-o2m-kanban-delete"]');
    await nextTick();
    await flushPromises();
    expect(removeItemAt.calls.length).toBe(1);

    removeItemAt.mockClear();
    m.click('[data-testid="choy-o2m-kanban-card"]');
    await nextTick();
    await flushPromises();
    m.click('[data-testid="choy-o2m-kanban-dialog-delete"]');
    await nextTick();
    await flushPromises();
    expect(removeItemAt.calls.length).toBe(1);
    m.unmount();
  });

  test('kanbanCardDragClass paints overIndex ring; dragstart sets HTML5 drag image', async () => {
    installStubs('edit');
    const { binding } = makeBinding({
      items: [
        { Id: '1', Name: 'Alpha', Sequence: 1 },
        { Id: '2', Name: 'Beta', Sequence: 2 },
      ],
    });
    binding.relationStore = {
      fieldsMetadata: { Sequence: { type: 'int' } },
    } as any;
    const m = mountApp(ChoyOneToManyKanbanField as any, {
      props: { binding, cardTitleField: 'Name', cardSubtitleField: '' },
    });
    await nextTick();
    await flushPromises();

    const state = m.setupState() as {
      kanbanCardDragClass?: (index: number) => Record<string, boolean>;
      kanbanReorder?: {
        draggingIndex: { value: number | null };
        overIndex: { value: number | null };
      };
      sortableEnabled?: boolean;
    };
    expect(state.sortableEnabled).toBe(true);
    expect(state.kanbanCardDragClass?.(0)).toEqual({
      'opacity-60': false,
      'bg-accent/40 ring-1 ring-inset ring-primary/30': false,
    });

    state.kanbanReorder!.draggingIndex.value = 0;
    state.kanbanReorder!.overIndex.value = 1;
    expect(state.kanbanCardDragClass?.(0)).toEqual({
      'opacity-60': true,
      'bg-accent/40 ring-1 ring-inset ring-primary/30': false,
    });
    expect(state.kanbanCardDragClass?.(1)).toEqual({
      'opacity-60': false,
      'bg-accent/40 ring-1 ring-inset ring-primary/30': true,
    });

    const handle = m.q('[data-testid="choy-o2m-kanban-handle"]') as HTMLElement | null;
    const card = m.q('[data-testid="choy-o2m-kanban-card"]') as HTMLElement | null;
    expect(handle).toBeTruthy();
    expect(card).toBeTruthy();
    Object.defineProperty(card, 'getBoundingClientRect', {
      value: () => ({ left: 10, top: 20, width: 120, height: 80, right: 130, bottom: 100, x: 10, y: 20, toJSON: () => ({}) }),
    });
    const setData = fnRecorder();
    const setDragImage = fnRecorder();
    const onKanbanDragStart = (m.setupState() as { onKanbanDragStart?: (index: number, event: DragEvent) => void })
      .onKanbanDragStart;
    expect(typeof onKanbanDragStart).toBe('function');
    onKanbanDragStart!(0, {
      preventDefault: fnRecorder(),
      stopPropagation: fnRecorder(),
      clientX: 40,
      clientY: 50,
      currentTarget: handle,
      dataTransfer: { effectAllowed: '', setData, setDragImage },
    } as unknown as DragEvent);
    await nextTick();
    expect(setDragImage.calls.length).toBe(1);
    expect(setDragImage.calls[0]?.[0]).toBe(card);
    expect(state.kanbanReorder!.draggingIndex.value).toBe(0);

    // Non-sortable (showHandle=false) returns empty class map.
    m.unmount();
    const hidden = makeBinding({
      items: [
        { Id: '1', Name: 'Alpha', Sequence: 1 },
        { Id: '2', Name: 'Beta', Sequence: 2 },
      ],
    });
    hidden.binding.relationStore = {
      fieldsMetadata: { Sequence: { type: 'int' } },
    } as any;
    const hm = mountApp(ChoyOneToManyKanbanField as any, {
      props: { binding: hidden.binding, showHandle: false },
    });
    await nextTick();
    const hiddenState = hm.setupState() as {
      kanbanCardDragClass?: (index: number) => Record<string, boolean>;
    };
    expect(hiddenState.kanbanCardDragClass?.(0)).toEqual({});
    hm.unmount();
  });
});
