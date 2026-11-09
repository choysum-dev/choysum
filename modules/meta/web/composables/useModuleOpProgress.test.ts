// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  clampRetryDelay,
  createModuleOpProgressSession,
  type ModuleOpProgressDeps,
  type ModuleOpStatusSnapshot,
} from './useModuleOpProgress';
import { emptyAsyncIterable, fnRecorder } from '@/web/web/__tests__/mountApp';

function snapshot(partial: Partial<ModuleOpStatusSnapshot> & { status: string }): ModuleOpStatusSnapshot {
  return { ...partial };
}

type FakeTimer = { id: number; at: number; fn: () => void };

function createFakeClock(): {
  deps: Pick<ModuleOpProgressDeps, 'now' | 'schedule' | 'clearSchedule'>;
  advance: (ms: number) => Promise<void>;
} {
  let nowMs = 0;
  let nextId = 1;
  let timers: FakeTimer[] = [];

  const deps: Pick<ModuleOpProgressDeps, 'now' | 'schedule' | 'clearSchedule'> = {
    now: () => nowMs,
    schedule: (fn, ms) => {
      const id = nextId++;
      timers.push({ id, at: nowMs + ms, fn });
      return id as unknown as ReturnType<typeof setTimeout>;
    },
    clearSchedule: id => {
      const n = id as unknown as number;
      timers = timers.filter(t => t.id !== n);
    },
  };

  async function advance(ms: number): Promise<void> {
    nowMs += ms;
    for (;;) {
      const due = timers.filter(t => t.at <= nowMs).sort((a, b) => a.at - b.at || a.id - b.id);
      if (due.length === 0) return;
      const fired = new Set(due.map(t => t.id));
      timers = timers.filter(t => !fired.has(t.id));
      for (const t of due) {
        t.fn();
      }
      await Promise.resolve();
    }
  }

  return { deps, advance };
}

test('clampRetryDelay: clamps finite server delay into poll bounds', () => {
  expect(clampRetryDelay(undefined, 1000)).toBe(1000);
  expect(clampRetryDelay(Number.NaN, 1000)).toBe(1000);
  expect(clampRetryDelay(2500, 1000)).toBe(2500);
  expect(clampRetryDelay(60_000, 1000)).toBe(5000);
});

test('createModuleOpProgressSession: skips tip when boot status is terminal', async () => {
  const onTips = fnRecorder(async () => undefined);
  const subscribeModuleOp = fnRecorder(() => emptyAsyncIterable());
  const fetchStatus = fnRecorder(async () => snapshot({ status: 'succeeded', ReloadWeb: false }));
  const onTerminal = fnRecorder();
  const session = createModuleOpProgressSession(
    {
      fetchStatus: fetchStatus as any,
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: onTerminal as any,
      onTimeout: () => undefined,
    },
    { onTips: onTips as any, subscribeModuleOp: subscribeModuleOp as any }
  );

  await session.watch('job-1');

  expect(fetchStatus.calls.length).toBe(1);
  expect(onTerminal.calls.length).toBe(1);
  expect(onTips.calls.length).toBe(0);
  expect(subscribeModuleOp.calls.length).toBe(0);
});

test('createModuleOpProgressSession: reloads when terminal boot requests ReloadWeb', async () => {
  const reloadWeb = fnRecorder();
  const session = createModuleOpProgressSession(
    {
      fetchStatus: async () => snapshot({ status: 'succeeded', ReloadWeb: true }),
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: () => undefined,
    },
    {
      onTips: async () => undefined,
      subscribeModuleOp: () => emptyAsyncIterable(),
      reloadWeb: reloadWeb as any,
    }
  );

  await session.watch('job-reload');
  expect(reloadWeb.calls.length).toBe(1);
});

test('createModuleOpProgressSession: no-ops for empty job ids', async () => {
  const fetchStatus = fnRecorder();
  const onTips = fnRecorder(async () => undefined);
  const session = createModuleOpProgressSession(
    {
      fetchStatus: fetchStatus as any,
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: () => undefined,
    },
    { onTips: onTips as any, subscribeModuleOp: () => emptyAsyncIterable() }
  );
  await session.watch('   ');
  expect(fetchStatus.calls.length).toBe(0);
  expect(onTips.calls.length).toBe(0);
});

