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
import { createTranslate } from '../../service/i18n';

describe('resource declaration helpers', () => {
  beforeEach(() => {
    clearResourceDeclarations();
  });

  it('defineRoute maps title to pageTitle and injects resourceId and routeSequence', () => {
    const route = defineRoute('auth.route.user_list', {
      actions: ['auth.action.user_create'],
      defaultRoles: ['base.user', ' base.user '],
      override: true,
      requires: [{ model: 'auth.User' }, { model: 'auth.User', method: 'Browse' }],
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

  it('normalizes term reference titles to English fallbacks plus metadata', () => {
    const { _lt } = createTranslate('auth');
    const reference = _lt('Users', { scope: 'auth.route.users' });
    const route = defineRoute('auth.route.users', {
      path: '/auth/users',
      title: reference,
    } as any);
    const menu = defineMenu('auth.menu.users', {
      path: '/auth/users',
      title: reference,
    });

    expect((route as any).meta.pageTitle).toBe('Users');
    expect((route as any).meta.pageTitleText).toEqual(reference);
    expect(menu.title).toBe('Users');
    expect(menu.titleText).toEqual(reference);
    expect(getResourceDeclaration('auth.route.users')).toMatchObject({
      title: 'Users',
      titleText: reference,
    });
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
    expect(getResourceDeclaration('auth.action.user_role_create')).toMatchObject({
      id: 'auth.action.user_role_create',
      kind: 'action',
      title: 'Create User Role',
      requires: [],
      defaultRoles: [],
      override: false,
    });
    expect(getResourceDeclaration('auth.action.user_role_edit')).toMatchObject({
      title: 'Edit User Role',
    });
  });

  it('defineModelActions registers term reference titles for model actions', () => {
    const { _lt } = createTranslate('base', { scope: 'web/views/CountryListView' });
    const reference = _lt('Country');
    const actions = defineModelActions('base.Country', { entityTitle: reference });

    expect(actions.create).toBe('base.action.country_create');
    expect(getResourceDeclaration('base.action.country_create')).toMatchObject({
      id: 'base.action.country_create',
      kind: 'action',
      title: 'Create Country',
      titleText: createTranslate('base', { scope: 'web/views/CountryListView' })._lt('Create Country'),
      requires: [],
      defaultRoles: [],
    });
  });

  it('defineModelActions returns empty map on invalid model id', () => {
    expect(defineModelActions('invalid')).toEqual({});
    expect(defineModelActions('auth.')).toEqual({});
    expect(defineModelActions('.User')).toEqual({});
  });

  it('throws on invalid resource requires, default roles, and actions', () => {
    expect(() =>
      defineRoute('demo.route.invalid_requires', {
        path: '/demo',
        requires: { model: 'demo.Model' } as any,
      } as any)
    ).toThrow('invalid_resource_requires');

    expect(() =>
      defineRoute('demo.route.blank_model', {
        path: '/demo',
        requires: [{ model: ' ' }],
      } as any)
    ).toThrow('invalid_resource_requires');

    expect(() =>
      defineRoute('demo.route.non_object_require', {
        path: '/demo',
        requires: ['auth.User'] as any,
      } as any)
    ).toThrow('invalid_resource_requires');

    expect(() =>
      defineRoute('demo.route.invalid_roles', {
        path: '/demo',
        defaultRoles: 'base.user' as any,
      } as any)
    ).toThrow('invalid_resource_default_roles');

    expect(() =>
      defineRoute('demo.route.non_string_roles', {
        path: '/demo',
        defaultRoles: [1] as any,
      } as any)
    ).toThrow('invalid_resource_default_roles');

    expect(() =>
      defineRoute('demo.route.invalid_actions', {
        path: '/demo',
        actions: 1 as any,
      } as any)
    ).toThrow('invalid_resource_actions');

    expect(() =>
      defineRoute('demo.route.non_string_actions', {
        path: '/demo',
        actions: [null] as any,
      } as any)
    ).toThrow('invalid_resource_actions');
  });

  it('accepts nullish list fields and blank/duplicate string entries as empty', () => {
    const route = defineRoute('demo.route.nullish_lists', {
      path: '/demo',
      requires: null as any,
      defaultRoles: null as any,
      actions: undefined,
      sequence: '',
    } as any);

    expect(getResourceDeclarationFromMeta((route as any).meta)).toMatchObject({
      requires: [],
      defaultRoles: [],
      actions: [],
      sequence: undefined,
    });
  });

  it('dedupes requires and rejects invalid sequence values', () => {
    const route = defineRoute('demo.route.dedupe_requires', {
      path: '/demo',
      requires: [
        { model: 'demo.Model', method: 'Browse' },
        { model: 'demo.Model', method: 'Browse' },
        { model: 'demo.Model' },
      ],
      defaultRoles: ['role.a', '', 'role.a', '  role.b  '],
      actions: ['a1', 'a1', '  a2  ', ''],
      sequence: 0,
    } as any);

    expect(getResourceDeclarationFromMeta((route as any).meta)).toMatchObject({
      requires: [
        { kind: 'rpc', model: 'demo.Model', method: 'Browse' },
        { kind: 'rpc', model: 'demo.Model' },
      ],
      defaultRoles: ['role.a', 'role.b'],
      actions: ['a1', 'a2'],
      sequence: 0,
    });

    expect(() =>
      defineRoute('demo.route.bad_sequence', {
        path: '/demo',
        sequence: -1,
      } as any)
    ).toThrow();

    expect(() =>
      defineRoute('demo.route.obj_sequence', {
        path: '/demo',
        sequence: true as any,
      } as any)
    ).toThrow();
  });
});
