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
  runWithLoadingSync,
  scheduleNavigationSettleIfIdle,
  settleNavigationIfActive,
  loadingTiming,
  useControlBusy,
  useLoadingBarVisible,
  useLoadingSlowHintVisible,
  useHasActiveSurfaceBusy,
  useNavigationHandoffBusy,
  useSurfaceBusy,
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

test('older navigation settlement does not clear a newer ticket', async () => {
  afterEachReset();
  const genA = beginNavigationLoading();
  scheduleNavigationSettleIfIdle(30);
  const genB = beginNavigationLoading();
  expect(genB).toBeGreaterThan(genA);
  expect(getLoadingRegistryDebugState().navTicketActive).toBe(true);
  settleNavigationIfActive({ generation: genA });
  expect(getLoadingRegistryDebugState().navTicketActive).toBe(true);
  await new Promise<void>(resolve => setTimeout(resolve, 50));
  // Prior idle timer must not have cleared B's ticket.
  expect(getLoadingRegistryDebugState().navTicketActive).toBe(true);
  settleNavigationIfActive({ generation: genB });
  expect(getLoadingRegistryDebugState().navTicketActive).toBe(false);
});

test('overlapping runWithLoading scopes remove their own stack frame', async () => {
  afterEachReset();
  let releaseInner!: () => void;
  const innerGate = new Promise<void>(resolve => {
    releaseInner = resolve;
  });
  const innerDone = runWithLoading({ layer: 'surface', target: 'outer', mode: 'block' }, async () => {
    const inner = runWithLoading({ silent: true }, async () => {
      await innerGate;
    });
    await Promise.resolve();
    expect(getLoadingRegistryDebugState().stackDepth).toBe(2);
    // Do not await inner — outer finishes first so stack.pop-by-identity is required.
    void inner;
  });
  await innerDone;
  expect(getLoadingRegistryDebugState().stackDepth).toBe(1);
  releaseInner();
  await new Promise<void>(resolve => setTimeout(resolve, 0));
  expect(getLoadingRegistryDebugState().stackDepth).toBe(0);
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

test('runWithLoadingSync acquires and releases surface/control frames', () => {
  afterEachReset();
  const out = runWithLoadingSync({ layer: 'surface', target: 'sync-s', mode: 'block' }, () => {
    expect(getSurfaceBusy('sync-s')).toEqual({ busy: true, mode: 'block' });
    return 7;
  });
  expect(out).toBe(7);
  expect(getSurfaceBusy('sync-s').busy).toBe(false);

  runWithLoadingSync({ layer: 'control', target: 'sync-c', pendingLabel: 'Go' }, () => {
    expect(getControlBusy('sync-c')).toEqual({ busy: true, pendingLabel: 'Go' });
  });
  expect(getControlBusy('sync-c').busy).toBe(false);
});

test('runWithLoadingSync rethrows and still releases', () => {
  afterEachReset();
  let caught: unknown;
  try {
    runWithLoadingSync({ layer: 'chrome' }, () => {
      throw new Error('sync-boom');
    });
  } catch (e) {
    caught = e;
  }
  expect(caught instanceof Error && caught.message === 'sync-boom').toBe(true);
  expect(getLoadingRegistryDebugState().stackDepth).toBe(0);
  expect(getLoadingRegistryDebugState().barCount).toBe(0);
});

test('RPC start is idempotent per spanId', () => {
  afterEachReset();
  loadingRegistryOnRpcStart('rpc-dup');
  loadingRegistryOnRpcStart('rpc-dup');
  expect(getLoadingRegistryDebugState().barCount).toBe(1);
  expect(getLoadingRegistryDebugState().userBusyCount).toBe(1);
  loadingRegistryOnRpcFinish('rpc-dup');
  loadingRegistryOnRpcFinish('rpc-dup');
  expect(getLoadingRegistryDebugState().barCount).toBe(0);
  expect(getLoadingRegistryDebugState().userBusyCount).toBe(0);
});

test('chrome layer owns bar and suppresses nested RPC chrome', async () => {
  afterEachReset();
  await runWithLoading({ layer: 'chrome' }, async () => {
    expect(getLoadingRegistryDebugState().barCount).toBe(1);
    loadingRegistryOnRpcStart('rpc-chrome');
    expect(getLoadingRegistryDebugState().barCount).toBe(1);
    loadingRegistryOnRpcFinish('rpc-chrome');
  });
  expect(getLoadingRegistryDebugState().barCount).toBe(0);
});

test('nested same-target surface/control increments counts', async () => {
  afterEachReset();
  await runWithLoading({ layer: 'surface', target: 'nest', mode: 'stale' }, async () => {
    await runWithLoading({ layer: 'surface', target: 'nest', mode: 'skeleton' }, async () => {
      expect(getSurfaceBusy('nest')).toEqual({ busy: true, mode: 'skeleton' });
    });
    expect(getSurfaceBusy('nest')).toEqual({ busy: true, mode: 'skeleton' });
  });
  expect(getSurfaceBusy('nest').busy).toBe(false);

  await runWithLoading({ layer: 'control', target: 'nest-c', pendingLabel: 'A' }, async () => {
    await runWithLoading({ layer: 'control', target: 'nest-c', pendingLabel: 'B' }, async () => {
      expect(getControlBusy('nest-c')).toEqual({ busy: true, pendingLabel: 'B' });
    });
  });
  expect(getControlBusy('nest-c').busy).toBe(false);
});

test('useSurfaceBusy and useControlBusy track targets reactively', async () => {
  afterEachReset();
  const surface = useSurfaceBusy('u-s');
  const control = useControlBusy('u-c');
  expect(surface.value.busy).toBe(false);
  expect(control.value.busy).toBe(false);
  await runWithLoading({ layer: 'surface', target: 'u-s', mode: 'block' }, async () => {
    expect(surface.value).toEqual({ busy: true, mode: 'block' });
  });
  await runWithLoading({ layer: 'control', target: 'u-c', pendingLabel: 'X' }, async () => {
    expect(control.value).toEqual({ busy: true, pendingLabel: 'X' });
  });
  expect(surface.value.busy).toBe(false);
  expect(control.value.busy).toBe(false);
});

test('bar delay reveals showBar after BAR_DELAY_MS', async () => {
  afterEachReset();
  loadingRegistryOnRpcStart('rpc-bar-delay');
  expect(useLoadingBarVisible().value).toBe(false);
  await new Promise<void>(resolve => setTimeout(resolve, loadingTiming.BAR_DELAY_MS + 30));
  expect(useLoadingBarVisible().value).toBe(true);
  loadingRegistryOnRpcFinish('rpc-bar-delay');
  expect(useLoadingBarVisible().value).toBe(false);
});

test('slow hint schedules after SLOW_HINT_MS without flush', async () => {
  afterEachReset();
  loadingRegistryOnRpcStart('rpc-slow');
  expect(useLoadingSlowHintVisible().value).toBe(false);
  await new Promise<void>(resolve => setTimeout(resolve, loadingTiming.SLOW_HINT_MS + 30));
  expect(useLoadingSlowHintVisible().value).toBe(true);
  loadingRegistryOnRpcFinish('rpc-slow');
  await new Promise<void>(resolve => setTimeout(resolve, loadingTiming.SLOW_HINT_MIN_SHOW_MS + 30));
  expect(useLoadingSlowHintVisible().value).toBe(false);
});

test('scheduleNavigationSettleIfIdle no-ops when nav inactive or surface busy', async () => {
  afterEachReset();
  scheduleNavigationSettleIfIdle(0);
  await new Promise<void>(resolve => setTimeout(resolve, 10));
  expect(getLoadingRegistryDebugState().navTicketActive).toBe(false);

  beginNavigationLoading();
  await runWithLoading({ layer: 'surface', target: 'idle-surface', mode: 'block' }, async () => {
    scheduleNavigationSettleIfIdle(0);
    await new Promise<void>(resolve => setTimeout(resolve, 10));
    expect(getLoadingRegistryDebugState().navTicketActive).toBe(true);
  });
});

test('navigation timeout settles orphaned nav ticket', async () => {
  afterEachReset();
  beginNavigationLoading();
  expect(getLoadingRegistryDebugState().navTicketActive).toBe(true);
  await new Promise<void>(resolve => setTimeout(resolve, loadingTiming.NAV_TIMEOUT_MS + 30));
  expect(getLoadingRegistryDebugState().navTicketActive).toBe(false);
  expect(getLoadingRegistryDebugState().barCount).toBe(0);
});

test('slow hint hides immediately when min-show already elapsed', async () => {
  afterEachReset();
  loadingRegistryOnRpcStart('rpc-min-elapsed');
  flushLoadingDisplayForTests();
  expect(useLoadingSlowHintVisible().value).toBe(true);
  // Backdate shown-at so remain === 0 on the next removeUserBusy.
  await new Promise<void>(resolve => setTimeout(resolve, loadingTiming.SLOW_HINT_MIN_SHOW_MS + 30));
  loadingRegistryOnRpcFinish('rpc-min-elapsed');
  expect(useLoadingSlowHintVisible().value).toBe(false);
});

test('finish unknown RPC span is a no-op', () => {
  afterEachReset();
  loadingRegistryOnRpcFinish('never-started');
  expect(getLoadingRegistryDebugState().barCount).toBe(0);
});
