// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  createModuleCatalogOpDrawerProgressHooks,
  mapOperationRowToSnapshot,
} from './moduleCatalogOpDrawerProgress';

test('mapOperationRowToSnapshot: maps status, errors, and reload flags', () => {
  expect(
    mapOperationRowToSnapshot({
      Status: 'Succeeded',
      ReloadTriggered: true,
      ReloadFailed: false,
      Phase: 'reload',
    }),
  ).toEqual({
    status: 'succeeded',
    summary: { message: 'reload' },
    resultStatus: 'SUCCEEDED',
    partialFailure: false,
    ReloadTriggered: true,
    ReloadFailed: false,
    ReloadWeb: true,
    errorDomain: undefined,
    errorCode: undefined,
  });

  expect(
    mapOperationRowToSnapshot({
      Status: 'failed',
      ErrorDomain: 'OPS',
      ErrorCode: 'E1',
      ErrorMessage: 'boom',
      ReloadTriggered: true,
      ReloadFailed: true,
    }),
  ).toMatchObject({
    status: 'failed',
    resultStatus: 'FAILED',
    summary: 'boom',
    ReloadWeb: false,
    errorDomain: 'OPS',
    errorCode: 'E1',
  });
});

test('mapOperationRowToSnapshot: sync PlanJson fills summary and partialFailure', () => {
  const planJson = JSON.stringify({
    baseRevision: 'sync',
    syncResult: { total: 3, success: 2, failed: 1, catalogRows: 10, origin: 'all' },
  });
  const snap = mapOperationRowToSnapshot({
    Action: 'sync',
    Status: 'succeeded',
    PlanJson: planJson,
  });
  expect(snap.resultStatus).toBe('SUCCEEDED');
  expect(snap.partialFailure).toBe(true);
  expect(snap.summary).toEqual({
    syncResult: { total: 3, success: 2, failed: 1, catalogRows: 10, origin: 'all' },
  });

  const ok = mapOperationRowToSnapshot({
    Action: 'sync',
    Status: 'succeeded',
    PlanJson: JSON.stringify({
      syncResult: { Total: 2, Success: 2, Failed: 0, CatalogRows: 5 },
    }),
  });
  expect(ok.partialFailure).toBe(false);
  expect((ok.summary as any).syncResult.catalogRows).toBe(5);
});

test('createModuleCatalogOpDrawerProgressHooks: terminal, timeout, and error bindings', () => {
  const steps: string[] = [];
  const statuses: unknown[] = [];
  const warns: string[] = [];
  const errors: string[] = [];
  const hooks = createModuleCatalogOpDrawerProgressHooks({
    fetchStatus: async () => ({ status: 'queued' }),
    isDrawerOpen: () => true,
    setOpStatus: s => statuses.push(s),
    setDrawerStep: s => steps.push(s),
    warn: m => warns.push(m),
    error: m => errors.push(m),
    messages: {
      jobStillRunning: () => 'still-running',
      serviceRestarting: () => 'restarting',
      failedToGetStatus: () => 'no-status',
    },
  });

  expect(hooks.isActive()).toBe(true);
  hooks.onStatus({ status: 'queued' });
  hooks.onTerminal({ status: 'succeeded' });
  hooks.onTimeout();
  hooks.onTransientNetworkError?.();
  hooks.onHardError?.('Failed to get status');
  hooks.onHardError?.('other');

  expect(steps).toEqual(['result', 'result']);
  expect(statuses.length).toBeGreaterThanOrEqual(2);
  expect(warns).toEqual(['still-running', 'restarting']);
  expect(errors).toEqual(['no-status', 'other']);
});