test('createModuleOpProgressSession: returns early when dialog becomes inactive during boot', async () => {
  let active = true;
  const onTips = fnRecorder(async () => undefined);
  const session = createModuleOpProgressSession(
    {
      fetchStatus: async () => {
        active = false;
        return snapshot({ status: 'queued' });
      },
      isActive: () => active,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: () => undefined,
    },
    { onTips: onTips as any, subscribeModuleOp: () => emptyAsyncIterable() }
  );
  await session.watch('job-inactive-boot');
  expect(onTips.calls.length).toBe(0);
});

test('createModuleOpProgressSession: refreshes on tip and reaches terminal without polling', async () => {
  const { deps, advance } = createFakeClock();
  const statuses = [
    snapshot({ status: 'queued' }),
    snapshot({ status: 'dispatching' }),
    snapshot({ status: 'succeeded' }),
  ];
  const fetchStatus = fnRecorder(async () => statuses.shift()!);
  const onTerminal = fnRecorder();
  const subscribeModuleOp = fnRecorder(() => emptyAsyncIterable());
  const onTips = fnRecorder(async (_stream: unknown, callback: () => Promise<void>) => {
    await callback();
    await advance(80);
    await callback();
    await advance(80);
  });

  const session = createModuleOpProgressSession(
    {
      fetchStatus: fetchStatus as any,
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: onTerminal as any,
      onTimeout: () => undefined,
    },
    {
      ...deps,
      onTips: onTips as any,
      subscribeModuleOp: subscribeModuleOp as any,
    }
  );

  await session.watch('job-2');

  expect(subscribeModuleOp.calls[0]?.[0]).toEqual({ operationId: 'job-2' });
  expect(onTerminal.calls.length).toBe(1);
  expect(fetchStatus.calls.length).toBeGreaterThanOrEqual(3);
  const afterTip = fetchStatus.calls.length;
  await advance(5_000);
  expect(fetchStatus.calls.length).toBe(afterTip);
});

test('createModuleOpProgressSession: reports tip refresh hard errors', async () => {
  const { deps, advance } = createFakeClock();
  const onHardError = fnRecorder();
  let n = 0;
  const fetchStatus = fnRecorder(async () => {
    n += 1;
    if (n === 1) return snapshot({ status: 'queued' });
    if (n === 2) throw new Error('status exploded');
    return snapshot({ status: 'succeeded' });
  });
  const onTips = fnRecorder(async (_stream: unknown, callback: () => Promise<void>) => {
    await callback();
    await advance(80);
  });

  const session = createModuleOpProgressSession(
    {
      fetchStatus: fetchStatus as any,
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: () => undefined,
      onHardError: onHardError as any,
    },
    { ...deps, onTips: onTips as any, subscribeModuleOp: () => emptyAsyncIterable() }
  );
  const pending = session.watch('job-tip-err');
  await Promise.resolve();
  await advance(1_000);
  await pending;
  expect(onHardError.calls[0]?.[0]).toBe('status exploded');
});

test('createModuleOpProgressSession: notifies transient tip refresh errors once', async () => {
  const { deps, advance } = createFakeClock();
  const onTransientNetworkError = fnRecorder();
  let n = 0;
  const fetchStatus = fnRecorder(async () => {
    n += 1;
    if (n === 1) return snapshot({ status: 'queued' });
    if (n === 2) throw new Error('Failed to fetch');
    if (n === 3) throw new Error('NetworkError again');
    return snapshot({ status: 'succeeded' });
  });
  const onTips = fnRecorder(async (_stream: unknown, callback: () => Promise<void>) => {
    await callback();
    await advance(80);
    await callback();
    await advance(80);
  });

  const session = createModuleOpProgressSession(
    {
      fetchStatus: fetchStatus as any,
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: () => undefined,
      onTransientNetworkError: onTransientNetworkError as any,
    },
    { ...deps, onTips: onTips as any, subscribeModuleOp: () => emptyAsyncIterable() }
  );
  const pending = session.watch('job-tip-transient');
  await Promise.resolve();
  await advance(1_000);
  await pending;
  expect(onTransientNetworkError.calls.length).toBe(1);
});

