// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { canRoute, canMenu, hasAction, type PermissionState } from './permission';

function makeState(byCompany: PermissionState['byCompany']): PermissionState {
  return {
    permStateVersion: 1,
    byCompany,
  };
}

describe('permission helpers', () => {
  it('allows when wildcard is present', () => {
    const state = makeState({
      '*': {
        ui: {
          routes: ['*'],
          menus: ['*'],
          actions: ['*'],
        },
      },
    });

    const ctx = { activeCompanyId: 'c1', enabledCompanyIds: ['c1'] };
    expect(canRoute('auth.route.user_list', state, ctx)).toBe(true);
    expect(canMenu('auth.menu.user_list', state, ctx)).toBe(true);
    expect(hasAction('auth.action.user_export', state, ctx)).toBe(true);
  });

  it('uses enabled company scope by default and active scope when requested', () => {
    const state = makeState({
      '*': { ui: { routes: [], menus: [], actions: [] } },
      c1: {
        ui: {
          routes: ['auth.route.company1'],
          menus: ['auth.menu.company1'],
          actions: ['auth.action.company1'],
        },
      },
      c2: {
        ui: {
          routes: ['auth.route.company2'],
          menus: ['auth.menu.company2'],
          actions: ['auth.action.company2'],
        },
      },
    });

    const ctx = { activeCompanyId: 'c1', enabledCompanyIds: ['c1', 'c2'] };

    expect(canRoute('auth.route.company2', state, ctx)).toBe(true);
    expect(canRoute('auth.route.company2', state, ctx, 'active')).toBe(false);

    expect(canMenu('auth.menu.company2', state, ctx)).toBe(true);
    expect(canMenu('auth.menu.company2', state, ctx, 'active')).toBe(false);

    expect(hasAction('auth.action.company2', state, ctx)).toBe(true);
    expect(hasAction('auth.action.company2', state, ctx, 'active')).toBe(false);
  });

  it('treats empty id as allowed and missing state as denied for non-empty id', () => {
    const state = makeState({
      '*': { ui: { routes: [], menus: [], actions: [] } },
    });
    const ctx = { activeCompanyId: 'c1', enabledCompanyIds: ['c1'] };

    expect(canRoute('', state, ctx)).toBe(true);
    expect(canMenu(undefined, state, ctx)).toBe(true);
    expect(hasAction('', state, ctx)).toBe(true);

    expect(canRoute('auth.route.x', null, ctx)).toBe(false);
    expect(canMenu('auth.menu.x', undefined, ctx)).toBe(false);
    expect(hasAction('auth.action.x', null, ctx)).toBe(false);
  });

  it('falls back to active company when enabledCompanyIds is empty', () => {
    const state = makeState({
      '*': { ui: { routes: [], menus: [], actions: [] } },
      c1: {
        ui: {
          routes: ['auth.route.c1'],
          menus: ['auth.menu.c1'],
          actions: ['auth.action.c1'],
        },
      },
    });

    const ctx = { activeCompanyId: 'c1', enabledCompanyIds: [] };

    expect(canRoute('auth.route.c1', state, ctx)).toBe(true);
    expect(canMenu('auth.menu.c1', state, ctx)).toBe(true);
    expect(hasAction('auth.action.c1', state, ctx)).toBe(true);
  });

  it('normalizes whitespace in permission snapshot ids', () => {
    const state = makeState({
      '*': {
        ui: {
          routes: ['  auth.route.trimmed  '],
          menus: ['  auth.menu.trimmed  '],
          actions: ['  auth.action.trimmed  '],
        },
      },
    });

    const ctx = { activeCompanyId: 'c1', enabledCompanyIds: ['c1'] };

    expect(canRoute('auth.route.trimmed', state, ctx)).toBe(true);
    expect(canMenu('auth.menu.trimmed', state, ctx)).toBe(true);
    expect(hasAction('auth.action.trimmed', state, ctx)).toBe(true);
  });
});
