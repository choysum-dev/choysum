// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getRouteActionsFromMeta } from '@/core/web/resource';
import { sandboxRoutes } from '../route/routes';
import { sandboxRecordActions, sandboxRecordOpenDetailAction } from '../views/sandbox_record_actions';
import { sandboxTagActions } from '../views/sandbox_tag_actions';

const recordActions = [
  'sandbox.action.sandbox_record_create',
  'sandbox.action.sandbox_record_edit',
  'sandbox.action.sandbox_record_delete',
  'sandbox.action.sandbox_record_copy',
  'sandbox.action.sandbox_record_open_detail',
];

const tagActions = [
  'sandbox.action.sandbox_tag_create',
  'sandbox.action.sandbox_tag_edit',
  'sandbox.action.sandbox_tag_delete',
  'sandbox.action.sandbox_tag_copy',
];

const viewPaths = {
  list: '/sandbox/records',
  kanban: '/sandbox/records/kanban',
  chart: '/sandbox/records/chart',
};

test('sandbox resource wiring: each record route declares expected actions', () => {
  const byName = Object.fromEntries(sandboxRoutes.map(route => [String(route.name), route]));

  for (const name of [
    'SandboxRecordList',
    'SandboxRecordKanban',
    'SandboxRecordChart',
    'SandboxRecordDetail',
    'SandboxRecordCreate',
  ]) {
    expect(getRouteActionsFromMeta(byName[name]?.meta as any)).toEqual(recordActions);
  }
});

test('sandbox resource wiring: tag create/detail routes declare expected actions', () => {
  const byName = Object.fromEntries(sandboxRoutes.map(route => [String(route.name), route]));

  for (const name of ['SandboxTagCreate', 'SandboxTagDetail']) {
    expect(getRouteActionsFromMeta(byName[name]?.meta as any)).toEqual(tagActions);
  }
  expect(byName.SandboxTagCreate?.path).toBe('sandbox/tags/new');
  expect(byName.SandboxTagDetail?.path).toBe('sandbox/tags/:id');
});

test('sandbox resource wiring: shared sandbox_record_actions matches route actions', () => {
  expect(sandboxRecordOpenDetailAction).toBe('sandbox.action.sandbox_record_open_detail');
  expect(sandboxRecordActions.create).toBe('sandbox.action.sandbox_record_create');
  expect(sandboxRecordActions.edit).toBe('sandbox.action.sandbox_record_edit');
  expect(sandboxRecordActions.copy).toBe('sandbox.action.sandbox_record_copy');
  expect(sandboxRecordActions.delete).toBe('sandbox.action.sandbox_record_delete');
});

test('sandbox resource wiring: shared sandbox_tag_actions matches route actions', () => {
  expect(sandboxTagActions.create).toBe('sandbox.action.sandbox_tag_create');
  expect(sandboxTagActions.edit).toBe('sandbox.action.sandbox_tag_edit');
  expect(sandboxTagActions.copy).toBe('sandbox.action.sandbox_tag_copy');
  expect(sandboxTagActions.delete).toBe('sandbox.action.sandbox_tag_delete');
});

test('sandbox resource wiring: list/kanban/chart routes expose view switcher meta', () => {
  const byName = Object.fromEntries(sandboxRoutes.map(route => [String(route.name), route]));

  expect(byName.SandboxRecordList?.meta).toMatchObject({
    viewKind: 'list',
    defaultGroups: [],
    viewPaths,
  });
  expect(byName.SandboxRecordKanban?.meta).toMatchObject({
    viewKind: 'kanban',
    defaultGroups: ['State'],
    viewPaths,
  });
  expect(byName.SandboxRecordChart?.meta).toMatchObject({
    viewKind: 'chart',
    defaultGroups: ['State', 'IsActive'],
    viewPaths,
  });
});
