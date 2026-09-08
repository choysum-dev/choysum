// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { PermissionState } from '@/auth/web/permission';
import { usePermission } from './usePermission';

function makeState(routes: string[], menus: string[], actions: string[]): PermissionState {
  return {
    permStateVersion: 1,
    byCompany: {
      '*': { ui: { routes, menus, actions } },
    },
  };
}

test('usePermission: computes ctx from identity.metadata', () => {
  const mockAuthStore = {
    identity: {
      metadata: {
        activeCompanyId: 'c1',
        enabledCompanyIds: ['c1', 'c2'],
      },
    },
    permissionState: makeState([], [], []),
  };

  const perm = usePermission({ getAuthStore: () => mockAuthStore as any });

  expect(perm.ctx.value).toEqual({
    activeCompanyId: 'c1',
    enabledCompanyIds: ['c1', 'c2'],
  });
});

test('usePermission: delegates canRoute/canMenu/hasAction to real permission helpers', () => {
  const permissionState = makeState(['auth.route.other'], ['auth.menu.user_list'], ['auth.action.user_edit']);
  const mockAuthStore = {
    identity: {
      metadata: {
        activeCompanyId: 'c9',
        enabledCompanyIds: ['c9'],
      },
    },
    permissionState,
  };

  const perm = usePermission({ getAuthStore: () => mockAuthStore as any });

  expect(perm.canRoute('auth.route.user_list')).toBe(false);
  expect(perm.canMenu('auth.menu.user_list')).toBe(true);
  expect(perm.hasAction('auth.action.user_edit')).toBe(true);
  expect(perm.permissionState).toBe(permissionState);
});
