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

  it('reloads when terminal boot status requests reload_web', async () => {
    const reload = vi.fn();
    vi.stubGlobal('location', { reload });
    const fetchStatus = vi.fn(async () => snapshot({ status: 'succeeded', reload_web: true }));
    const session = createModuleOpProgressSession({
      fetchStatus,
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: () => undefined,
    });

    await session.watch('job-reload');
    expect(reload).toHaveBeenCalled();
    vi.unstubAllGlobals();
  });

  it('no-ops for empty job ids', async () => {
    const fetchStatus = vi.fn();
    const session = createModuleOpProgressSession({
      fetchStatus,
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: () => undefined,
    });
    await session.watch('   ');
    expect(fetchStatus).not.toHaveBeenCalled();
    expect(onTips).not.toHaveBeenCalled();
  });

  it('returns early when dialog becomes inactive during boot', async () => {
    let active = true;
    const fetchStatus = vi.fn(async () => {
      active = false;
      return snapshot({ status: 'queued' });
    });
    const session = createModuleOpProgressSession({
      fetchStatus,
      isActive: () => active,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: () => undefined,
    });
    await session.watch('job-inactive-boot');
    expect(onTips).not.toHaveBeenCalled();
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
    expect(fetchStatus.mock.calls.length).toBeLessThan(5);
  });

  it('debounces tip refreshes and reports tip refresh hard errors', async () => {
    const onHardError = vi.fn();
    const fetchStatus = vi
      .fn()
      .mockResolvedValueOnce(snapshot({ status: 'queued' }))
      .mockRejectedValueOnce(new Error('status exploded'));
    onTips.mockImplementation(async (_stream, callback: () => Promise<void>) => {
      await callback();
      await callback();
      await vi.advanceTimersByTimeAsync(80);
    });
    const session = createModuleOpProgressSession({
      fetchStatus,
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: () => undefined,
      onHardError,
    });
    await session.watch('job-tip-err');
    expect(onHardError).toHaveBeenCalledWith('status exploded');
  });

  it('notifies transient tip refresh errors once', async () => {
    const onTransientNetworkError = vi.fn();
    const fetchStatus = vi
      .fn()
      .mockResolvedValueOnce(snapshot({ status: 'queued' }))
      .mockRejectedValueOnce(new Error('Failed to fetch'))
      .mockRejectedValueOnce(new Error('NetworkError again'));
    onTips.mockImplementation(async (_stream, callback: () => Promise<void>) => {
      await callback();
      await vi.advanceTimersByTimeAsync(80);
      await callback();
      await vi.advanceTimersByTimeAsync(80);
    });
    const session = createModuleOpProgressSession({
      fetchStatus,
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: () => undefined,
      onTransientNetworkError,
    });
    await session.watch('job-tip-transient');
    expect(onTransientNetworkError).toHaveBeenCalledTimes(1);
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

  it('respects retryAfterMs and reaches terminal via poll', async () => {
    const onTerminal = vi.fn();
    const fetchStatus = vi
      .fn()
      .mockResolvedValueOnce(snapshot({ status: 'queued', retryAfterMs: 1500 }))
      .mockResolvedValueOnce(snapshot({ status: 'queued', retryAfterMs: 1500 }))
      .mockResolvedValueOnce(snapshot({ status: 'failed' }));
    onTips.mockResolvedValue(undefined);
    const session = createModuleOpProgressSession({
      fetchStatus,
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal,
      onTimeout: () => undefined,
    });
    await session.watch('job-retry-after');
    await vi.advanceTimersByTimeAsync(1500);
    await vi.advanceTimersByTimeAsync(1500);
    expect(onTerminal).toHaveBeenCalled();
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

  it('reports boot hard and transient errors then continues watching', async () => {
    const onHardError = vi.fn();
    const onTransientNetworkError = vi.fn();
    const fetchStatus = vi
      .fn()
      .mockRejectedValueOnce(new Error('boom'))
      .mockResolvedValue(snapshot({ status: 'queued' }));
    onTips.mockResolvedValue(undefined);
    const session = createModuleOpProgressSession({
      fetchStatus,
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: () => undefined,
      onHardError,
      onTransientNetworkError,
    });
    await session.watch('job-boot-hard');
    expect(onHardError).toHaveBeenCalledWith('boom');
    expect(onTips).toHaveBeenCalled();

    onHardError.mockClear();
    fetchStatus.mockReset();
    fetchStatus.mockRejectedValueOnce(new Error('ERR_CONNECTION_REFUSED')).mockResolvedValue(snapshot({ status: 'queued' }));
    onTips.mockResolvedValue(undefined);
    await session.watch('job-boot-transient');
    expect(onTransientNetworkError).toHaveBeenCalled();
  });

  it('reports poll hard and transient errors', async () => {
    const onHardError = vi.fn();
    const onTransientNetworkError = vi.fn();
    const fetchStatus = vi
      .fn()
      .mockResolvedValueOnce(snapshot({ status: 'queued' }))
      .mockRejectedValueOnce(new Error('Load failed'))
      .mockRejectedValueOnce(new Error('permanent'))
      .mockResolvedValue(snapshot({ status: 'queued' }));
    onTips.mockResolvedValue(undefined);
    const session = createModuleOpProgressSession({
      fetchStatus,
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: () => undefined,
      onHardError,
      onTransientNetworkError,
    });
    await session.watch('job-poll-err');
    await vi.advanceTimersByTimeAsync(1000);
    await vi.advanceTimersByTimeAsync(1500);
    expect(onTransientNetworkError).toHaveBeenCalledTimes(1);
    expect(onHardError).toHaveBeenCalledWith('permanent');
    session.stop();
  });

  it('stop aborts tip and resolves the watch promise', async () => {
    const fetchStatus = vi.fn(async () => snapshot({ status: 'queued' }));
    let capturedSignal: AbortSignal | undefined;
    onTips.mockImplementation(async (_stream, _cb, signal?: AbortSignal) => {
      capturedSignal = signal;
      await new Promise<void>((resolve) => {
        signal?.addEventListener('abort', () => resolve(), { once: true });
      });
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
    await expect(done).resolves.toBeUndefined();
  });

  it('times out while the tip stream stays open', async () => {
    const onTimeout = vi.fn();
    const fetchStatus = vi.fn(async () => snapshot({ status: 'queued' }));
    onTips.mockImplementation(async (_stream, _cb, signal?: AbortSignal) => {
      await new Promise<void>((resolve) => {
        signal?.addEventListener('abort', () => resolve(), { once: true });
      });
    });
    const session = createModuleOpProgressSession({
      fetchStatus,
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout,
    });
    const done = session.watch('job-tip-timeout');
    await Promise.resolve();
    await Promise.resolve();
    await vi.advanceTimersByTimeAsync(10 * 60 * 1000);
    expect(onTimeout).toHaveBeenCalled();
    await expect(done).resolves.toBeUndefined();
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

  it('stops polling when the dialog becomes inactive', async () => {
    let active = true;
    const fetchStatus = vi.fn(async () => snapshot({ status: 'queued' }));
    onTips.mockResolvedValue(undefined);
    const session = createModuleOpProgressSession({
      fetchStatus,
      isActive: () => active,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: () => undefined,
    });
    await session.watch('job-inactive-poll');
    await vi.advanceTimersByTimeAsync(0);
    expect(fetchStatus).toHaveBeenCalled();
    active = false;
    fetchStatus.mockClear();
    await vi.advanceTimersByTimeAsync(5_000);
    expect(fetchStatus).not.toHaveBeenCalled();
  });

  it('ignores deadline timeout after the dialog becomes inactive', async () => {
    let active = true;
    const onTimeout = vi.fn();
    const fetchStatus = vi.fn(async () => snapshot({ status: 'queued' }));
    onTips.mockImplementation(async (_stream, _cb, signal?: AbortSignal) => {
      await new Promise<void>((resolve) => {
        signal?.addEventListener('abort', () => resolve(), { once: true });
      });
    });
    const session = createModuleOpProgressSession({
      fetchStatus,
      isActive: () => active,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout,
    });
    const done = session.watch('job-timeout-inactive');
    await Promise.resolve();
    await Promise.resolve();
    active = false;
    await vi.advanceTimersByTimeAsync(10 * 60 * 1000);
    expect(onTimeout).not.toHaveBeenCalled();
    session.stop();
    await expect(done).resolves.toBeUndefined();
  });

  it('skips applyFetched when the session is inactive up front', async () => {
    const fetchStatus = vi.fn(async () => snapshot({ status: 'queued' }));
    const session = createModuleOpProgressSession({
      fetchStatus,
      isActive: () => false,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: () => undefined,
    });
    await session.watch('job-inactive-apply');
    expect(fetchStatus).not.toHaveBeenCalled();
    expect(onTips).not.toHaveBeenCalled();
  });

  it('swallows boot errors that arrive after the dialog closes', async () => {
    let active = true;
    const onHardError = vi.fn();
    const fetchStatus = vi.fn(async () => {
      active = false;
      throw new Error('boot after close');
    });
    const session = createModuleOpProgressSession({
      fetchStatus,
      isActive: () => active,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: () => undefined,
      onHardError,
    });
    await session.watch('job-boot-closed');
    expect(onHardError).not.toHaveBeenCalled();
    expect(onTips).not.toHaveBeenCalled();
  });

  it('ignores poll errors after stop bumps the session generation', async () => {
    const onHardError = vi.fn();
    let session!: ReturnType<typeof createModuleOpProgressSession>;
    const fetchStatus = vi
      .fn()
      .mockResolvedValueOnce(snapshot({ status: 'queued' }))
      .mockImplementationOnce(async () => {
        session.stop();
        throw new Error('poll after stop');
      });
    onTips.mockResolvedValue(undefined);
    session = createModuleOpProgressSession({
      fetchStatus,
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: () => undefined,
      onHardError,
    });
    await session.watch('job-poll-stopped');
    await vi.advanceTimersByTimeAsync(1000);
    expect(onHardError).not.toHaveBeenCalled();
  });

  it('drops debounced tip refreshes after stop advances the generation', async () => {
    let session!: ReturnType<typeof createModuleOpProgressSession>;
    const fetchStatus = vi.fn(async () => snapshot({ status: 'queued' }));
    onTips.mockImplementation(async (_stream, callback: () => Promise<void>) => {
      await callback();
      await vi.advanceTimersByTimeAsync(40);
      session.stop();
      await vi.advanceTimersByTimeAsync(40);
    });
    session = createModuleOpProgressSession({
      fetchStatus,
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: () => undefined,
    });
    await session.watch('job-tip-stale-generation');
    // boot fetch only — debounced tip refresh discarded after stop()
    expect(fetchStatus).toHaveBeenCalledTimes(1);
  });

  it('ignores tip-refresh failures after stop', async () => {
    const onHardError = vi.fn();
    let session!: ReturnType<typeof createModuleOpProgressSession>;
    const fetchStatus = vi
      .fn()
      .mockResolvedValueOnce(snapshot({ status: 'queued' }))
      .mockImplementationOnce(async () => {
        session.stop();
        throw new Error('tip refresh after stop');
      });
    onTips.mockImplementation(async (_stream, callback: () => Promise<void>) => {
      await callback();
      await vi.advanceTimersByTimeAsync(80);
    });
    session = createModuleOpProgressSession({
      fetchStatus,
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: () => undefined,
      onHardError,
    });
    await session.watch('job-tip-refresh-stop');
    expect(onHardError).not.toHaveBeenCalled();
  });

  it('covers optional hard-error hooks and empty error messages', async () => {
    const onHardError = vi.fn();
    const fetchStatus = vi
      .fn()
      .mockRejectedValueOnce('')
      .mockResolvedValueOnce(snapshot({ status: 'queued' }))
      .mockRejectedValueOnce('')
      .mockResolvedValue(snapshot({ status: 'queued' }));
    onTips.mockResolvedValue(undefined);
    const withHook = createModuleOpProgressSession({
      fetchStatus,
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: () => undefined,
      onHardError,
    });
    await withHook.watch('job-empty-error-message');
    expect(onHardError).toHaveBeenCalledWith('Failed to get status');

    const withoutHook = createModuleOpProgressSession({
      fetchStatus,
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: () => undefined,
    });
    await withoutHook.watch('job-optional-hard-error');
    expect(onTips).toHaveBeenCalled();
  });

  it('treats empty/undefined terminal markers and job ids safely', async () => {
    const onTerminal = vi.fn();
    const fetchStatus = vi.fn(async () => snapshot({ status: '' as string }));
    const session = createModuleOpProgressSession({
      fetchStatus,
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal,
      onTimeout: () => undefined,
    });
    onTips.mockResolvedValue(undefined);
    await session.watch(undefined as unknown as string);
    expect(fetchStatus).not.toHaveBeenCalled();
    await session.watch('job-empty-status');
    expect(onTerminal).not.toHaveBeenCalled();
  });

  it('swallows progress-hook throws from boot and poll paths', async () => {
    const onHardError = vi.fn(() => {
      throw new Error('notify threw');
    });
    const fetchStatus = vi
      .fn()
      .mockRejectedValueOnce(new Error('boot hard'))
      .mockResolvedValueOnce(snapshot({ status: 'queued' }))
      .mockRejectedValueOnce(new Error('poll hard'))
      .mockResolvedValue(snapshot({ status: 'queued' }));
    onTips.mockResolvedValue(undefined);
    const session = createModuleOpProgressSession({
      fetchStatus,
      isActive: () => true,
      onStatus: () => undefined,
      onTerminal: () => undefined,
      onTimeout: () => undefined,
      onHardError,
    });
    await expect(session.watch('job-hook-throw')).resolves.toBeUndefined();
    await vi.advanceTimersByTimeAsync(1000);
    expect(onHardError).toHaveBeenCalled();
    session.stop();
  });
});
