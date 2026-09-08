// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { actorUserId } from './actorUserId';

describe('actorUserId', () => {
  const prevChoysum = (globalThis as any).$choysum;

  afterEach(() => {
    if (prevChoysum === undefined) delete (globalThis as any).$choysum;
    else (globalThis as any).$choysum = prevChoysum;
  });

  test('prefers auth store currentUser.Id', () => {
    expect(
      actorUserId({
        useAuthStore: (() => ({
          currentUser: { Id: 'user-from-store' },
          identity: { userId: 'ignored' },
        })) as any,
      })
    ).toBe('user-from-store');
  });

  test('falls back to auth identity.userId', () => {
    expect(
      actorUserId({
        useAuthStore: (() => ({
          currentUser: {},
          identity: { userId: '  identity-user  ' },
        })) as any,
      })
    ).toBe('identity-user');
  });

  test('falls back to $choysum.request.context when auth store is empty', () => {
    (globalThis as any).$choysum = {
      request: { context: { identity: { userId: 'ctx-user' } } },
    };
    expect(
      actorUserId({
        useAuthStore: (() => ({
          currentUser: null,
          identity: null,
        })) as any,
      })
    ).toBe('ctx-user');
  });

  test('returns empty when auth store throws and context is missing', () => {
    delete (globalThis as any).$choysum;
    expect(
      actorUserId({
        useAuthStore: (() => {
          throw new Error('no pinia');
        }) as any,
      })
    ).toBe('');
  });

  test('returns empty when context identity access throws', () => {
    Object.defineProperty(globalThis, '$choysum', {
      configurable: true,
      get() {
        throw new Error('boom');
      },
    });
    try {
      expect(
        actorUserId({
          useAuthStore: (() => {
            throw new Error('no pinia');
          }) as any,
        })
      ).toBe('');
    } finally {
      delete (globalThis as any).$choysum;
    }
  });
});
