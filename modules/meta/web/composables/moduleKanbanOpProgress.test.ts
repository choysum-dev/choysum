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
    const fetchStatus = vi.fn(async () => ({ status: 'queued' }));
    const jobStillRunning = vi.fn(() => 'Job is still running in the background; refresh later');
    const serviceRestarting = vi.fn(() => 'Service is restarting; status will retry automatically');
    const failedToGetStatus = vi.fn(() => 'Failed to get status');

    const hooks = createModuleKanbanOpProgressHooks({
      fetchStatus,
      isDialogOpen: () => true,
      setOpStatus,
      setDialogStep,
      warn,
      error,
      messages: {
        jobStillRunning,
        serviceRestarting,
        failedToGetStatus,
      },
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
    expect(jobStillRunning).toHaveBeenCalledTimes(1);
    expect(warn).toHaveBeenCalledWith('Job is still running in the background; refresh later');

    hooks.onTransientNetworkError?.();
    expect(serviceRestarting).toHaveBeenCalledTimes(1);
    expect(warn).toHaveBeenCalledWith('Service is restarting; status will retry automatically');

    hooks.onHardError?.('boom');
    expect(error).toHaveBeenCalledWith('boom');
    expect(failedToGetStatus).not.toHaveBeenCalled();
    hooks.onHardError?.('');
    expect(failedToGetStatus).toHaveBeenCalledTimes(1);
    expect(error).toHaveBeenCalledWith('Failed to get status');
  });
});
