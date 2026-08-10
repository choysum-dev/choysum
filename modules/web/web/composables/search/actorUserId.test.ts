// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const { authState } = vi.hoisted(() => ({
  authState: {
    throwOnAccess: false,
    currentUser: null as any,
    identity: null as any,
  },
}));

vi.mock('@/auth/web/stores/auth', () => ({
  useAuthStore: () => {
    if (authState.throwOnAccess) throw new Error('no pinia');
    return {
      currentUser: authState.currentUser,
      identity: authState.identity,
    };
  },
}));

import { actorUserId } from './actorUserId';

describe('actorUserId', () => {
  const prevChoysum = (globalThis as any).$choysum;

  beforeEach(() => {
    authState.throwOnAccess = false;
    authState.currentUser = null;
    authState.identity = null;
    delete (globalThis as any).$choysum;
  });

  afterEach(() => {
    if (prevChoysum === undefined) delete (globalThis as any).$choysum;
    else (globalThis as any).$choysum = prevChoysum;
  });

  it('prefers auth store currentUser.Id', () => {
    authState.currentUser = { Id: 'user-from-store' };
    authState.identity = { userId: 'ignored' };
    expect(actorUserId()).toBe('user-from-store');
  });

  it('falls back to auth identity.userId', () => {
    authState.currentUser = {};
    authState.identity = { userId: '  identity-user  ' };
    expect(actorUserId()).toBe('identity-user');
  });

  it('falls back to $choysum.request.context when auth store is empty', () => {
    (globalThis as any).$choysum = {
      request: { context: { identity: { userId: 'ctx-user' } } },
    };
    expect(actorUserId()).toBe('ctx-user');
  });

  it('returns empty when auth store throws and context is missing', () => {
    authState.throwOnAccess = true;
    expect(actorUserId()).toBe('');
  });

  it('returns empty when context identity access throws', () => {
    authState.throwOnAccess = true;
    Object.defineProperty(globalThis, '$choysum', {
      configurable: true,
      get() {
        throw new Error('boom');
      },
    });
    expect(actorUserId()).toBe('');
    delete (globalThis as any).$choysum;
  });
});
