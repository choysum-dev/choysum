// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createI18n } from 'vue-i18n';
import { createTextDescriptor } from '@/core/service/i18n';
import { projectTerminologyMessages } from '../i18n/terminology';

const mocks = vi.hoisted(() => {
  const nprogress = {
    configure: vi.fn(),
    start: vi.fn(),
    done: vi.fn(),
  };

  const useTitle = vi.fn();

  let lastRouter: any = null;
  let titleSource: any = null;

  const createRouter = vi.fn(() => {
    lastRouter = {
      currentRoute: { value: null },
      beforeEach: vi.fn(),
      afterEach: vi.fn(),
      onError: vi.fn(),
    };
    return lastRouter;
  });

  const createWebHistory = vi.fn((base: string) => ({ base }));

  return {
    nprogress,
    useTitle,
    createRouter,
    createWebHistory,
    getBeforeEachGuard() {
      return lastRouter?.beforeEach?.mock?.calls?.[0]?.[0];
    },
    getTitleSource() {
      return titleSource;
    },
    reset() {
      nprogress.configure.mockClear();
      nprogress.start.mockClear();
      nprogress.done.mockClear();
      useTitle.mockReset();
      useTitle.mockImplementation(source => {
        titleSource = source;
      });
      createRouter.mockClear();
      createWebHistory.mockClear();
      lastRouter = null;
      titleSource = null;
    },
  };
});

vi.mock('@vueuse/core', () => ({
  useTitle: mocks.useTitle,
}));

vi.mock('nprogress', () => ({
  default: mocks.nprogress,
}));

vi.mock('vue-router', () => ({
  createRouter: mocks.createRouter,
  createWebHistory: mocks.createWebHistory,
}));

vi.mock('./routes', () => ({
  default: [],
}));

import { createAppRouter } from './index';

describe('createAppRouter guards', () => {
  let consoleErrorSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    mocks.reset();
    consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
  });

  afterEach(() => {
    consoleErrorSpy.mockRestore();
  });

  it('returns true when beforeEach succeeds', async () => {
    createAppRouter('/');

    const guard = mocks.getBeforeEachGuard();
    expect(typeof guard).toBe('function');

    const result = await guard(
      {
        path: '/home',
        meta: { pageTitle: 'Home' },
      } as any,
      {} as any
    );

    expect(result).toBe(true);
    expect(mocks.nprogress.start).toHaveBeenCalledTimes(1);
    expect(mocks.useTitle).toHaveBeenCalledTimes(1);
    expect(mocks.getTitleSource().value).toBe('Home - Choysum');
  });

  it('updates a descriptor title without another navigation', async () => {
    const i18n = createI18n({
      legacy: false,
      locale: 'en',
      missingWarn: false,
      fallbackWarn: false,
      messages: { en: {}, 'zh-CN': projectTerminologyMessages({
        base: { 'base.route.users': { Users: '用户' } },
      }) },
    });
    createAppRouter('/', i18n.global);
    const guard = mocks.getBeforeEachGuard();
    await guard(
      {
        path: '/home',
        meta: {
          pageTitle: 'Users',
          pageTitleText: createTextDescriptor('base', 'Users', { scope: 'base.route.users' }),
        },
      } as any,
      {} as any
    );

    expect(mocks.getTitleSource().value).toBe('Users - Choysum');
    i18n.global.locale.value = 'zh-CN';
    expect(mocks.getTitleSource().value).toBe('用户 - Choysum');
  });
});