test('createModuleOpProgressSession: polls when tip subscribe is unavailable', async () => {
  const { deps, advance } = createFakeClock();
  let calls = 0;
  const fetchStatus = fnRecorder(async () => {
    calls += 1;
    if (calls >= 2) return snapshot({ status: 'succeeded' });
    return snapshot({ status: 'queued' });
  });
  const onTerminal = fnRecorder();
  const session = createModuleOpProgressSession(
    {
      fetchStatus: fetchStatus as any,
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: onTerminal as any,
      onTimeout: () => undefined,
    },
    {
      ...deps,
      // Non-functions force the tip-unavailable poll path (?? would keep real tip defaults).
      onTips: 0 as any,
      subscribeModuleOp: 0 as any,
    },
  );

  const pending = session.watch('job-no-tip');
  await Promise.resolve();
  await advance(1_000);
  await advance(1_500);
  await pending.catch(() => undefined);
  expect(onTerminal.calls.length).toBe(1);
});

test('createModuleOpProgressSession: starts poll fallback when tip ends without terminal', async () => {
  const { deps, advance } = createFakeClock();
  let calls = 0;
  const fetchStatus = fnRecorder(async () => {
    calls += 1;
    if (calls >= 3) return snapshot({ status: 'succeeded' });
    return snapshot({ status: 'queued' });
  });
  const onTerminal = fnRecorder();
  const onTips = fnRecorder(async () => undefined);

  const session = createModuleOpProgressSession(
    {
      fetchStatus: fetchStatus as any,
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: onTerminal as any,
      onTimeout: () => undefined,
    },
    { ...deps, onTips: onTips as any, subscribeModuleOp: () => emptyAsyncIterable() }
  );

  // watch stays pending through tip end + poll fallback until terminal.
  const pending = session.watch('job-poll');
  let settled = false;
  void pending.then(() => {
    settled = true;
  });
  for (let i = 0; i < 40 && !settled; i++) {
    await Promise.resolve();
    await advance(500);
  }
  await pending;
  expect(onTerminal.calls.length).toBe(1);
  expect(fetchStatus.calls.length).toBeGreaterThanOrEqual(3);
});

test('createModuleOpProgressSession: fires timeout via deadline schedule', async () => {
  const { deps, advance } = createFakeClock();
  const onTimeout = fnRecorder();
  const fetchStatus = fnRecorder(async () => snapshot({ status: 'queued' }));
  const onTips = fnRecorder(async () => {
    await advance(10 * 60 * 1000);
  });

  const session = createModuleOpProgressSession(
    {
      fetchStatus: fetchStatus as any,
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: onTimeout as any,
    },
    { ...deps, onTips: onTips as any, subscribeModuleOp: () => emptyAsyncIterable() }
  );

  await session.watch('job-timeout');
  expect(onTimeout.calls.length).toBe(1);
});

test('createModuleOpProgressSession: stop aborts an in-flight tip session', async () => {
  const { deps, advance } = createFakeClock();
  const fetchStatus = fnRecorder(async () => snapshot({ status: 'queued' }));
  let tipStarted = false;
  const onTips = fnRecorder(async (_stream: unknown, _cb: () => Promise<void>, signal: AbortSignal) => {
    tipStarted = true;
    await new Promise<void>(resolve => {
      const check = () => {
        if (signal.aborted) {
          resolve();
          return;
        }
        void advance(10).then(check);
      };
      check();
    });
  });

  const session = createModuleOpProgressSession(
    {
      fetchStatus: fetchStatus as any,
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: () => undefined,
    },
    { ...deps, onTips: onTips as any, subscribeModuleOp: () => emptyAsyncIterable() }
  );

  const watchPromise = session.watch('job-stop');
  for (let i = 0; i < 50 && !tipStarted; i++) {
    await Promise.resolve();
    await advance(1);
  }
  expect(tipStarted).toBe(true);
  session.stop();
  await watchPromise;
});

