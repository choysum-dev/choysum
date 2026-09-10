// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createFormController } from './formController';
import { handoffCache } from '@/web/web/query/utils/handoff';
import { asyncFnRecorder, fnRecorder } from '@/web/web/__tests__/mountApp';

test('formController > beginDisplay awaits field selection when export is empty', async () => {
  const awaitFieldSelection = asyncFnRecorder(async () => undefined);
  const exportFieldSelection = fnRecorder(() => [] as string[]);
  const execute = asyncFnRecorder(async () => ({
    kind: 'search',
    rows: [{ kind: 'record', key: '1', payload: { Id: '1', Name: 'n' }, raw: {} }],
    total: 1,
    ts: Date.now(),
  }));
  const store = {
    fullModelName: 'demo.Widget',
    storeId: 'demo.Widget',
    fieldsMetadata: { Name: { type: 'varchar' } },
    state: {},
    getContext: () => ({}),
  } as any;

  const controller = createFormController(store, {
    awaitFieldSelection: awaitFieldSelection as any,
    exportFieldSelection: exportFieldSelection as any,
    buildBrowseContext: (() => ({ model: 'demo.Widget', shape: 'browse', queryState: {} })) as any,
    buildPlan: (() => ({ main: { kind: 'browse', hash: 'h' } })) as any,
    execute: execute as any,
    flashRead: (() => undefined) as any,
  });

  await controller.beginDisplay('1');

  expect(exportFieldSelection.calls.length).toBeGreaterThan(0);
  expect(awaitFieldSelection.calls.length).toBe(1);
  expect(execute.calls.length).toBe(1);
  expect(typeof (execute.calls[0]?.[3] as any)?.createStoreByModel).toBe('function');
  expect(controller.vm.original).toMatchObject({ Id: '1', Name: 'n' });
});

test('formController > create submit uses default handoff cache', async () => {
  handoffCache.clear();
  const Create = asyncFnRecorder(async () => ({ Id: 'new-1', Name: 'created' }));
  const store = {
    fullModelName: 'demo.Widget',
    storeId: 'demo.Widget',
    fieldsMetadata: { Name: { type: 'varchar' } },
    state: {},
    getContext: () => ({}),
    Create,
  } as any;

  const controller = createFormController(store, {
    exportFieldSelection: (() => ['Name']) as any,
  });
  await controller.beginCreate({ Name: 'seed' });
  // Skip DefaultGet noise: force draft after beginCreate.
  controller.vm.draft = { Name: 'seed' } as any;

  await controller.submit();

  expect(Create.calls.length).toBe(1);
  expect(handoffCache.get('new-1')).toMatchObject({ Id: 'new-1', Name: 'created' });
  handoffCache.clear();
});
