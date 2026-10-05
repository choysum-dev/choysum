// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createMenuPlugin, MenuSymbol, getInstalledMenu, resetInstalledMenu } from './plugin';

test('createMenuPlugin: installs menu manager onto app globals and injection without global router leakage', () => {
  const provideCalls: unknown[][] = [];
  const provide = (...args: unknown[]) => {
    provideCalls.push(args);
  };
  const app = {
    config: {
      globalProperties: {
        $router: { push: () => undefined },
      },
    },
    provide,
  } as any;

  delete (globalThis as any).__CHOYSUM_ROUTER__;

  const plugin = createMenuPlugin();
  plugin.install(app);

  expect(app.config.globalProperties.$menu).toBe(plugin.manager);
  expect(getInstalledMenu()).toBe(plugin.manager);
  expect(provideCalls).toEqual([[MenuSymbol, plugin.manager]]);
  expect((globalThis as any).__CHOYSUM_ROUTER__).toBeUndefined();
});

test('createMenuPlugin: restores prior getInstalledMenu on unmount; create does not clear', () => {
  resetInstalledMenu();
  const unmountCalls: number[] = [];
  const firstApp = {
    config: { globalProperties: {} },
    provide() {},
    unmount() {
      unmountCalls.push(1);
    },
  } as any;
  const secondApp = {
    config: { globalProperties: {} },
    provide() {},
    unmount() {
      unmountCalls.push(2);
    },
  } as any;

  const first = createMenuPlugin();
  first.install(firstApp);
  expect(getInstalledMenu()).toBe(first.manager);

  const second = createMenuPlugin();
  second.install(secondApp);
  expect(getInstalledMenu()).toBe(second.manager);
  secondApp.unmount();
  expect(getInstalledMenu()).toBe(first.manager);
  expect(unmountCalls).toEqual([2]);

  firstApp.unmount();
  expect(getInstalledMenu()).toBeNull();
  expect(unmountCalls).toEqual([2, 1]);

  const a = createMenuPlugin();
  const b = createMenuPlugin();
  const appA = {
    config: { globalProperties: {} },
    provide() {},
    unmount() {},
  } as any;
  const appB = {
    config: { globalProperties: {} },
    provide() {},
    unmount() {},
  } as any;
  a.install(appA);
  b.install(appB);
  appA.unmount();
  expect(getInstalledMenu()).toBe(b.manager);
  appB.unmount();
  expect(getInstalledMenu()).toBeNull();

  const leftover = createMenuPlugin();
  leftover.install({
    config: { globalProperties: {} },
    provide() {},
  } as any);
  expect(getInstalledMenu()).toBe(leftover.manager);
  createMenuPlugin();
  expect(getInstalledMenu()).toBe(leftover.manager);
  resetInstalledMenu();
  expect(getInstalledMenu()).toBeNull();
});

test('createMenuPlugin: delegates menu operations to the underlying manager', () => {
  const plugin = createMenuPlugin();

  plugin.addMenu({ id: 'root', title: 'Root', path: '/root' });

  expect(plugin.hasMenu('root')).toBe(true);
  expect(plugin.getMenuByPath('/root')?.id).toBe('root');
  expect(plugin.getMenus().map(item => item.id)).toEqual(['root']);
});
