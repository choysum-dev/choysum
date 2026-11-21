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
  // Clear before the delay fires (no long sleep before clear — avoids CI stalls).
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

test('useDelayedVisualBusy shows immediately when delay is 0', async () => {
  const scope = effectScope();
  const busy = ref(false);
  const visual = scope.run(() => useDelayedVisualBusy(busy, 0))!;
  expect(visual.value).toBe(false);

  busy.value = true;
  await nextTick();
  expect(visual.value).toBe(true);

  scope.stop();
});

test('useDelayedVisualBusy defaults to SURFACE_BUSY_DELAY_MS', async () => {
  expect(loadingTiming.SURFACE_BUSY_DELAY_MS).toBe(loadingTiming.BAR_DELAY_MS);
  expect(loadingTiming.SURFACE_BUSY_DELAY_MS).toBeGreaterThanOrEqual(100);

  const scope = effectScope();
  const busy = ref(true);
  const visual = scope.run(() => useDelayedVisualBusy(busy))!;
  await nextTick();
  expect(visual.value).toBe(false);

  await sleep(Math.max(40, Math.floor(loadingTiming.SURFACE_BUSY_DELAY_MS / 2)));
  expect(visual.value).toBe(false);

  await sleep(loadingTiming.SURFACE_BUSY_DELAY_MS);
  expect(visual.value).toBe(true);

  scope.stop();
});

test('useDelayedVisualBusy clears pending timer on scope dispose', async () => {
  const scope = effectScope();
  const busy = ref(true);
  const visual = scope.run(() => useDelayedVisualBusy(busy, 80))!;
  await nextTick();
  expect(visual.value).toBe(false);

  scope.stop();
  await sleep(120);
  // Timer must not fire after dispose.
  expect(visual.value).toBe(false);
});

test('useDelayedVisualBusy ignores busy-getter churn when normalized value is unchanged', async () => {
  const scope = effectScope();
  const progress = ref(0);
  const ticks = ref(0);
  const visual = scope.run(() =>
    useDelayedVisualBusy(
      () => {
        ticks.value;
        return progress.value < 100;
      },
      50,
    ),
  )!;

  progress.value = 10;
  await nextTick();
  expect(visual.value).toBe(false);

  // Mid-delay churn: if the timer restarted, visual would still be false at the
  // original 50ms deadline (restart would need another full 50ms).
  await sleep(30);
  ticks.value += 1;
  await nextTick();
  await sleep(25);
  expect(visual.value).toBe(true);

  scope.stop();
});

test('useDelayedVisualBusy restarts when delayMs changes while busy', async () => {
  const scope = effectScope();
  const busy = ref(true);
  const delayMs = ref(80);
  const visual = scope.run(() => useDelayedVisualBusy(busy, delayMs))!;
  await nextTick();
  expect(visual.value).toBe(false);

  delayMs.value = 0;
  await nextTick();
  expect(visual.value).toBe(true);

  scope.stop();
});
