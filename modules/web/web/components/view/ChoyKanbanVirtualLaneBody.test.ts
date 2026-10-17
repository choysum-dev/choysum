// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyKanbanVirtualLaneBody from './ChoyKanbanVirtualLaneBody.vue';
import { CHOY_KANBAN_LANE_DRAG_MIME } from './kanbanVirtualHelpers';
import type { ChoyKanbanCard, ChoyKanbanLane } from './kanbanViewHelpers';

const lane: ChoyKanbanLane = { key: 'todo', label: 'Todo', cards: [] };

function card(id: string, subtitle?: string): ChoyKanbanCard {
  return { id, title: `Title ${id}`, subtitle, laneKey: 'todo' };
}

function dragStart(el: Element, withTransfer: boolean) {
  const event = new Event('dragstart', { bubbles: true, cancelable: true });
  if (withTransfer) {
    const data: Record<string, string> = {};
    Object.defineProperty(event, 'dataTransfer', {
      value: {
        effectAllowed: '',
        setData(type: string, value: string) {
          data[type] = value;
        },
        setDragImage() {},
        data,
      },
    });
  }
  Object.defineProperty(event, 'clientX', { value: 4 });
  Object.defineProperty(event, 'clientY', { value: 6 });
  el.dispatchEvent(event);
  return event as Event & { dataTransfer?: { data: Record<string, string>; effectAllowed: string } };
}

test('flat lane body clicks, drags, and labels move targets', async () => {
  const clicks: string[] = [];
  const moves: string[] = [];
  const loads: number[] = [];
  const mounted = mountApp(ChoyKanbanVirtualLaneBody as any, {
    props: {
      cards: [card('c1', 'sub'), card('c2')],
      lane,
      virtualize: false,
      laneDrag: true,
      remain: 2,
      otherLanes: [{ key: 'done', label: 'Done' }],
      moveToLabelFn: (label: string) => `Go ${label}`,
    },
    on: {
      onCardClick: (c: ChoyKanbanCard) => clicks.push(c.id),
      onLoadMore: () => loads.push(1),
      onMoveToLane: (_card: ChoyKanbanCard, to: string) => moves.push(to),
    },
  });
  await flushPromises();

  const articles = mounted.qa('article');
  expect(articles.length).toBe(2);
  mounted.click('article');
  const keydown = new Event('keydown', { bubbles: true, cancelable: true });
  Object.defineProperty(keydown, 'key', { value: 'Enter' });
  articles[0].dispatchEvent(keydown);
  expect(clicks.length).toBeGreaterThan(0);

  const dragged = dragStart(articles[0], true);
  expect(dragged.dataTransfer?.data[CHOY_KANBAN_LANE_DRAG_MIME]).toContain('c1');
  expect(dragged.dataTransfer?.effectAllowed).toBe('move');
  dragStart(articles[0], false);

  const blank = mountApp(ChoyKanbanVirtualLaneBody as any, {
    props: {
      cards: [card('   ')],
      lane: { key: ' ', label: 'Todo', cards: [] },
      virtualize: false,
      laneDrag: true,
      otherLanes: [],
    },
  });
  await flushPromises();
  const blankArticle = blank.q('article');
  expect(blankArticle).toBeTruthy();
  const prevented = dragStart(blankArticle!, true);
  expect(prevented.defaultPrevented).toBe(true);
  blank.unmount();

  expect(mounted.text()).toContain('Go Done');
  const item = mounted.q('[data-testid=choy-kanban-move-to-lane]');
  expect(item).toBeTruthy();

  mounted.unmount();
});

test('virtualized lane measures cards and uses the default move label', async () => {
  const mounted = mountApp(ChoyKanbanVirtualLaneBody as any, {
    props: {
      cards: [card('c1')],
      lane,
      virtualize: true,
      estimateCardSize: 80,
      laneDrag: true,
      otherLanes: [{ key: 'done', label: 'Done' }],
      remain: 0,
    },
  });
  await flushPromises();
  expect(mounted.q('[data-testid=choy-kanban-virtual-spacer]')).not.toBeNull();
  expect(mounted.q('[data-virtualized=true]')).not.toBeNull();
  mounted.root.measureElement(null);
  mounted.root.measureElement(document.createElement('div'));

  const labeled = mountApp(ChoyKanbanVirtualLaneBody as any, {
    props: {
      cards: [card('c1')],
      lane,
      virtualize: false,
      laneDrag: true,
      otherLanes: [{ key: 'done', label: 'Done' }],
    },
  });
  await flushPromises();
  expect(labeled.text()).toContain('Move to Done');
  labeled.unmount();
  const article = mounted.q('article');
  if (article) {
    article.dispatchEvent(new MouseEvent('click', { bubbles: true }));
  }
  mounted.unmount();
});

test('empty lane shows the empty label', async () => {
  const mounted = mountApp(ChoyKanbanVirtualLaneBody as any, {
    props: {
      cards: [],
      lane,
      virtualize: false,
      emptyLabel: 'Nothing here',
    },
  });
  await flushPromises();
  expect(mounted.text()).toContain('Nothing here');
  mounted.unmount();
});