test('createModuleOpProgressSession: poll fallback reports errors and times out', async () => {
  const { deps, advance } = createFakeClock();
  const onHardError = fnRecorder();
  const onTransientNetworkError = fnRecorder();
  const onTimeout = fnRecorder();
  let n = 0;
  const fetchStatus = fnRecorder(async () => {
    n += 1;
    if (n === 1) return snapshot({ status: 'queued' });
    if (n === 2) throw new Error('Failed to fetch');
    if (n === 3) throw new Error('status exploded');
    return snapshot({ status: 'succeeded' });
  });
  const session = createModuleOpProgressSession(
    {
      fetchStatus: fetchStatus as any,
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: onTimeout as any,
      onHardError: onHardError as any,
      onTransientNetworkError: onTransientNetworkError as any,
    },
    { ...deps, onTips: 0 as any, subscribeModuleOp: 0 as any },
  );

  const pending = session.watch('job-poll-errors');
  let settled = false;
  void pending.then(() => {
    settled = true;
  });
  for (let i = 0; i < 40 && !settled; i++) {
    await Promise.resolve();
    await advance(500);
  }
  await pending;
  expect(onTransientNetworkError.calls.length).toBe(1);
  expect(onHardError.calls[0]?.[0]).toBe('status exploded');

  // Tip-unavailable path clears the tip deadline; poll itself must enforce MAX_DURATION.
  n = 0;
  const timeoutSession = createModuleOpProgressSession(
    {
      fetchStatus: async () => snapshot({ status: 'queued' }),
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: onTimeout as any,
    },
    { ...deps, onTips: 0 as any, subscribeModuleOp: 0 as any },
  );
  const timeoutPending = timeoutSession.watch('job-poll-timeout');
  settled = false;
  void timeoutPending.then(() => {
    settled = true;
  });
  for (let i = 0; i < 30 && !settled; i++) {
    await Promise.resolve();
    await advance(60_000);
  }
  await timeoutPending;
  expect(onTimeout.calls.length).toBe(1);
});

test('createModuleOpProgressSession: poll fallback settles on stop and inactive reads', async () => {
  const { deps, advance } = createFakeClock();
  let active = true;
  let calls = 0;
  const fetchStatus = fnRecorder(async () => {
    calls += 1;
    if (calls === 1) return snapshot({ status: 'queued' });
    active = false;
    return snapshot({ status: 'queued' });
  });
  const inactiveSession = createModuleOpProgressSession(
    {
      fetchStatus: fetchStatus as any,
      isActive: () => active,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: () => undefined,
    },
    { ...deps, onTips: 0 as any, subscribeModuleOp: 0 as any },
  );
  await inactiveSession.watch('job-poll-inactive');

  active = true;
  calls = 0;
  let release: (() => void) | undefined;
  const blockedFetch = fnRecorder(async () => {
    calls += 1;
    if (calls === 1) return snapshot({ status: 'queued' });
    await new Promise<void>(resolve => {
      release = resolve;
    });
    throw new Error('after-stop');
  });
  const stopSession = createModuleOpProgressSession(
    {
      fetchStatus: blockedFetch as any,
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: () => undefined,
      onHardError: () => undefined,
    },
    { ...deps, onTips: 0 as any, subscribeModuleOp: 0 as any },
  );
  const pending = stopSession.watch('job-poll-stop');
  for (let i = 0; i < 40 && !release; i++) await Promise.resolve();
  expect(!!release).toBe(true);
  stopSession.stop();
  release?.();
  await pending;

  // Orphaned poll timer after stop (clearSchedule no-op) still exits on generation mismatch.
  const orphan = createModuleOpProgressSession(
    {
      fetchStatus: async () => snapshot({ status: 'queued' }),
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: () => undefined,
    },
    {
      ...deps,
      clearSchedule: () => undefined,
      onTips: 0 as any,
      subscribeModuleOp: 0 as any,
    },
  );
  const orphanPending = orphan.watch('job-poll-orphan-tick');
  for (let i = 0; i < 20; i++) await Promise.resolve();
  orphan.stop();
  await orphanPending;
  await advance(1_000);
  await Promise.resolve();
});

