// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ensureHeaders } from '../testing/qjs_polyfills';
import {
  beginNavigationLoading,
  flushLoadingDisplayForTests,
  getControlBusy,
  getLoadingRegistryDebugState,
  getSurfaceBusy,
  loadingRegistryOnRpcFinish,
  loadingRegistryOnRpcStart,
  resetLoadingRegistryForTests,
  runWithLoading,
  scheduleNavigationSettleIfIdle,
  settleNavigationIfActive,
  loadingTiming,
  useLoadingBarVisible,
  useLoadingSlowHintVisible,
  useHasActiveSurfaceBusy,
  useNavigationHandoffBusy,
} from './loadingRegistry';

ensureHeaders();

function afterEachReset(): void {
  resetLoadingRegistryForTests();
}

test('runWithLoading surface owns bar and user-busy; nested RPC does not double-count', async () => {
  afterEachReset();
  const p = runWithLoading({ layer: 'surface', target: 'list-1', mode: 'stale' }, async () => {
    expect(getSurfaceBusy('list-1')).toEqual({ busy: true, mode: 'stale' });
    expect(getLoadingRegistryDebugState().barCount).toBe(1);
    expect(getLoadingRegistryDebugState().userBusyCount).toBe(1);
    loadingRegistryOnRpcStart('rpc-1');
    expect(getLoadingRegistryDebugState().barCount).toBe(1);
    loadingRegistryOnRpcFinish('rpc-1');
    expect(getLoadingRegistryDebugState().barCount).toBe(1);
    return 1;
  });
  expect(await p).toBe(1);
  expect(getSurfaceBusy('list-1').busy).toBe(false);
  expect(getLoadingRegistryDebugState().barCount).toBe(0);
  expect(getLoadingRegistryDebugState().userBusyCount).toBe(0);
});

test('runWithLoading control counts user-busy but not bar; nested RPC silent to chrome', async () => {
  afterEachReset();
  await runWithLoading({ layer: 'control', target: 'save-1', pendingLabel: 'Saving…' }, async () => {
    expect(getControlBusy('save-1')).toEqual({ busy: true, pendingLabel: 'Saving…' });
    expect(getLoadingRegistryDebugState().barCount).toBe(0);
    expect(getLoadingRegistryDebugState().userBusyCount).toBe(1);
    loadingRegistryOnRpcStart('rpc-2');
    expect(getLoadingRegistryDebugState().barCount).toBe(0);
    loadingRegistryOnRpcFinish('rpc-2');
  });
  expect(getControlBusy('save-1').busy).toBe(false);
  expect(getLoadingRegistryDebugState().userBusyCount).toBe(0);
});

test('silent skips bar and slow-hint counters', async () => {
  afterEachReset();
  await runWithLoading({ silent: true }, async () => {
    loadingRegistryOnRpcStart('rpc-s');
    expect(getLoadingRegistryDebugState().barCount).toBe(0);
    expect(getLoadingRegistryDebugState().userBusyCount).toBe(0);
    loadingRegistryOnRpcFinish('rpc-s');
  });
});

test('unmarked RPC opens chrome bar and user-busy tickets', () => {
  afterEachReset();
  loadingRegistryOnRpcStart('rpc-u');
  expect(getLoadingRegistryDebugState().barCount).toBe(1);
  expect(getLoadingRegistryDebugState().userBusyCount).toBe(1);
  flushLoadingDisplayForTests();
  expect(useLoadingBarVisible().value).toBe(true);
  expect(useLoadingSlowHintVisible().value).toBe(true);
  loadingRegistryOnRpcFinish('rpc-u');
  flushLoadingDisplayForTests();
  expect(useLoadingBarVisible().value).toBe(false);
  expect(useLoadingSlowHintVisible().value).toBe(false);
});

test('slow hint dwells for min-show after user-busy ends', async () => {
  afterEachReset();
  loadingRegistryOnRpcStart('rpc-dwell');
  flushLoadingDisplayForTests();
  expect(useLoadingSlowHintVisible().value).toBe(true);
  loadingRegistryOnRpcFinish('rpc-dwell');
  // Busy gone but hint stays until min-show elapses (flush bypasses dwell).
  expect(useLoadingSlowHintVisible().value).toBe(true);
  await new Promise<void>(resolve => setTimeout(resolve, loadingTiming.SLOW_HINT_MIN_SHOW_MS + 50));
  expect(useLoadingSlowHintVisible().value).toBe(false);
});

