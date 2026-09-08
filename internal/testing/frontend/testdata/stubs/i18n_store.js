// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/** FE unit stub for i18nStore used by Login.vue preference apply. */
import { defineStore } from 'pinia';

export function langToUiKey(lang) {
  return String(lang || 'en_US');
}

export function downloadTerminologyPo() {
  return Promise.resolve(new Uint8Array());
}

export const useI18nStore = defineStore('i18n-fe-stub', function () {
  return {
    setUiKey: function () {
      return Promise.resolve();
    },
    setDisplayOverrides: function () {},
  };
});

export default { useI18nStore: useI18nStore, langToUiKey: langToUiKey, downloadTerminologyPo: downloadTerminologyPo };
