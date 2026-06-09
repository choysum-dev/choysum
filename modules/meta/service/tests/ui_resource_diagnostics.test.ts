// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import IrUiResource from '@/meta/service/models/ir_ui_resource';
import IrUiResourceRouteAction from '@/meta/service/models/ir_ui_resource_route_action';

const RR_CACHE_KEY = Symbol.for('choysum.recordrule.cache');
const FR_CACHE_KEY = Symbol.for('choysum.fieldrule.cache');
const CTX_OVERRIDE_KEY = Symbol.for('choysum.ctx.override');
const CTX_FROZEN_KEY = Symbol.for('choysum.ctx.frozen');

function ensureRequestContext(): any {
  const root: any = (globalThis as any).$choysum;
  if (!root) {
    throw new Error('missing global $choysum; meta ui resource tests must run under the QuickJS-first harness');
  }

  if (!root.request) root.request = {};
  if (!root.request.context) root.request.context = {};

  const jsCtx = root.request.context;
  if (!jsCtx.ctx) jsCtx.ctx = {};
  if (!jsCtx.req) jsCtx.req = {};
  if (!jsCtx.identity) jsCtx.identity = {};

  return jsCtx;
}

function resetRequestContext(): void {
  const jsCtx = ensureRequestContext();
  jsCtx.ctx = {};
  jsCtx.req = {
    depth: 0,
    companyMode: 'skip',
    recordRuleMode: 'skip',
    fieldRuleMode: 'skip',
  };
  jsCtx.identity = { userId: 'admin' };

  delete (jsCtx as any)[RR_CACHE_KEY];
  delete (jsCtx as any)[FR_CACHE_KEY];
  delete (jsCtx as any)[CTX_OVERRIDE_KEY];
  delete (jsCtx as any)[CTX_FROZEN_KEY];
}

function uid(prefix: string): string {
  const xid = (globalThis as any).$choysum?.xid?.New?.();
  const value = typeof xid === 'string' && xid.trim() ? xid.trim() : String(Date.now());
  return `${prefix}_${value}`;
}

test('meta.IrUiResource GetEffectiveDeclarations returns normalized declaration views', async () => {
  resetRequestContext();

  const moduleName = uid('mod');
  const rootMenu = await IrUiResource.Create(
    {
      Name: uid('root_menu'),
      Type: 'MENU',
      Title: 'Root',
      Module: moduleName,
      UiPath: '/root',
      DefaultRoles: ['base.user', ' base.user '],
    } as any,
    ['Id', 'Name'] as any
  );

  const childMenu = await IrUiResource.Create(
    {
      Name: uid('child_menu'),
      Type: 'MENU',
      Title: 'Users',
      Module: moduleName,
      UiPath: '/users',
      ParentId: { Id: rootMenu.Id } as any,
      Requires: ['rpc:/auth.User/*', 'rpc:/auth.User/*'],
    } as any,
    ['Id', 'Name'] as any
  );

  const route = await IrUiResource.Create(
    {
      Name: uid('route'),
      Type: 'ROUTE',
      Title: 'User List',
      Sequence: 20,
      Module: moduleName,
      UiPath: '/users',
      Requires: ['rpc:/auth.User/Browse', 'rpc:/auth.User/Browse'],
      DefaultRoles: ['base.user'],
    } as any,
    ['Id', 'Name'] as any
  );

  const action = await IrUiResource.Create(
    {
      Name: uid('action'),
      Type: 'ACTION',
      Title: 'Export Users',
      Module: moduleName,
      Requires: ['rpc:/auth.User/Export'],
    } as any,
    ['Id', 'Name'] as any
  );

  await IrUiResourceRouteAction.Create(
    {
      RouteUiResourceId: { Id: route.Id } as any,
      ActionUiResourceId: { Id: action.Id } as any,
    } as any,
    ['Id'] as any
  );

  const result = await IrUiResource.GetEffectiveDeclarations({ module: moduleName });

  expect(result.total).toBe(4);
  expect(result.filtered).toBe(4);
  expect(result.returned).toBe(4);

  const declById = Object.fromEntries(result.declarations.map(item => [item.id, item]));

  expect(declById[rootMenu.Name]).toEqual({
    id: rootMenu.Name,
    kind: 'menu',
    title: 'Root',
    sequence: undefined,
    path: '/root',
    parentMenu: undefined,
    requires: [],
    defaultRoles: ['base.user'],
    override: false,
    module: moduleName,
    application: undefined,
  });

  expect(declById[childMenu.Name]).toEqual({
    id: childMenu.Name,
    kind: 'menu',
    title: 'Users',
    sequence: undefined,
    path: '/users',
    parentMenu: rootMenu.Name,
    requires: [{ kind: 'rpc', model: 'auth.User' }],
    defaultRoles: [],
    override: false,
    module: moduleName,
    application: undefined,
  });

  expect(declById[route.Name]).toEqual({
    id: route.Name,
    kind: 'route',
    title: 'User List',
    sequence: 20,
    path: '/users',
    actions: [action.Name],
    requires: [{ kind: 'rpc', model: 'auth.User', method: 'Browse' }],
    defaultRoles: ['base.user'],
    override: false,
    module: moduleName,
    application: undefined,
  });

  expect(declById[action.Name]).toEqual({
    id: action.Name,
    kind: 'action',
    title: 'Export Users',
    sequence: undefined,
    path: undefined,
    requires: [{ kind: 'rpc', model: 'auth.User', method: 'Export' }],
    defaultRoles: [],
    override: false,
    module: moduleName,
    application: undefined,
  });
});
