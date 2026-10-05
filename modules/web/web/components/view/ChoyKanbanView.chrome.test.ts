// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { h, nextTick } from 'vue';
import { ChoyMessage } from '@/web/web/composables/useChoyMessage';
import ChoyKanbanView from './ChoyKanbanView.vue';
import type { ChoyKanbanCard, ChoyKanbanLane, ChoyKanbanMove } from './kanbanViewHelpers';

function makeStore() {
  return {
    storeId: 'web.kanban.chrome.test',
    state: { queryState: {} },
  } as any;
}

function twoLanes(opts?: { remain?: number; subtitle?: string }): ChoyKanbanLane[] {
  return [
    {
      key: 'todo',
      label: 'Todo',
      remain: opts?.remain,
      cards: [
        {
          id: 'c1',
          title: 'Card One',
          subtitle: opts?.subtitle,
          laneKey: 'todo',
        },
      ],
    },
    {
      key: 'done',
      label: 'Done',
      cards: [],
    },
  ];
}

type KanbanSetup = {
  displayLanes: ChoyKanbanLane[];
  onDragChange: (laneKey: string, evt: any) => void;
  moveCardToLane: (card: ChoyKanbanCard, from: string, to: string) => void;
  commitMove: (move: ChoyKanbanMove) => void;
  onCardClick: (card: ChoyKanbanCard) => void;
  onCreate: () => void;
  onLaneLoadMore: (lane: ChoyKanbanLane) => void;
  onMobileLaneChange: (value: unknown) => void;
  handleMoveError: (error: unknown) => void;
  mobileLaneKey: string;
  otherLanes: (fromKey: string) => ChoyKanbanLane[];
};

