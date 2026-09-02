// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';
import { createModuleKanbanOpProgressHooks } from './moduleKanbanOpProgress';

describe('createModuleKanbanOpProgressHooks', () => {
  it('wires status, terminal, timeout, and error callbacks', async () => {
    const setOpStatus = vi.fn();
    const setDialogStep = vi.fn();
    const warn = vi.fn();
    const error = vi.fn();
    const t = vi.fn((message: string) => `t:${message}`);
    const fetchStatus = vi.fn(async () => ({ status: 'queued' }));

    const hooks = createModuleKanbanOpProgressHooks({
      fetchStatus,
      isDialogOpen: () => true,
      setOpStatus,
      setDialogStep,
      warn,
      error,
      t,
    });

    expect(hooks.isActive()).toBe(true);
    await expect(hooks.fetchStatus('job-1')).resolves.toEqual({ status: 'queued' });

    hooks.onStatus({ status: 'dispatching' });
    expect(setOpStatus).toHaveBeenCalledWith({ status: 'dispatching' });

    hooks.onTerminal({ status: 'succeeded' });
    expect(setDialogStep).toHaveBeenCalledWith('result');

    hooks.onTimeout();
    expect(setDialogStep).toHaveBeenLastCalledWith('result');
    expect(setOpStatus).toHaveBeenLastCalledWith({
      status: 'dispatching',
      resultStatus: undefined,
    });
    expect(warn).toHaveBeenCalledWith('t:Job is still running in the background; refresh later');

    hooks.onTransientNetworkError?.();
    expect(warn).toHaveBeenCalledWith('t:Service is restarting; status will retry automatically');

    hooks.onHardError?.('boom');
    expect(error).toHaveBeenCalledWith('boom');
    hooks.onHardError?.('');
    expect(error).toHaveBeenCalledWith('t:Failed to get status');
  });
});
