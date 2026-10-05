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

test('createMenuPlugin: clears getInstalledMenu on unmount; create does not', () => {
  const unmountCalls: number[] = [];
  const app = {
    config: { globalProperties: {} },
    provide() {},
    unmount() {
      unmountCalls.push(1);
    },
  } as any;

  const plugin = createMenuPlugin();
  plugin.install(app);
  expect(getInstalledMenu()).toBe(plugin.manager);
  app.unmount();
  expect(getInstalledMenu()).toBeNull();
  expect(unmountCalls).toEqual([1]);

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
