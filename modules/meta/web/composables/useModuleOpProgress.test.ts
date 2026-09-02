// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createModuleOpProgressSession, type ModuleOpStatusSnapshot } from './useModuleOpProgress';

const onTips = vi.fn();
const subscribeModuleOp = vi.fn(() => ({}));

vi.mock('@/core/web/tip', () => ({
  onTips: (...args: unknown[]) => onTips(...args),
  subscribeModuleOp: (...args: unknown[]) => subscribeModuleOp(...args),
}));

function snapshot(partial: Partial<ModuleOpStatusSnapshot> & { status: string }): ModuleOpStatusSnapshot {
  return { ...partial };
}

describe('createModuleOpProgressSession', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    onTips.mockReset();
    subscribeModuleOp.mockReset();
    subscribeModuleOp.mockReturnValue({});
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('skips tip subscribe when boot GetOpStatus is already terminal', async () => {
    const fetchStatus = vi.fn(async () => snapshot({ status: 'succeeded', reload_web: false }));
    const onStatus = vi.fn();
    const onTerminal = vi.fn();
    const session = createModuleOpProgressSession({
      fetchStatus,
      isActive: () => true,
      onStatus,
      onTerminal,
      onTimeout: () => undefined,
    });

    await session.watch('job-1');

    expect(fetchStatus).toHaveBeenCalledTimes(1);
    expect(onTerminal).toHaveBeenCalledTimes(1);
    expect(onTips).not.toHaveBeenCalled();
    expect(subscribeModuleOp).not.toHaveBeenCalled();
  });

  it('refreshes on tip and does not poll while streaming', async () => {
    const statuses = [
      snapshot({ status: 'queued' }),
      snapshot({ status: 'dispatching' }),
      snapshot({ status: 'succeeded' }),
    ];
    const fetchStatus = vi.fn(async () => statuses.shift()!);
    const onStatus = vi.fn();
    const onTerminal = vi.fn();
    onTips.mockImplementation(async (_stream, callback: () => Promise<void>) => {
      await callback();
      await vi.advanceTimersByTimeAsync(80);
    });

    const session = createModuleOpProgressSession({
      fetchStatus,
      isActive: () => true,
      onStatus,
      onTerminal,
      onTimeout: () => undefined,
    });

    await session.watch('job-2');

    expect(subscribeModuleOp).toHaveBeenCalledWith('job-2', expect.any(AbortSignal));
    expect(onTerminal).toHaveBeenCalled();
    expect(fetchStatus.mock.calls.length).toBeGreaterThanOrEqual(2);
    await vi.advanceTimersByTimeAsync(5_000);
    // No fallback poll after terminal via tip.
    expect(fetchStatus.mock.calls.length).toBeLessThan(5);
  });

  it('starts short-backoff poll when tip stream ends without terminal', async () => {
    const fetchStatus = vi.fn(async () => snapshot({ status: 'queued' }));
    onTips.mockResolvedValue(undefined);
    const session = createModuleOpProgressSession({
      fetchStatus,
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: () => undefined,
    });

    await session.watch('job-3');
    fetchStatus.mockClear();
    await vi.advanceTimersByTimeAsync(1000);
    expect(fetchStatus).toHaveBeenCalled();
    session.stop();
  });

  it('starts poll fallback when tip stream rejects', async () => {
    const fetchStatus = vi.fn(async () => snapshot({ status: 'dispatching' }));
    onTips.mockRejectedValue(new Error('stream down'));
    const session = createModuleOpProgressSession({
      fetchStatus,
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: () => undefined,
    });

    await session.watch('job-4');
    fetchStatus.mockClear();
    await vi.advanceTimersByTimeAsync(1000);
    expect(fetchStatus).toHaveBeenCalled();
    session.stop();
  });

  it('stop aborts tip and clears poll', async () => {
    const fetchStatus = vi.fn(async () => snapshot({ status: 'queued' }));
    let capturedSignal: AbortSignal | undefined;
    onTips.mockImplementation(async (_stream, _cb, signal?: AbortSignal) => {
      capturedSignal = signal;
      await new Promise(() => undefined);
    });
    const session = createModuleOpProgressSession({
      fetchStatus,
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: () => undefined,
    });

    const done = session.watch('job-5');
    await Promise.resolve();
    await Promise.resolve();
    session.stop();
    expect(capturedSignal?.aborted).toBe(true);
    fetchStatus.mockClear();
    await vi.advanceTimersByTimeAsync(5_000);
    expect(fetchStatus).not.toHaveBeenCalled();
    await Promise.race([done, Promise.resolve()]);
  });

  it('times out fallback poll after 10 minutes', async () => {
    const fetchStatus = vi.fn(async () => snapshot({ status: 'queued' }));
    const onTimeout = vi.fn();
    onTips.mockResolvedValue(undefined);
    const session = createModuleOpProgressSession({
      fetchStatus,
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout,
    });

    await session.watch('job-6');
    await vi.advanceTimersByTimeAsync(10 * 60 * 1000 + 5_000);
    expect(onTimeout).toHaveBeenCalled();
    session.stop();
  });
});