describe('ChoyKanbanView chrome', () => {
  test('renders ActionTray chrome and user-actions slot', async () => {
    const store = makeStore();
    const mounted = mountApp(ChoyKanbanView as any, {
      props: {
        store,
        autoBootstrap: false,
        createLabel: 'Create card',
      },
      slots: {
        'user-actions': () => h('button', { type: 'button', 'data-test': 'user-act' }, 'Extra'),
      },
    });
    await flushPromises();

    const trays = mounted.qa('[data-testid=choy-action-tray]');
    expect(trays.length).toBe(2);
    expect(trays[0]?.getAttribute('aria-label')).toBe('System actions');
    expect(trays[1]?.getAttribute('aria-label')).toBe('User actions');
    expect(mounted.text()).toContain('Create card');
    expect(mounted.q('[data-test=user-act]')).not.toBeNull();
    expect(mounted.q('[data-anchor="choy.kanban.view-chrome"]')).not.toBeNull();
    expect(mounted.q('[data-anchor="choy.kanban-view"]')).not.toBeNull();

    mounted.unmount();
  });

  test('shows translated empty board when there are no lanes', async () => {
    const store = makeStore();
    const mounted = mountApp(ChoyKanbanView as any, {
      props: {
        store,
        autoBootstrap: false,
        showActions: false,
      },
    });
    await flushPromises();
    expect(mounted.text()).toContain('No lanes');
    mounted.unmount();
  });

  test('shows No cards empty lane, subtitle, and move-to-lane control', async () => {
    const store = makeStore();
    const mounted = mountApp(ChoyKanbanView as any, {
      props: {
        store,
        autoBootstrap: false,
        showActions: false,
      },
    });
    await flushPromises();

    const state = mounted.setupState() as KanbanSetup;
    state.displayLanes = twoLanes({ subtitle: 'Sub line' });
    await nextTick();
    await flushPromises();

    expect(mounted.text()).toContain('Todo');
    expect(mounted.text()).toContain('Done');
    expect(mounted.text()).toContain('Card One');
    expect(mounted.text()).toContain('Sub line');
    expect(mounted.text()).toContain('No cards');
    expect(mounted.q('[data-testid=choy-kanban-move-to-lane]')).not.toBeNull();
    expect(mounted.q('[data-testid=choy-kanban-mobile-lane]')).not.toBeNull();
    expect(state.otherLanes('todo').map((l) => l.key)).toEqual(['done']);

    mounted.unmount();
  });

  test('defaults create label and emits create / card-click', async () => {
    const creates: unknown[] = [];
    const clicks: ChoyKanbanCard[] = [];
    const mounted = mountApp(ChoyKanbanView as any, {
      props: {
        store: makeStore(),
        autoBootstrap: false,
      },
      on: {
        onCreate: () => creates.push(1),
        onCardClick: (c: ChoyKanbanCard) => clicks.push(c),
      },
    });
    await flushPromises();
    expect(mounted.text()).toContain('New');

    const state = mounted.setupState() as KanbanSetup;
    state.onCreate();
    const card = { id: 'c1', title: 'T', laneKey: 'todo' };
    state.onCardClick(card);
    expect(creates.length).toBe(1);
    expect(clicks).toEqual([card]);
    mounted.unmount();
  });

  test('mobile lane select updates key; clears when lanes emptied', async () => {
    const mounted = mountApp(ChoyKanbanView as any, {
      props: {
        store: makeStore(),
        autoBootstrap: false,
        showActions: false,
      },
    });
    await flushPromises();
    const state = mounted.setupState() as KanbanSetup;
    state.displayLanes = twoLanes();
    await nextTick();
    expect(state.mobileLaneKey).toBe('todo');

    state.onMobileLaneChange('done');
    expect(state.mobileLaneKey).toBe('done');
    state.onMobileLaneChange(null);
    expect(state.mobileLaneKey).toBe('done');
    state.onMobileLaneChange('');
    expect(state.mobileLaneKey).toBe('done');

    state.displayLanes = [];
    await nextTick();
    expect(state.mobileLaneKey).toBe('');
    mounted.unmount();
  });

  test('move-to-lane menu path emits card-move via commitMove', async () => {
    const moves: ChoyKanbanMove[] = [];
    const mounted = mountApp(ChoyKanbanView as any, {
      props: {
        store: makeStore(),
        autoBootstrap: false,
        showActions: false,
      },
      on: {
        onCardMove: (m: ChoyKanbanMove) => moves.push(m),
      },
    });
    await flushPromises();
    const state = mounted.setupState() as KanbanSetup;
    state.displayLanes = twoLanes();
    await nextTick();

    const card = state.displayLanes[0]!.cards[0]!;
    state.moveCardToLane(card, 'todo', 'done');
    expect(moves.length).toBe(1);
    expect(moves[0]!.fromLaneKey).toBe('todo');
    expect(moves[0]!.toLaneKey).toBe('done');
    expect(moves[0]!.cardId).toBe('c1');

    // Unknown target / no-op same-lane index should not emit again.
    const before = moves.length;
    state.moveCardToLane(card, 'done', 'missing');
    state.displayLanes = [
      { key: 'todo', label: 'Todo', cards: [] },
      { key: 'done', label: 'Done', cards: [{ id: 'c1', title: 'Card One', laneKey: 'done' }] },
    ];
    state.commitMove({
      cardId: 'c1',
      fromLaneKey: 'done',
      toLaneKey: 'done',
      toIndex: 0,
    });
    expect(moves.length).toBe(before);
    await flushPromises();
    mounted.unmount();
  });

  test('onDragChange covers removed/added/moved and readonly guard', async () => {
    const moves: ChoyKanbanMove[] = [];
    const mounted = mountApp(ChoyKanbanView as any, {
      props: {
        store: makeStore(),
        autoBootstrap: false,
        showActions: false,
      },
      on: {
        onCardMove: (m: ChoyKanbanMove) => moves.push(m),
      },
    });
    await flushPromises();
    const state = mounted.setupState() as KanbanSetup;
    state.displayLanes = twoLanes();
    await nextTick();

    const dragged = { id: 'c1', title: 'Card One', laneKey: 'todo' };
    state.onDragChange('todo', null);
    state.onDragChange('todo', { removed: { element: dragged, oldIndex: 0 } });
    state.onDragChange('done', { added: { element: dragged, newIndex: 0 } });
    await flushPromises();
    expect(moves.length).toBe(1);
    expect(moves[0]).toEqual({
      cardId: 'c1',
      fromLaneKey: 'todo',
      toLaneKey: 'done',
      toIndex: 0,
    });
    expect(dragged.laneKey).toBe('done');

    // added without matching pendingRemoval falls back to element.laneKey
    const orphan = { id: 'c2', title: 'Two', laneKey: 'todo' };
    state.displayLanes = [
      { key: 'todo', label: 'Todo', cards: [orphan] },
      { key: 'done', label: 'Done', cards: [] },
    ];
    await nextTick();
    state.onDragChange('done', { added: { element: orphan, newIndex: 0 } });
    await flushPromises();
    expect(moves[moves.length - 1]!.fromLaneKey).toBe('todo');

    // same-lane reorder
    state.onDragChange('done', {
      moved: { element: orphan, newIndex: 0, oldIndex: 0 },
    });
    await flushPromises();
    expect(moves[moves.length - 1]!.fromLaneKey).toBe('done');
    expect(moves[moves.length - 1]!.toLaneKey).toBe('done');

    mounted.unmount();

    const readonlyMount = mountApp(ChoyKanbanView as any, {
      props: {
        store: makeStore(),
        autoBootstrap: false,
        showActions: false,
        readonly: true,
      },
    });
    await flushPromises();
    const rs = readonlyMount.setupState() as KanbanSetup;
    rs.displayLanes = twoLanes();
    await nextTick();
    const before = moves.length;
    rs.onDragChange('todo', { removed: { element: dragged, oldIndex: 0 } });
    rs.moveCardToLane(dragged, 'todo', 'done');
    expect(moves.length).toBe(before);
    expect(readonlyMount.q('[data-testid=choy-kanban-move-to-lane]')).toBeNull();
    expect(readonlyMount.q('.fe-stub-draggable')).toBeNull();
    expect(readonlyMount.q('[data-testid=choy-kanban-lane-static]')).not.toBeNull();
    expect(readonlyMount.text()).toContain('Card One');
    readonlyMount.unmount();
  });

  test('handleMoveError uses onMoveError or default ChoyMessage', async () => {
    const custom: unknown[] = [];
    const withCustom = mountApp(ChoyKanbanView as any, {
      props: {
        store: makeStore(),
        autoBootstrap: false,
        showActions: false,
        onMoveError: (e: unknown) => custom.push(e),
      },
    });
    await flushPromises();
    (withCustom.setupState() as KanbanSetup).handleMoveError(new Error('boom'));
    expect(custom.length).toBe(1);
    withCustom.unmount();

    const errors: string[] = [];
    const orig = ChoyMessage.error;
    ChoyMessage.error = ((msg: string) => {
      errors.push(msg);
    }) as typeof ChoyMessage.error;
    try {
      const def = mountApp(ChoyKanbanView as any, {
        props: {
          store: makeStore(),
          autoBootstrap: false,
          showActions: false,
        },
      });
      await flushPromises();
      const ss = def.setupState() as KanbanSetup;
      ss.handleMoveError(new Error('persist failed'));
      ss.handleMoveError('not-an-error');
      expect(errors).toContain('persist failed');
      expect(errors).toContain('Failed to move card');
      def.unmount();
    } finally {
      ChoyMessage.error = orig;
    }
  });

  test('lane load-more emits when remain > 0 and skips otherwise', async () => {
    const payloads: Array<{ laneKey: string }> = [];
    const mounted = mountApp(ChoyKanbanView as any, {
      props: {
        store: makeStore(),
        autoBootstrap: false,
        showActions: false,
      },
      on: {
        onLaneLoadMore: (p: { laneKey: string }) => payloads.push(p),
      },
    });
    await flushPromises();
    const state = mounted.setupState() as KanbanSetup;
    state.displayLanes = twoLanes({ remain: 3 });
    await nextTick();
    expect(mounted.q('[data-testid=choy-kanban-load-more]')).not.toBeNull();
    expect(mounted.text()).toContain('Load more');

    state.onLaneLoadMore(state.displayLanes[0]!);
    await flushPromises();
    expect(payloads).toEqual([{ laneKey: 'todo' }]);

    state.onLaneLoadMore({ key: 'done', label: 'Done', cards: [], remain: 0 });
    expect(payloads.length).toBe(1);
    mounted.unmount();
  });
});
