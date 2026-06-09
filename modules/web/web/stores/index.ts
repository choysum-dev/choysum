// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { isClient } from '@vueuse/core';

// Re-export existing stores.
export * from './i18nStore';
export * from './layoutStore';

// /**
//  * Creates the Pinia instance.
//  * @param app Vue application instance.
//  * @returns Configured Pinia instance.
//  */
// export function createAppPinia(app?: App) {
//   const pinia = createPinia();

//   // Add the persistence plugin only in client environments.
//   if (isClient) {
//     pinia.use(piniaPluginPersistedstate);
//   }

//   // Set the active Pinia instance.
//   setActivePinia(pinia);

//   // Install Pinia when an app instance is provided.
//   if (app) {
//     app.use(pinia);
//   }

//   return pinia;
// }

/**
 * Clears all persisted state entries.
 * Useful during sign-out or cache cleanup.
 */
export function clearAllPersistedState(prefix: string = 'choysum'): void {
  if (!isClient) return;

  try {
    const keys = [];
    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i);
      if (key && key.startsWith(prefix)) {
        keys.push(key);
      }
    }

    keys.forEach(key => localStorage.removeItem(key));
    // cleared persisted states
  } catch (e) {
    console.error('Error clearing persisted states:', e);
  }
}

// // Create the default Pinia instance for non-SSR environments.
// export const pinia = isClient ? createPinia() : null;
// if (isClient && pinia) {
//   pinia.use(piniaPluginPersistedstate);
//   setActivePinia(pinia);
// }
