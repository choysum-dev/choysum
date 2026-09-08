// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  createModuleOpProgressSession,
  type ModuleOpProgressDeps,
  type ModuleOpStatusSnapshot,
} from './useModuleOpProgress';

type CallRecorder = { calls: unknown[][] };

function fnRecorder<T = undefined>(
  impl?: (...args: unknown[]) => T | Promise<T>
): CallRecorder & ((...args: unknown[]) => T | Promise<T>) {
  const rec: CallRecorder & ((...args: unknown[]) => T | Promise<T>) = Object.assign(
    (...args: unknown[]) => {
      rec.calls.push(args);
      return impl ? impl(...args) : (undefined as T);
    },
    { calls: [] as unknown[][] }
  );
  return rec;
}

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

test('createModuleOpProgressSession: skips tip when boot status is terminal', async () => {
  const onTips = fnRecorder(async () => undefined);
  const subscribeModuleOp = fnRecorder(() => ({}));
  const fetchStatus = fnRecorder(async () => snapshot({ status: 'succeeded', reload_web: false }));
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

test('createModuleOpProgressSession: reloads when terminal boot requests reload_web', async () => {
  const reloadWeb = fnRecorder();
  const session = createModuleOpProgressSession(
    {
      fetchStatus: async () => snapshot({ status: 'succeeded', reload_web: true }),
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: () => undefined,
    },
    {
      onTips: async () => undefined,
      subscribeModuleOp: () => ({}),
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
    { onTips: onTips as any, subscribeModuleOp: () => ({}) }
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
    { onTips: onTips as any, subscribeModuleOp: () => ({}) }
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
  const subscribeModuleOp = fnRecorder(() => ({ stream: true }));
  const onTips = fnRecorder(async (_stream: unknown, callback: () => Promise<void>) => {
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

  expect(subscribeModuleOp.calls[0]?.[0]).toBe('job-2');
  expect(onTerminal.calls.length).toBe(1);
  expect(fetchStatus.calls.length).toBeGreaterThanOrEqual(2);
  const afterTip = fetchStatus.calls.length;
  await advance(5_000);
  expect(fetchStatus.calls.length).toBe(afterTip);
});

test('createModuleOpProgressSession: reports tip refresh hard errors', async () => {
  const { deps, advance } = createFakeClock();
  const onHardError = fnRecorder();
  const fetchStatus = fnRecorder(async () => {
    if (fetchStatus.calls.length === 1) return snapshot({ status: 'queued' });
    throw new Error('status exploded');
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
    { ...deps, onTips: onTips as any, subscribeModuleOp: () => ({}) }
  );
  await session.watch('job-tip-err');
  expect(onHardError.calls[0]?.[0]).toBe('status exploded');
});

test('createModuleOpProgressSession: notifies transient tip refresh errors once', async () => {
  const { deps, advance } = createFakeClock();
  const onTransientNetworkError = fnRecorder();
  let n = 0;
  const fetchStatus = fnRecorder(async () => {
    n += 1;
    if (n === 1) return snapshot({ status: 'queued' });
    throw new Error(n === 2 ? 'Failed to fetch' : 'NetworkError again');
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
    { ...deps, onTips: onTips as any, subscribeModuleOp: () => ({}) }
  );
  await session.watch('job-tip-transient');
  expect(onTransientNetworkError.calls.length).toBe(1);
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
    { ...deps, onTips: onTips as any, subscribeModuleOp: () => ({}) }
  );

  // watch resolves when the tip stream ends; poll fallback continues in the background.
  await session.watch('job-poll');
  await advance(1_000);
  await advance(1_500);
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
    { ...deps, onTips: onTips as any, subscribeModuleOp: () => ({}) }
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
    { ...deps, onTips: onTips as any, subscribeModuleOp: () => ({}) }
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
