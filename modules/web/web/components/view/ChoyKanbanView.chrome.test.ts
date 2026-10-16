// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { computed, h, nextTick, shallowRef } from 'vue';
import { ChoyMessage } from '@/web/web/composables/useChoyMessage';
import { PageContextKey, type PageActionTarget } from '@/web/web/composables/usePageContext';
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

  test('ungrouped flat lane tiles cards in a grid (no column chrome)', async () => {
    const store = makeStore();
    const mounted = mountApp(ChoyKanbanView as any, {
      props: {
        store,
        autoBootstrap: false,
        showActions: false,
        readonly: true,
      },
    });
    await flushPromises();
    const state = mounted.setupState() as KanbanSetup;
    state.displayLanes = [
      {
        key: 'all',
        label: 'All',
        cards: [
          { id: 'c1', title: 'Alpha', laneKey: 'all' },
          { id: 'c2', title: 'Beta', laneKey: 'all' },
        ],
      },
    ];
    await nextTick();

    expect(mounted.q('[data-testid=choy-kanban-flat-board]')).not.toBeNull();
    expect(mounted.q('.choy-kanban-view__lane')).toBeNull();
    expect(mounted.q('[data-testid=choy-kanban-mobile-lane]')).toBeNull();
    expect(mounted.text()).toContain('Alpha');
    expect(mounted.text()).toContain('Beta');
    expect(mounted.text()).not.toContain('All');
    mounted.unmount();
  });

  test('lane header shows count badge without parentheses', async () => {
    const store = makeStore();
    const mounted = mountApp(ChoyKanbanView as any, {
      props: {
        store,
        autoBootstrap: false,
        showActions: false,
        readonly: true,
      },
    });
    await flushPromises();
    const state = mounted.setupState() as KanbanSetup;
    state.displayLanes = twoLanes();
    await nextTick();

    const counts = mounted.qa('[data-testid=choy-kanban-lane-count]');
    expect(counts.length).toBe(2);
    expect(counts[0]!.textContent?.trim()).toBe('1');
    expect(counts[1]!.textContent?.trim()).toBe('0');
    expect(mounted.q('.choy-kanban-view__card')?.className).toContain('hover:bg-accent/40');
    // Soft lane wash, no outer bordered frame (cards carry the border).
    const laneClass = mounted.q('.choy-kanban-view__lane')?.className ?? '';
    expect(laneClass).toContain('bg-muted/40');
    expect(laneClass).not.toMatch(/\bborder\b/);
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

  test('defaults to lane drag with VirtualLaneBody (no Sortable)', async () => {
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

    expect(mounted.q('[data-drag-mode=lane]')).not.toBeNull();
    expect(mounted.q('[data-testid=choy-kanban-virtual-lane-body]')).not.toBeNull();
    expect(mounted.q('.fe-stub-draggable')).toBeNull();
    expect(mounted.q('[data-testid=choy-kanban-move-to-lane]')).not.toBeNull();
    mounted.unmount();
  });

  test('readonly forces none dragMode and blocks move-to-lane', async () => {
    const moves: ChoyKanbanMove[] = [];
    const readonlyMount = mountApp(ChoyKanbanView as any, {
      props: {
        store: makeStore(),
        autoBootstrap: false,
        showActions: false,
        readonly: true,
      },
      on: {
        onCardMove: (m: ChoyKanbanMove) => moves.push(m),
      },
    });
    await flushPromises();
    const rs = readonlyMount.setupState() as KanbanSetup;
    rs.displayLanes = twoLanes();
    await nextTick();
    const dragged = { id: 'c1', title: 'Card One', laneKey: 'todo' };
    rs.moveCardToLane(dragged, 'todo', 'done');
    expect(moves.length).toBe(0);
    expect(readonlyMount.q('[data-testid=choy-kanban-move-to-lane]')).toBeNull();
    expect(readonlyMount.q('[data-testid=choy-kanban-virtual-lane-body]')).not.toBeNull();
    expect(readonlyMount.q('[data-drag-mode=none]')).not.toBeNull();
    expect(readonlyMount.q('[data-virtualized=false]')).not.toBeNull();
    expect(readonlyMount.text()).toContain('Card One');
    readonlyMount.unmount();
  });

  test('readonly long lane mounts virtual spacer; short lane does not', async () => {
    const mounted = mountApp(ChoyKanbanView as any, {
      props: {
        store: makeStore(),
        autoBootstrap: false,
        showActions: false,
        readonly: true,
      },
    });
    await flushPromises();
    const state = mounted.setupState() as KanbanSetup;
    const many = Array.from({ length: 40 }, (_, i) => ({
      id: `c${i}`,
      title: `Card ${i}`,
      laneKey: 'todo',
    }));
    state.displayLanes = [
      { key: 'todo', label: 'Todo', cards: many },
      { key: 'done', label: 'Done', cards: [{ id: 'd1', title: 'Done One', laneKey: 'done' }] },
    ];
    await nextTick();
    await flushPromises();

    const spacer = mounted.q('[data-testid=choy-kanban-virtual-spacer]');
    expect(spacer).not.toBeNull();
    expect(spacer?.className ?? '').toContain('shrink-0');
    const bodies = mounted.qa('[data-testid=choy-kanban-virtual-lane-body]');
    expect(bodies.some((el) => el.getAttribute('data-virtualized') === 'true')).toBe(true);
    expect(bodies.some((el) => el.getAttribute('data-virtualized') === 'false')).toBe(true);
    expect(mounted.q('.fe-stub-draggable')).toBeNull();
    mounted.unmount();
  });

  test('lane drop commits cross-lane move (default dragMode)', async () => {
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
    const state = mounted.setupState() as KanbanSetup & {
      onLaneDrop: (event: DragEvent, lane: ChoyKanbanLane) => void;
      onLaneDragOver: (event: DragEvent, lane: ChoyKanbanLane) => void;
      onLaneDragLeave: (event: DragEvent, lane: ChoyKanbanLane) => void;
      laneDropActiveKey: string;
      engine: { loadMorePending: { value: boolean } };
    };
    state.displayLanes = twoLanes();
    await nextTick();

    expect(mounted.q('[data-drag-mode=lane]')).not.toBeNull();
    expect(mounted.q('[data-testid=choy-kanban-virtual-lane-body]')).not.toBeNull();
    expect(mounted.q('[data-testid=choy-kanban-move-to-lane]')).not.toBeNull();

    const mime = 'application/x-choy-kanban-card';
    const payload = JSON.stringify({ cardId: 'c1', fromLaneKey: 'todo' });
    const dropLane = state.displayLanes[1]!;
    const dt = {
      types: [mime, 'text/plain'],
      dropEffect: 'none',
      getData: (type: string) => (type === mime ? payload : type === 'text/plain' ? 'c1' : ''),
    };
    state.onLaneDragOver(
      { preventDefault() {}, dataTransfer: dt } as unknown as DragEvent,
      dropLane,
    );
    expect(state.laneDropActiveKey).toBe('done');

    const laneEl = { contains: (node: unknown) => node === laneEl };
    state.onLaneDragLeave(
      { relatedTarget: laneEl, currentTarget: laneEl } as unknown as DragEvent,
      dropLane,
    );
    expect(state.laneDropActiveKey).toBe('done');
    state.onLaneDragLeave(
      { relatedTarget: null, currentTarget: laneEl } as unknown as DragEvent,
      dropLane,
    );
    expect(state.laneDropActiveKey).toBe('');

    state.engine.loadMorePending.value = true;
    state.onLaneDragOver(
      { preventDefault() {}, dataTransfer: dt } as unknown as DragEvent,
      dropLane,
    );
    expect(state.laneDropActiveKey).toBe('');
    state.engine.loadMorePending.value = false;

    state.onLaneDrop(
      { preventDefault() {}, dataTransfer: dt } as unknown as DragEvent,
      dropLane,
    );

    // Assert optimistic board state before persistMove/sync settles (fake store).
    expect(moves.length).toBe(1);
    expect(moves[0]).toEqual({
      cardId: 'c1',
      fromLaneKey: 'todo',
      toLaneKey: 'done',
      toIndex: 0,
    });
    expect(state.displayLanes[0]!.cards.length).toBe(0);
    expect(state.displayLanes[1]!.cards.map((c) => c.id)).toEqual(['c1']);
    await flushPromises();
    mounted.unmount();
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

  test('lane with remain shows load-more button and scroll sentinel', async () => {
    const store = makeStore();
    const mounted = mountApp(ChoyKanbanView as any, {
      props: {
        store,
        autoBootstrap: false,
        showActions: false,
        readonly: true,
      },
    });
    await flushPromises();
    const state = mounted.setupState() as KanbanSetup;
    state.displayLanes = twoLanes({ remain: 12 });
    await nextTick();

    expect(mounted.q('[data-testid=choy-kanban-load-more]')).not.toBeNull();
    expect(mounted.text()).toContain('12 more');
    expect(mounted.qa('[data-testid=choy-kanban-load-more-sentinel]').length).toBeGreaterThan(0);
    expect(mounted.q('[data-testid=choy-kanban-lane-count]')?.textContent?.trim()).toBe('13');
    mounted.unmount();
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
    expect(mounted.text()).toContain('3 more');

    state.onLaneLoadMore(state.displayLanes[0]!);
    await flushPromises();
    expect(payloads).toEqual([{ laneKey: 'todo' }]);

    state.onLaneLoadMore({ key: 'done', label: 'Done', cards: [], remain: 0 });
    expect(payloads.length).toBe(1);
    mounted.unmount();
  });

  test('registers an empty selection target and refresh does not reject', async () => {
    const store = makeStore();
    const holder: { target: PageActionTarget | null } = { target: null };
    const mounted = mountApp(ChoyKanbanView as any, {
      props: {
        store,
        autoBootstrap: false,
        showActions: false,
        registerActionTarget: true,
      },
      provide: {
        [PageContextKey as symbol]: {
          store: computed(() => store),
          actionTarget: shallowRef(null),
          registerActionTarget(next: PageActionTarget) {
            holder.target = next;
          },
          unregisterActionTarget() {},
          title: computed(() => ''),
          titleId: 'page-title',
          titleActionsIo: computed(() => null),
          titleHostedByView: shallowRef(false),
          claimPageTitle() {},
          releasePageTitle() {},
        },
      },
    });
    await flushPromises();
    expect(holder.target?.selectedItems).toEqual([]);
    const state = mounted.setupState() as { onRefresh: () => void };
    state.onRefresh();
    await flushPromises();
    mounted.unmount();
  });
});