test('createModuleOpProgressSession: default reloadWeb uses window when ReloadWeb', async () => {
  const reloads: number[] = [];
  const prev = (globalThis as any).window;
  (globalThis as any).window = { location: { reload: () => reloads.push(1) } };
  try {
    const session = createModuleOpProgressSession(
      {
        fetchStatus: async () => snapshot({ status: 'succeeded', ReloadWeb: true }),
        isActive: () => true,
        onStatus: () => undefined,
        onTerminal: () => undefined,
        onTimeout: () => undefined,
      },
      { onTips: async () => undefined, subscribeModuleOp: () => emptyAsyncIterable() },
    );
    await session.watch('job-host-reload');
    expect(reloads.length).toBe(1);
  } finally {
    (globalThis as any).window = prev;
  }
});

test('createModuleOpProgressSession: boot fetch errors notify then continue tip path', async () => {
  const onTransientNetworkError = fnRecorder();
  const onHardError = fnRecorder();
  let transientCalls = 0;
  const transientFetch = fnRecorder(async () => {
    transientCalls += 1;
    if (transientCalls === 1) throw new Error('NetworkError when attempting to fetch resource');
    return snapshot({ status: 'succeeded' });
  });
  const session = createModuleOpProgressSession(
    {
      fetchStatus: transientFetch as any,
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: () => undefined,
      onTransientNetworkError: onTransientNetworkError as any,
      onHardError: onHardError as any,
    },
    { onTips: async () => undefined, subscribeModuleOp: () => emptyAsyncIterable() },
  );
  await session.watch('job-boot-transient');
  expect(onTransientNetworkError.calls.length).toBe(1);
  expect(onHardError.calls.length).toBe(0);

  const onHardErrorOnly = fnRecorder();
  let hardCalls = 0;
  const hardFetch = fnRecorder(async () => {
    hardCalls += 1;
    if (hardCalls === 1) throw new Error('boot-status-fail');
    return snapshot({ status: 'succeeded' });
  });
  const hardSession = createModuleOpProgressSession(
    {
      fetchStatus: hardFetch as any,
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: () => undefined,
      onHardError: onHardErrorOnly as any,
    },
    { onTips: async () => undefined, subscribeModuleOp: () => emptyAsyncIterable() },
  );
  await hardSession.watch('job-boot-hard');
  expect(onHardErrorOnly.calls[0]?.[0]).toBe('boot-status-fail');
});

test('createModuleOpProgressSession: tip refresh errors surface transient and hard', async () => {
  const { deps, advance } = createFakeClock();
  const onTransientNetworkError = fnRecorder();
  const onHardError = fnRecorder();
  let n = 0;
  const fetchStatus = fnRecorder(async () => {
    n += 1;
    if (n === 1) return snapshot({ status: 'queued' });
    if (n === 2) throw new Error('NetworkError when attempting to fetch resource');
    if (n === 3) throw new Error('tip-refresh-fail');
    return snapshot({ status: 'succeeded' });
  });
  const onTips = fnRecorder(async (_stream: unknown, callback: () => Promise<void>) => {
    await callback();
    await advance(80);
    await callback();
    await advance(80);
    await callback();
    await advance(80);
  });
  const session = createModuleOpProgressSession(
    {
      fetchStatus: fetchStatus as any,
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: () => undefined,
      onTransientNetworkError: onTransientNetworkError as any,
      onHardError: onHardError as any,
    },
    { ...deps, onTips: onTips as any, subscribeModuleOp: () => emptyAsyncIterable() },
  );
  await session.watch('job-tip-errors');
  expect(onTransientNetworkError.calls.length).toBe(1);
  expect(onHardError.calls[0]?.[0]).toBe('tip-refresh-fail');
});

test('createModuleOpProgressSession: poll honors retryAfterMs clamp', async () => {
  const { deps, advance } = createFakeClock();
  let calls = 0;
  const fetchStatus = fnRecorder(async () => {
    calls += 1;
    if (calls === 1) return snapshot({ status: 'queued', retryAfterMs: 2500 });
    return snapshot({ status: 'succeeded' });
  });
  const session = createModuleOpProgressSession(
    {
      fetchStatus: fetchStatus as any,
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: () => undefined,
    },
    { ...deps, onTips: 0 as any, subscribeModuleOp: 0 as any },
  );
  const pending = session.watch('job-retry-after');
  let settled = false;
  void pending.then(() => {
    settled = true;
  });
  for (let i = 0; i < 20 && !settled; i += 1) {
    await advance(500);
    await Promise.resolve();
  }
  await pending;
  expect(calls).toBeGreaterThanOrEqual(2);
});

