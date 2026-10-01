// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { h, nextTick } from 'vue';
import ChoyKanbanView from './ChoyKanbanView.vue';
import type { ChoyKanbanLane } from './kanbanViewHelpers';

function makeStore() {
  return {
    storeId: 'web.kanban.chrome.test',
    state: { queryState: {} },
  } as any;
}

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

  test('shows No cards empty lane and move-to-lane control', async () => {
    const store = makeStore();
    const mounted = mountApp(ChoyKanbanView as any, {
      props: {
        store,
        autoBootstrap: false,
        showActions: false,
      },
    });
    await flushPromises();

    const lanes: ChoyKanbanLane[] = [
      {
        key: 'todo',
        label: 'Todo',
        cards: [{ id: 'c1', title: 'Card One', laneKey: 'todo' }],
      },
      {
        key: 'done',
        label: 'Done',
        cards: [],
      },
    ];
    const state = mounted.setupState() as { displayLanes: ChoyKanbanLane[] };
    state.displayLanes = lanes;
    await nextTick();
    await flushPromises();

    expect(mounted.text()).toContain('Todo');
    expect(mounted.text()).toContain('Done');
    expect(mounted.text()).toContain('Card One');
    expect(mounted.text()).toContain('No cards');
    expect(mounted.q('[data-testid=choy-kanban-move-to-lane]')).not.toBeNull();
    expect(mounted.q('[data-testid=choy-kanban-mobile-lane]')).not.toBeNull();

    mounted.unmount();
  });
});
