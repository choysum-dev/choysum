// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/**
 * FE unit stub for auth store used when mounting Login.vue under QJS.
 * Activated via host_bundle path stub for modules/auth/web/stores/auth.
 */
import { defineStore } from 'pinia';
import { ref, computed } from 'vue';

export const useAuthStore = defineStore('auth-fe-stub', function () {
  var loading = ref(false);
  var isAuthenticated = ref(false);
  var currentUser = ref(null);
  var identity = ref({ metadata: {} });
  return {
    loading: loading,
    isAuthenticated: computed(function () {
      return !!isAuthenticated.value;
    }),
    currentUser: currentUser,
    identity: identity,
    ensureAuthReady: function () {
      return Promise.resolve();
    },
    login: function () {
      isAuthenticated.value = true;
      return Promise.resolve();
    },
    loadUser: function () {
      return Promise.resolve(currentUser.value);
    },
  };
});

export default useAuthStore;
