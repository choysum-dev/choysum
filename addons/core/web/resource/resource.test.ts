// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, it } from 'vitest';
import {
  clearResourceDeclarations,
  defineAction,
  defineMenu,
  defineModelActions,
  defineRoute,
  getResourceDeclaration,
  getResourceDeclarationFromMeta,
  listResourceDeclarations,
} from './index';

describe('resource declaration helpers', () => {
  beforeEach(() => {
    clearResourceDeclarations();
  });

  it('defineRoute maps title to pageTitle and injects resourceId and routeSequence', () => {
    const route = defineRoute('auth.route.user_list', {
      actions: ['auth.action.user_create'],
      defaultRoles: ['base.user', ' base.user '],
      override: true,
      requires: [{ model: 'auth.User' }, { model: 'auth.User', method: 'Browse' }, { model: ' ' } as any],
      sequence: 30,
      path: '/auth/users',
      name: 'UserList',
      title: 'User List',
    } as any);

    expect((route as any).meta?.resourceId).toBe('auth.route.user_list');
    expect((route as any).meta?.routeSequence).toBe(30);
    expect((route as any).meta?.pageTitle).toBe('User List');
    expect(getResourceDeclarationFromMeta((route as any).meta)).toEqual({
      id: 'auth.route.user_list',
      kind: 'route',
      title: 'User List',
      sequence: 30,
      path: '/auth/users',
      actions: ['auth.action.user_create'],
      requires: [
        { kind: 'rpc', model: 'auth.User' },
        { kind: 'rpc', model: 'auth.User', method: 'Browse' },
      ],
      defaultRoles: ['base.user'],
      override: true,
    });
    expect(getResourceDeclaration('auth.route.user_list')).toEqual(getResourceDeclarationFromMeta((route as any).meta));
    expect((route as any).actions).toBeUndefined();
  });

  it('defineRoute preserves explicit pageTitle override', () => {
    const route = defineRoute('auth.route.user_detail', {
      path: '/auth/users/:id',
      title: 'User Details',
      meta: { pageTitle: 'User Details - Dynamic Override' },
    } as any);

    expect((route as any).meta?.pageTitle).toBe('User Details - Dynamic Override');
  });

  it('defineRoute strips actions from returned route config', () => {
    const route = defineRoute('auth.route.user_detail', {
      path: '/auth/users/:id',
      actions: ['auth.action.user_edit', 'auth.action.user_delete'],
    } as any);

    expect((route as any).meta?.resourceId).toBe('auth.route.user_detail');
    expect((route as any).actions).toBeUndefined();
  });

  it('defineMenu maps sequence to order and injects resourceId', () => {
    const menu = defineMenu('auth.menu.user_list', {
      title: 'User List',
      defaultRoles: ['base.user'],
      requires: [{ model: 'auth.User' }],
      sequence: 30,
      path: '/auth/users',
    });

    expect(menu.id).toBe('auth.menu.user_list');
    expect(menu.order).toBe(30);
    expect((menu.meta as any)?.resourceId).toBe('auth.menu.user_list');
    expect(getResourceDeclarationFromMeta(menu.meta as any)).toEqual({
      id: 'auth.menu.user_list',
      kind: 'menu',
      title: 'User List',
      sequence: 30,
      path: '/auth/users',
      parentMenu: undefined,
      requires: [{ kind: 'rpc', model: 'auth.User' }],
      defaultRoles: ['base.user'],
      override: false,
    });
  });

  it('defineAction returns stable id and registers explicit action declaration', () => {
    const action = defineAction('auth.action.user_export', {
      defaultRoles: ['base.user'],
      requires: [{ model: 'auth.User', method: 'Export' }],
      title: 'Export Users',
    });

    expect(action).toBe('auth.action.user_export');
    expect(getResourceDeclaration('auth.action.user_export')).toEqual({
      id: 'auth.action.user_export',
      kind: 'action',
      title: 'Export Users',
      sequence: undefined,
      requires: [{ kind: 'rpc', model: 'auth.User', method: 'Export' }],
      defaultRoles: ['base.user'],
      override: false,
    });
  });

  it('lists registered declarations for route menu and action uniformly', () => {
    defineRoute('demo.route.list', {
      path: '/demo',
      requires: [{ model: 'demo.Model' }],
    } as any);
    defineMenu('demo.menu.list', {
      path: '/demo',
    });
    defineAction('demo.action.open', {
      title: 'Open',
    });

    expect(listResourceDeclarations().map(item => item.id)).toEqual(['demo.route.list', 'demo.menu.list', 'demo.action.open']);
  });

  it('defineModelActions accepts display config and generates snake ids', () => {
    const actions = defineModelActions('auth.UserRole', {
      entityTitle: 'User Role',
      titles: { delete: 'Deactivate User Role' },
      exclude: ['delete'],
    });

    expect(actions.create).toBe('auth.action.user_role_create');
    expect(actions.edit).toBe('auth.action.user_role_edit');
    expect(actions.copy).toBe('auth.action.user_role_copy');
    expect(actions.delete).toBeUndefined();
  });

  it('defineModelActions returns empty map on invalid model id', () => {
    expect(defineModelActions('invalid')).toEqual({});
    expect(defineModelActions('auth.')).toEqual({});
    expect(defineModelActions('.User')).toEqual({});
  });
});
