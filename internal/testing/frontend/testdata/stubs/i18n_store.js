// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/** FE unit stub for i18nStore used by Login.vue preference apply. */
import { defineStore } from 'pinia';

export function bcp47FromCode(lang) {
  return String(lang || 'en_US').replace(/_/g, '-');
}

export function downloadTerminologyPo() {
  return Promise.resolve(new Uint8Array());
}

export function afterLangChange() {
  return Promise.resolve();
}

export function setLang() {
  return Promise.resolve(true);
}

export const useI18nStore = defineStore('i18n-fe-stub', function () {
  return {
    lang: 'en_US',
    activeLanguage: { code: 'en_US', bcp47: 'en-US' },
    setLang: function () {
      return Promise.resolve(true);
    },
  };
});

export default {
  useI18nStore: useI18nStore,
  bcp47FromCode: bcp47FromCode,
  setLang: setLang,
  downloadTerminologyPo: downloadTerminologyPo,
  afterLangChange: afterLangChange,
};