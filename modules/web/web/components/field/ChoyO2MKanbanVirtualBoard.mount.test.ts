// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { nextTick } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyO2MKanbanVirtualBoard from './ChoyO2MKanbanVirtualBoard.vue';
import { O2M_KANBAN_VIRTUAL_THRESHOLD } from './o2mKanbanVirtualHelpers';

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
  const rows = mounted.root.cols > 0 ? Math.ceil(items.length / mounted.root.cols) : items.length;
  expect(rows).toBeGreaterThan(1);
  // n card strides + (n-1) gaps (TanStack Virtual totalSize).
  expect(mounted.root.totalSize).toBe(80 * rows + 10 * (rows - 1));

  mounted.unmount();
});
