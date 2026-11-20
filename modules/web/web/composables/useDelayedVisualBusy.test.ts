// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { effectScope, nextTick, ref } from 'vue';
import { loadingTiming } from '@/core/web/client/loadingRegistry';
import { useDelayedVisualBusy } from './useDelayedVisualBusy';

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

test('useDelayedVisualBusy stays false until delay elapses', async () => {
  const scope = effectScope();
  const busy = ref(false);
  const visual = scope.run(() => useDelayedVisualBusy(busy, 40))!;
  expect(visual.value).toBe(false);

  busy.value = true;
  await nextTick();
  expect(visual.value).toBe(false);

  await sleep(20);
  expect(visual.value).toBe(false);

  await sleep(40);
  expect(visual.value).toBe(true);

  scope.stop();
});

test('useDelayedVisualBusy cancels pending show when busy clears early', async () => {
  const scope = effectScope();
  const busy = ref(false);
  const visual = scope.run(() => useDelayedVisualBusy(busy, 80))!;

  busy.value = true;
  await nextTick();
  await sleep(20);
  busy.value = false;
  await nextTick();
  expect(visual.value).toBe(false);

  await sleep(100);
  expect(visual.value).toBe(false);

  scope.stop();
});

test('useDelayedVisualBusy clears immediately after a shown overlay', async () => {
  const scope = effectScope();
  const busy = ref(true);
  const visual = scope.run(() => useDelayedVisualBusy(busy, 20))!;
  await sleep(40);
  expect(visual.value).toBe(true);

  busy.value = false;
  await nextTick();
  expect(visual.value).toBe(false);

  scope.stop();
});

test('useDelayedVisualBusy defaults to SURFACE_BUSY_DELAY_MS', () => {
  expect(loadingTiming.SURFACE_BUSY_DELAY_MS).toBe(loadingTiming.BAR_DELAY_MS);
  expect(loadingTiming.SURFACE_BUSY_DELAY_MS).toBeGreaterThanOrEqual(100);
});