test('createModuleOpProgressSession: fireTimeout is a no-op after stop bumps generation', async () => {
  const { deps, advance } = createFakeClock();
  const onTimeout = fnRecorder();
  const session = createModuleOpProgressSession(
    {
      fetchStatus: async () => snapshot({ status: 'queued' }),
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: onTimeout as any,
    },
    { ...deps, onTips: 0 as any, subscribeModuleOp: 0 as any },
  );
  const pending = session.watch('job-stop-timeout');
  await advance(1);
  session.stop();
  await advance(11 * 60 * 1000);
  await pending;
  expect(onTimeout.calls.length).toBe(0);
});

test('createModuleOpProgressSession: applyFetched returns early when inactive before second guard', async () => {
  const { deps, advance } = createFakeClock();
  let active = true;
  let release: (() => void) | undefined;
  let calls = 0;
  const fetchStatus = fnRecorder(async () => {
    calls += 1;
    if (calls === 1) return snapshot({ status: 'queued' });
    await new Promise<void>(resolve => {
      release = resolve;
    });
    active = false;
    return snapshot({ status: 'succeeded' });
  });
  const onTerminal = fnRecorder();
  const session = createModuleOpProgressSession(
    {
      fetchStatus: fetchStatus as any,
      isActive: () => active,
      onStatus: () => undefined,
      onTerminal: onTerminal as any,
      onTimeout: () => undefined,
    },
    { ...deps, onTips: 0 as any, subscribeModuleOp: 0 as any },
  );
  const pending = session.watch('job-inactive-mid-fetch');
  for (let i = 0; i < 30 && !release; i += 1) {
    await advance(500);
    await Promise.resolve();
  }
  release?.();
  await pending;
  expect(onTerminal.calls.length).toBe(0);
});

test('createModuleOpProgressSession: applyFetched skips when inactive before fetch', async () => {
  const fetchStatus = fnRecorder();
  const session = createModuleOpProgressSession(
    {
      fetchStatus: fetchStatus as any,
      isActive: () => false,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: () => undefined,
    },
    { onTips: async () => undefined, subscribeModuleOp: () => emptyAsyncIterable() },
  );
  await session.watch('job-inactive-pre-fetch');
  expect(fetchStatus.calls.length).toBe(0);
});

test('createModuleOpProgressSession: boot catch exits when session becomes inactive', async () => {
  const onHardError = fnRecorder();
  let active = true;
  const session = createModuleOpProgressSession(
    {
      fetchStatus: async () => {
        active = false;
        throw new Error('boot-status-fail');
      },
      isActive: () => active,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: () => undefined,
      onHardError: onHardError as any,
    },
    { onTips: async () => undefined, subscribeModuleOp: () => emptyAsyncIterable() },
  );
  await session.watch('job-boot-inactive');
  expect(onHardError.calls.length).toBe(0);
});

test('createModuleOpProgressSession: stale generation deadline does not call onTimeout', async () => {
  const { deps, advance } = createFakeClock();
  const onTimeout = fnRecorder();
  const session = createModuleOpProgressSession(
    {
      fetchStatus: async () => snapshot({ status: 'queued' }),
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: onTimeout as any,
    },
    {
      ...deps,
      clearSchedule: () => undefined,
      onTips: 0 as any,
      subscribeModuleOp: 0 as any,
    },
  );
  const pending = session.watch('job-stale-deadline');
  session.stop();
  await advance(11 * 60 * 1000);
  await pending;
  expect(onTimeout.calls.length).toBe(0);
});

test('createModuleOpProgressSession: deadline is ignored when session is inactive', async () => {
  const { deps, advance } = createFakeClock();
  const onTimeout = fnRecorder();
  let active = true;
  const session = createModuleOpProgressSession(
    {
      fetchStatus: async () => snapshot({ status: 'queued' }),
      isActive: () => active,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: onTimeout as any,
    },
    { ...deps, onTips: 0 as any, subscribeModuleOp: 0 as any },
  );
  const pending = session.watch('job-inactive-deadline');
  active = false;
  await advance(11 * 60 * 1000);
  await pending;
  expect(onTimeout.calls.length).toBe(0);
});

