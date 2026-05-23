// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';

let mockAuthStore: any;

const canRouteMock = vi.fn<(resourceId: string, permissionState: any, ctx: any) => boolean>(() => true);
const canMenuMock = vi.fn<(resourceId: string, permissionState: any, ctx: any) => boolean>(() => true);
const hasActionMock = vi.fn<(resourceId: string, permissionState: any, ctx: any) => boolean>(() => true);

vi.mock('@/auth/web/stores/auth', () => {
  return {
    useAuthStore: () => mockAuthStore,
  };
});

vi.mock('@/auth/web/permission', () => {
  return {
    canRoute: canRouteMock,
    canMenu: canMenuMock,
    hasAction: hasActionMock,
  };
});

describe('usePermission', () => {
  it('computes ctx from identity.metadata', async () => {
    mockAuthStore = {
      identity: {
        metadata: {
          activeCompanyId: 'c1',
          enabledCompanyIds: ['c1', 'c2'],
        },
      },
      permissionState: { any: 'state' },
    };

    const { usePermission } = await import('./usePermission');
    const perm = usePermission();

    expect(perm.ctx.value).toEqual({
      activeCompanyId: 'c1',
      enabledCompanyIds: ['c1', 'c2'],
    });
  });

  it('delegates canRoute/canMenu/hasAction to permission helpers', async () => {
    canRouteMock.mockClear();
    canMenuMock.mockClear();
    hasActionMock.mockClear();
    canRouteMock.mockReturnValueOnce(false);
    canMenuMock.mockReturnValueOnce(true);
    hasActionMock.mockReturnValueOnce(true);

    const permissionState = { v: 1 };
    mockAuthStore = {
      identity: {
        metadata: {
          activeCompanyId: 'c9',
          enabledCompanyIds: ['c9'],
        },
      },
      permissionState,
    };

    const { usePermission } = await import('./usePermission');
    const perm = usePermission();

    const routeOk = perm.canRoute('auth.route.user_list');
    const menuOk = perm.canMenu('auth.menu.user_list');
    const actionOk = perm.hasAction('auth.action.user_edit');

    expect(routeOk).toBe(false);
    expect(menuOk).toBe(true);
    expect(actionOk).toBe(true);

    expect(canRouteMock).toHaveBeenCalledTimes(1);
    expect(canRouteMock).toHaveBeenCalledWith('auth.route.user_list', permissionState, perm.ctx.value);
    expect(canMenuMock).toHaveBeenCalledTimes(1);
    expect(canMenuMock).toHaveBeenCalledWith('auth.menu.user_list', permissionState, perm.ctx.value);
    expect(hasActionMock).toHaveBeenCalledTimes(1);
    expect(hasActionMock).toHaveBeenCalledWith('auth.action.user_edit', permissionState, perm.ctx.value);
  });
});