test('navigation ticket counts bar and user-busy; settles on surface end', async () => {
  afterEachReset();
  beginNavigationLoading();
  expect(getLoadingRegistryDebugState().navTicketActive).toBe(true);
  expect(getLoadingRegistryDebugState().barCount).toBe(1);
  expect(getLoadingRegistryDebugState().userBusyCount).toBe(1);

  await runWithLoading({ layer: 'surface', target: 'page', mode: 'skeleton' }, async () => {
    expect(getLoadingRegistryDebugState().barCount).toBe(2);
    expect(getLoadingRegistryDebugState().userBusyCount).toBe(2);
  });
  expect(getLoadingRegistryDebugState().navTicketActive).toBe(false);
  expect(getLoadingRegistryDebugState().barCount).toBe(0);
  expect(getLoadingRegistryDebugState().userBusyCount).toBe(0);

  beginNavigationLoading();
  settleNavigationIfActive();
  expect(getLoadingRegistryDebugState().navTicketActive).toBe(false);
  expect(getLoadingRegistryDebugState().barCount).toBe(0);
  expect(getLoadingRegistryDebugState().userBusyCount).toBe(0);
});

test('scheduleNavigationSettleIfIdle drops nav bar and user-busy when no surface started', async () => {
  afterEachReset();
  beginNavigationLoading();
  expect(getLoadingRegistryDebugState().navTicketActive).toBe(true);
  expect(getLoadingRegistryDebugState().userBusyCount).toBe(1);
  scheduleNavigationSettleIfIdle(0);
  await new Promise<void>(resolve => setTimeout(resolve, 20));
  expect(getLoadingRegistryDebugState().navTicketActive).toBe(false);
  expect(getLoadingRegistryDebugState().barCount).toBe(0);
  expect(getLoadingRegistryDebugState().userBusyCount).toBe(0);
});

test('beginNavigationLoading is idempotent (handoff before router beforeEach)', () => {
  afterEachReset();
  beginNavigationLoading();
  beginNavigationLoading();
  expect(getLoadingRegistryDebugState().navTicketActive).toBe(true);
  expect(getLoadingRegistryDebugState().barCount).toBe(1);
  expect(getLoadingRegistryDebugState().userBusyCount).toBe(1);
  expect(useNavigationHandoffBusy().value).toBe(true);
  settleNavigationIfActive();
  expect(getLoadingRegistryDebugState().navTicketActive).toBe(false);
  expect(useNavigationHandoffBusy().value).toBe(false);
});

test('navigation handoff busy clears when surface takes over', async () => {
  afterEachReset();
  beginNavigationLoading();
  expect(useNavigationHandoffBusy().value).toBe(true);
  await runWithLoading({ layer: 'surface', target: 'land', mode: 'block' }, async () => {
    expect(useNavigationHandoffBusy().value).toBe(false);
    expect(useHasActiveSurfaceBusy().value).toBe(true);
  });
  expect(useNavigationHandoffBusy().value).toBe(false);
  expect(useHasActiveSurfaceBusy().value).toBe(false);
});

test('scheduleNavigationSettleIfIdle keeps nav ticket when surface starts within grace', async () => {
  afterEachReset();
  expect(loadingTiming.NAV_IDLE_SETTLE_MS).toBeGreaterThanOrEqual(2000);
  beginNavigationLoading();
  scheduleNavigationSettleIfIdle(30);
  await new Promise<void>(resolve => setTimeout(resolve, 10));
  expect(getLoadingRegistryDebugState().navTicketActive).toBe(true);
  await runWithLoading({ layer: 'surface', target: 'land-list', mode: 'block' }, async () => {
    await new Promise<void>(resolve => setTimeout(resolve, 40));
    expect(getLoadingRegistryDebugState().navTicketActive).toBe(true);
  });
  expect(getLoadingRegistryDebugState().navTicketActive).toBe(false);
  expect(getLoadingRegistryDebugState().barCount).toBe(0);
});

test('runWithLoading silent suppresses nested RPC chrome', async () => {
  afterEachReset();
  await runWithLoading({ silent: true }, async () => {
    loadingRegistryOnRpcStart('rpc-silent');
    expect(getLoadingRegistryDebugState().barCount).toBe(0);
    expect(getLoadingRegistryDebugState().userBusyCount).toBe(0);
    loadingRegistryOnRpcFinish('rpc-silent');
  });
  expect(getLoadingRegistryDebugState().barCount).toBe(0);
});

test('runWithLoading rethrows and still releases tickets', async () => {
  afterEachReset();
  let caught: unknown;
  try {
    await runWithLoading({ layer: 'control', target: 'x' }, async () => {
      throw new Error('boom');
    });
  } catch (e) {
    caught = e;
  }
  expect(caught instanceof Error && caught.message === 'boom').toBe(true);
  expect(getControlBusy('x').busy).toBe(false);
  expect(getLoadingRegistryDebugState().userBusyCount).toBe(0);
  expect(getLoadingRegistryDebugState().stackDepth).toBe(0);
});
