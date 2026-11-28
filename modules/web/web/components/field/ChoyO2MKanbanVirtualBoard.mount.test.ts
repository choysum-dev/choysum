// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { nextTick } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyO2MKanbanVirtualBoard from './ChoyO2MKanbanVirtualBoard.vue';
import { O2M_KANBAN_VIRTUAL_THRESHOLD } from './o2mKanbanVirtualHelpers';

function stubMeasuredRow(index: number, height: number): HTMLElement {
  const el = document.createElement('div');
  el.setAttribute('data-index', String(index));
  Object.defineProperty(el, 'offsetHeight', { value: height, configurable: true });
  Object.defineProperty(el, 'getBoundingClientRect', {
    configurable: true,
    value: () => ({
      x: 0,
      y: index * (height + 10),
      width: 200,
      height,
      top: index * (height + 10),
      left: 0,
      right: 200,
      bottom: index * (height + 10) + height,
      toJSON: () => ({}),
    }),
  });
  return el;
}

test('virtualized board keeps gap outside measured row size', async () => {
  const items = Array.from({ length: O2M_KANBAN_VIRTUAL_THRESHOLD }, (_, i) => ({
    Id: String(i),
  }));
  const mounted = mountApp(ChoyO2MKanbanVirtualBoard as any, {
    props: {
      items,
      virtualize: true,
      estimateCardSize: 80,
      gap: 10,
      minCardWidth: 200,
      maxHeight: 480,
    },
  });
  await flushPromises();
  await nextTick();

  const board = mounted.q('[data-testid=choy-o2m-kanban-board]');
  expect(board?.getAttribute('data-virtualized')).toBe('true');
  expect(mounted.q('[data-testid=choy-o2m-kanban-virtual-spacer]')).toBeTruthy();

  // Contract: card estimate and inter-row gap are separate (matches lane body).
  // measureElement must not swallow gap — it lives on virtualizer.gap.
  expect(mounted.root.estimateSize).toBe(80);
  expect(mounted.root.gapPx).toBe(10);
  expect(mounted.root.cols).toBeGreaterThan(0);
  const rows = Math.ceil(items.length / mounted.root.cols);
  expect(rows).toBeGreaterThan(1);
  // Pre-measure: n card strides + (n-1) gaps (TanStack Virtual totalSize).
  expect(mounted.root.totalSize).toBe(80 * rows + 10 * (rows - 1));

  // Measured path: stub every row taller than the estimate; gap must still apply.
  // Connect stubs so measureElement matches TanStack's connected-node path.
  const measuredHeight = 120;
  const stubs = Array.from({ length: rows }, (_, i) => stubMeasuredRow(i, measuredHeight));
  for (const el of stubs) document.body.appendChild(el);
  try {
    for (const el of stubs) mounted.root.measureElement(el);
    await nextTick();
    expect(mounted.root.totalSize).toBe(measuredHeight * rows + 10 * (rows - 1));
  } finally {
    for (const el of stubs) el.remove();
  }

  mounted.unmount();
});
