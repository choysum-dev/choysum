// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createModuleKanbanOpProgressHooks } from './moduleKanbanOpProgress';

function fnRecorder() {
  const rec: any = Object.assign(
    (...args: unknown[]) => {
      rec.calls.push(args);
    },
    { calls: [] as unknown[][] }
  );
  return rec;
}

function asyncFnRecorder(result: unknown) {
  const rec: any = Object.assign(
    async (...args: unknown[]) => {
      rec.calls.push(args);
      return result;
    },
    { calls: [] as unknown[][] }
  );
  return rec;
}

function valueFnRecorder(result: unknown) {
  const rec: any = Object.assign(
    (...args: unknown[]) => {
      rec.calls.push(args);
      return result;
    },
    { calls: [] as unknown[][] }
  );
  return rec;
}

test('createModuleKanbanOpProgressHooks: wires status, terminal, timeout, and error callbacks', async () => {
  const setOpStatus = fnRecorder();
  const setDialogStep = fnRecorder();
  const warn = fnRecorder();
  const error = fnRecorder();
  const fetchStatus = asyncFnRecorder({ status: 'queued' });
  const jobStillRunning = valueFnRecorder('Job is still running in the background; refresh later');
  const serviceRestarting = valueFnRecorder('Service is restarting; status will retry automatically');
  const failedToGetStatus = valueFnRecorder('Failed to get status');

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
  expect(await hooks.fetchStatus('job-1')).toEqual({ status: 'queued' });

  hooks.onStatus({ status: 'dispatching' });
  expect(setOpStatus.calls[setOpStatus.calls.length - 1]).toEqual([{ status: 'dispatching' }]);

  hooks.onTerminal({ status: 'succeeded' });
  expect(setDialogStep.calls[setDialogStep.calls.length - 1]).toEqual(['result']);

  hooks.onTimeout();
  expect(setDialogStep.calls[setDialogStep.calls.length - 1]).toEqual(['result']);
  expect(setOpStatus.calls[setOpStatus.calls.length - 1]).toEqual([
    {
      status: 'dispatching',
      resultStatus: undefined,
    },
  ]);
  expect(jobStillRunning.calls.length).toBe(1);
  expect(warn.calls[warn.calls.length - 1]).toEqual(['Job is still running in the background; refresh later']);

  hooks.onTransientNetworkError?.();
  expect(serviceRestarting.calls.length).toBe(1);
  expect(warn.calls[warn.calls.length - 1]).toEqual(['Service is restarting; status will retry automatically']);

  hooks.onHardError?.('boom');
  expect(error.calls[error.calls.length - 1]).toEqual(['boom']);
  expect(failedToGetStatus.calls.length).toBe(0);
  hooks.onHardError?.('');
  expect(failedToGetStatus.calls.length).toBe(1);
  expect(error.calls[error.calls.length - 1]).toEqual(['Failed to get status']);
  hooks.onHardError?.('Failed to get status');
  expect(failedToGetStatus.calls.length).toBe(2);
  expect(error.calls[error.calls.length - 1]).toEqual(['Failed to get status']);
});
