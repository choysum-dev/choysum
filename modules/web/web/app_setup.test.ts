// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, it, vi } from 'vitest';

const exposeBrowserI18nOnWindow = vi.hoisted(() => vi.fn());

vi.mock('./i18n', async importOriginal => {
  const actual = await importOriginal<typeof import('./i18n')>();
  return {
    ...actual,
    exposeBrowserI18nOnWindow,
  };
});
vi.mock('./directives', () => ({ registerGlobalDirectives: vi.fn() }));
vi.mock('pinia', () => ({ createPinia: () => ({ use: () => ({}) }) }));
vi.mock('pinia-plugin-persistedstate', () => ({ default: vi.fn() }));
vi.mock('./stores/i18nStore', () => ({
  useI18nStore: () => ({
    currentLocale: { code: 'en', elementLocale: {} },
    terminologyLang: 'en_US',
    lastTerminologyLoad: null,
    getDateTimeFormats: () => ({}),
    getNumberFormats: () => ({}),
    loadVueI18nMessages: vi.fn(async () => null),
  }),
}));
vi.mock('@/auth/web/stores/auth', () => ({ useAuthStore: () => ({}) }));
vi.mock('@/core/rpc/context', () => ({ setGlobalRequestContextProvider: vi.fn() }));
vi.mock('./utils/request_timezone', () => ({
  detectBrowserTimezone: () => 'UTC',
  resolveRequestTimezone: () => 'UTC',
}));
vi.mock('./utils/datetime', () => ({ setUserTimeZoneResolver: vi.fn() }));
vi.mock('vue-i18n', () => ({
  createI18n: vi.fn(() => ({ global: { locale: { value: 'en' } } })),
}));
vi.mock('./i18n/source', () => ({ default: {} }));
vi.mock('./stores/i18nStore/merge', () => ({
  createTerminologyCatalogMerger: vi.fn(() => vi.fn()),
}));
vi.mock('./router', () => ({ createAppRouter: vi.fn(() => ({})) }));
vi.mock('./menu', () => ({ createAppMenu: vi.fn(() => ({})) }));
vi.mock('element-plus', () => ({ default: {} }));
vi.mock('vue', async importOriginal => {
  const actual = await importOriginal<typeof import('vue')>();
  return { ...actual, watch: vi.fn() };
});

import { setupApp } from './app_setup';

describe('setupApp', () => {
  beforeEach(() => {
    exposeBrowserI18nOnWindow.mockClear();
  });

  it('exposes browser i18n globals during bootstrap', () => {
    const app = {
      config: { globalProperties: {} },
      usePlugin: vi.fn(),
    };

    setupApp(app as any);

    expect(exposeBrowserI18nOnWindow).toHaveBeenCalledTimes(1);
  });
});