test('createModuleOpProgressSession: stale fireTimeout after poll timeout is ignored', async () => {
  const { deps, advance } = createFakeClock();
  const onTimeout = fnRecorder();
  const session = createModuleOpProgressSession(
    {
      fetchStatus: async () => snapshot({ status: 'queued' }),
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: onTimeout as any,
    },
    { ...deps, onTips: 0 as any, subscribeModuleOp: 0 as any },
  );
  const pending = session.watch('job-double-timeout');
  let settled = false;
  void pending.then(() => {
    settled = true;
  });
  for (let i = 0; i < 40 && !settled; i += 1) {
    await advance(60_000);
    await Promise.resolve();
  }
  await pending;
  expect(onTimeout.calls.length).toBe(1);
  await advance(11 * 60 * 1000);
  expect(onTimeout.calls.length).toBe(1);
});

test('createModuleOpProgressSession: tip refresh ignores stale generation after stop', async () => {
  const { deps, advance } = createFakeClock();
  const onHardError = fnRecorder();
  let calls = 0;
  const fetchStatus = fnRecorder(async () => {
    calls += 1;
    if (calls === 1) return snapshot({ status: 'queued' });
    throw new Error('tip-refresh-fail');
  });
  let tipCalls = 0;
  let tipSession!: ReturnType<typeof createModuleOpProgressSession>;
  tipSession = createModuleOpProgressSession(
    {
      fetchStatus: fetchStatus as any,
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: () => undefined,
      onHardError: onHardError as any,
    },
    {
      ...deps,
      onTips: (async (_stream, callback) => {
        tipCalls += 1;
        if (tipCalls === 1) await callback();
        tipSession.stop();
        await callback();
        await advance(80);
        await callback();
      }) as any,
      subscribeModuleOp: () => emptyAsyncIterable(),
    },
  );
  await tipSession.watch('job-tip-stale');
  expect(onHardError.calls.length).toBe(0);
});

test('createModuleOpProgressSession: tip refresh errors after stop stay silent', async () => {
  const { deps, advance } = createFakeClock();
  const onHardError = fnRecorder();
  let calls = 0;
  let tipSession!: ReturnType<typeof createModuleOpProgressSession>;
  const fetchStatus = fnRecorder(async () => {
    calls += 1;
    if (calls === 1) return snapshot({ status: 'queued' });
    tipSession.stop();
    throw new Error('after-stop');
  });
  tipSession = createModuleOpProgressSession(
    {
      fetchStatus: fetchStatus as any,
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: () => undefined,
      onHardError: onHardError as any,
    },
    {
      ...deps,
      onTips: (async (_stream, callback) => {
        await callback();
        await advance(80);
      }) as any,
      subscribeModuleOp: () => emptyAsyncIterable(),
    },
  );
  await tipSession.watch('job-tip-stop-mid-error');
  expect(onHardError.calls.length).toBe(0);
});

test('createModuleOpProgressSession: clampRetryDelay bounds retryAfterMs on poll ticks', async () => {
  const { deps, advance } = createFakeClock();
  let calls = 0;
  const fetchStatus = fnRecorder(async () => {
    calls += 1;
    if (calls === 1) return snapshot({ status: 'queued', retryAfterMs: 60_000 });
    return snapshot({ status: 'succeeded' });
  });
  const session = createModuleOpProgressSession(
    {
      fetchStatus: fetchStatus as any,
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: () => undefined,
    },
    { ...deps, onTips: 0 as any, subscribeModuleOp: 0 as any },
  );
  const pending = session.watch('job-clamp-retry');
  let settled = false;
  void pending.then(() => {
    settled = true;
  });
  for (let i = 0; i < 30 && !settled; i += 1) {
    await advance(1_000);
    await Promise.resolve();
  }
  await pending;
  expect(calls).toBeGreaterThanOrEqual(2);
});
