// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import app from '@/web/web';
import type { ChoysumWebApp } from '@/core/web/application';
import { useAuthStore } from './stores/auth';
import { setCSRFProvider } from '@/core/web/rpc';
import { setupRouter } from './route';
import { setupAppMenu } from './menu';
import { watch } from 'vue';
import { applyPermissionToMenus } from './menu/applyPermissionToMenus';
import { hasAction as hasActionRaw } from '@/auth/web/permission';
import { setGlobalActionChecker } from '@/web/web/directives/action';
import { setupTokenProvider } from './setup_token_provider';

export { setupTokenProvider } from './setup_token_provider';

/**
 * Set up the auth module within the web application.
 *
 * @param app - Choysum web application instance.
 */
export function setupApp(app: ChoysumWebApp): void {
  const authStore = useAuthStore();

  setGlobalActionChecker((resourceId: string | undefined) => {
    const meta = (authStore.identity as any)?.metadata as any;
    return hasActionRaw(resourceId, authStore.permissionState, {
      activeCompanyId: meta?.activeCompanyId,
      enabledCompanyIds: meta?.enabledCompanyIds,
    });
  });

  // Install the access token provider first.
  setupTokenProvider(authStore);

  // Enable the CSRF provider only when the server expects it.
  if (import.meta.env.CHOYSUM_CSRF_ENABLED !== false) {
    setupCSRFProvider(authStore);
  }

  // Register auth routes and guards.
  setupRouter(app);

  // Register auth menus.
  setupAppMenu(app);

  /**
   * Recompute menu visibility and disabled state from permission metadata.
   */
  const refreshMenuPermission = () => {
    const meta = (authStore.identity as any)?.metadata as any;
    const ctx = {
      activeCompanyId: meta?.activeCompanyId,
      enabledCompanyIds: meta?.enabledCompanyIds,
    };
    applyPermissionToMenus(app.menu.getMenus(), authStore.permissionState, ctx);
  };

  // Warm up permission state early after login to avoid long fail-closed menus.
  watch(
    () => authStore.isAuthenticated,
    isAuthed => {
      if (isAuthed) {
        authStore.loadPermissionState(false).catch(() => {
          // Ignore failures here and keep the menu fail-closed.
        });
      }
      refreshMenuPermission();
    },
    { immediate: true }
  );

  // Refresh menu projection whenever permission state or identity metadata changes.
  watch(
    () => [authStore.permissionState, authStore.identity] as const,
    () => refreshMenuPermission(),
    { immediate: true }
  );

  // Clear auth-owned global state during unmount.
  app.unmount = () => {
    authStore.clearAuth();
    setGlobalActionChecker(undefined);

    if (import.meta.env.DEV) {
      console.debug('[Auth] Cleared auth state during app unmount');
    }
  };
}

/**
 * Register the CSRF token provider backed by the auth store.
 *
 * @param authStore - Auth state store.
 */
function setupCSRFProvider(authStore: ReturnType<typeof useAuthStore>): void {
  setCSRFProvider({
    getCSRFToken: () => authStore.getCsrfToken(),
  });

  if (import.meta.env.DEV) {
    console.debug('[Auth] CSRF provider configured');
  }
}

/**
 * Choysum auth web module.
 *
 * Builds auth-specific behavior on top of the shared web application.
 *
 * @module auth
 */
const authApp: ChoysumWebApp = app.setup(setupApp);
export default authApp;
