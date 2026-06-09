// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';
import { createMenuPlugin, MenuSymbol } from './plugin';

describe('createMenuPlugin', () => {
  it('installs menu manager onto app globals and injection without global router leakage', () => {
    const provide = vi.fn();
    const app = {
      config: {
        globalProperties: {
          $router: { push: vi.fn() },
        },
      },
      provide,
    } as any;

    delete (globalThis as any).__CHOYSUM_ROUTER__;

    const plugin = createMenuPlugin();
    plugin.install(app);

    expect(app.config.globalProperties.$menu).toBe(plugin.manager);
    expect(provide).toHaveBeenCalledWith(MenuSymbol, plugin.manager);
    expect((globalThis as any).__CHOYSUM_ROUTER__).toBeUndefined();
  });

  it('delegates menu operations to the underlying manager', () => {
    const plugin = createMenuPlugin();

    plugin.addMenu({ id: 'root', title: 'Root', path: '/root' });

    expect(plugin.hasMenu('root')).toBe(true);
    expect(plugin.getMenuByPath('/root')?.id).toBe('root');
    expect(plugin.getMenus().map(item => item.id)).toEqual(['root']);
  });
});
