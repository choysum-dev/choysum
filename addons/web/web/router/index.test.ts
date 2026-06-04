// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => {
  const nprogress = {
    configure: vi.fn(),
    start: vi.fn(),
    done: vi.fn(),
  };

  const useTitle = vi.fn();

  let lastRouter: any = null;

  const createRouter = vi.fn(() => {
    lastRouter = {
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
    reset() {
      nprogress.configure.mockClear();
      nprogress.start.mockClear();
      nprogress.done.mockClear();
      useTitle.mockReset();
      createRouter.mockClear();
      createWebHistory.mockClear();
      lastRouter = null;
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
  });

  it('redirects to /error/500 when beforeEach throws on non-error routes', async () => {
    mocks.useTitle.mockImplementation(() => {
      throw new Error('boom');
    });

    createAppRouter('/');

    const guard = mocks.getBeforeEachGuard();
    const result = await guard(
      {
        path: '/home',
        meta: { pageTitle: 'Home' },
      } as any,
      {} as any
    );

    expect(result).toBe('/error/500');
    expect(mocks.nprogress.done).toHaveBeenCalledTimes(1);
    expect(consoleErrorSpy).toHaveBeenCalledTimes(1);
  });

  it('does not redirect repeatedly when already on /error/500', async () => {
    mocks.useTitle.mockImplementation(() => {
      throw new Error('boom');
    });

    createAppRouter('/');

    const guard = mocks.getBeforeEachGuard();
    const result = await guard(
      {
        path: '/error/500',
        meta: { pageTitle: 'Error' },
      } as any,
      {} as any
    );

    expect(result).toBe(true);
    expect(mocks.nprogress.done).toHaveBeenCalledTimes(1);
    expect(consoleErrorSpy).toHaveBeenCalledTimes(1);
  });
});
