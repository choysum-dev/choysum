// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { useAuthStore } from '@/auth/web/stores/auth'
import { resolveRuntimeDefaultLandPath } from './resolveRuntimeDefaultLandPath'

export type RootRedirectTarget = string | { name: string }

/**
 * Resolve where `/` should go.
 * Guests stay in the guest chrome (login); authenticated users take the land path.
 */
export function resolveRootRedirect(): RootRedirectTarget {
  try {
    const auth = useAuthStore()
    if (!auth.isAuthenticated) {
      return { name: 'login' }
    }
  } catch {
    // Auth store unavailable (early boot / unit mounts without pinia) → guest entry.
    return { name: 'login' }
  }
  return resolveRuntimeDefaultLandPath()
}
