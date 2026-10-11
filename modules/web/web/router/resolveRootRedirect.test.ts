// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia'
import { createMenuPlugin } from '@/core/web/menu'
import { MODULE_BOARD_PATH } from './resolveDefaultLandPath'
import { resolveRootRedirect } from './resolveRootRedirect'

describe('resolveRootRedirect', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    createMenuPlugin().install({
      config: { globalProperties: {} },
      provide() {},
    } as any)
  })

  test('sends unauthenticated visitors to login', async () => {
    const { useAuthStore } = await import('@/auth/web/stores/auth')
    useAuthStore()
    expect(resolveRootRedirect()).toEqual({ name: 'login' })
  })

  test('sends authenticated users to the default land path', async () => {
    const { useAuthStore } = await import('@/auth/web/stores/auth')
    const auth = useAuthStore()
    auth.tokens = {
      accessToken: 'access',
      refreshToken: 'refresh',
      expiresAt: Date.now() + 3_600_000,
      refreshExpiresAt: Date.now() + 7_200_000,
    } as any
    expect(auth.isAuthenticated).toBe(true)
    expect(resolveRootRedirect()).toBe(MODULE_BOARD_PATH)
  })

  test('falls back to login when the auth store is unavailable', () => {
    setActivePinia(undefined as any)
    expect(resolveRootRedirect()).toEqual({ name: 'login' })
  })
})
