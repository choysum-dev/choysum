// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, it, vi } from 'vitest';

const vueMocks = vi.hoisted(() => {
  const installed: Array<{ app: any; plugin: any; options: any }> = [];
  let mountSeed = 0;

  const createVueApp = vi.fn(() => {
    const mountResult = { uid: ++mountSeed };
    const app: any = {
      config: { globalProperties: {} },
      provide: vi.fn(),
      use: vi.fn(function (plugin: any, options?: any) {
        installed.push({ app, plugin, options });
        if (plugin && typeof plugin.install === 'function') {
          plugin.install(app, options);
        }
        return app;
      }),
      mount: vi.fn(() => mountResult),
    };
    return app;
  });

  return {
    installed,
    createVueApp,
    reset() {
      installed.length = 0;
      mountSeed = 0;
      createVueApp.mockClear();
    },
  };
});

vi.mock('vue', () => ({
  createApp: vueMocks.createVueApp,
  defineComponent: (options: any) => options,
}));

import { defineComponent } from 'vue';
import { createApp, getPlugins } from './index';

const RootComponent = defineComponent({
  template: '<div>choysum</div>',
});

describe('createApp', () => {
  beforeEach(() => {
    vueMocks.reset();
  });

  it('runs setup immediately and returns the same app instance', () => {
    const app = createApp(RootComponent);
    const setup = vi.fn();

    const result = app.setup(setup);

    expect(result).toBe(app);
    expect(setup).toHaveBeenCalledTimes(1);
    expect(setup).toHaveBeenCalledWith(app);
  });

  it('exposes deferred plugins before mount and installs them once on mount', () => {
    const app = createApp(RootComponent);
    const install = vi.fn();
    const plugin = { install };

    app.usePlugin('demo', plugin as any, { locale: 'zh-CN' });

    expect((app as any).demo).toBe(plugin);
    expect(getPlugins(app, { demo: {} as any }).demo).toBe(plugin);
    expect(vueMocks.installed).toHaveLength(0);

    const mounted = app.mount('#app');

    expect(mounted).toBeTruthy();
    expect(install).toHaveBeenCalledTimes(1);
    expect(install.mock.calls[0]?.[1]).toEqual({ locale: 'zh-CN' });
    expect(vueMocks.installed).toHaveLength(1);
    expect(app.mount('#app')).toBe(mounted);
    expect(install).toHaveBeenCalledTimes(1);
  });

  it('deduplicates plugins by name before mount and after mount', () => {
    const app = createApp(RootComponent);
    const firstInstall = vi.fn();
    const secondInstall = vi.fn();

    app.usePlugin('demo', { install: firstInstall } as any);
    app.usePlugin('demo', { install: secondInstall } as any);

    app.mount('#app');

    expect(firstInstall).toHaveBeenCalledTimes(1);
    expect(secondInstall).not.toHaveBeenCalled();
    expect(vueMocks.installed).toHaveLength(1);

    app.usePlugin('demo', { install: vi.fn() } as any, undefined, false);
    expect(firstInstall).toHaveBeenCalledTimes(1);
    expect(vueMocks.installed).toHaveLength(1);
  });

  it('allows nested app.use during plugin installation', () => {
    const app = createApp(RootComponent);
    const nestedInstall = vi.fn();
    const nestedPlugin = { install: nestedInstall };
    const parentInstall = vi.fn((targetApp: any) => {
      targetApp.use(nestedPlugin, { source: 'parent' });
    });

    app.usePlugin('parent', { install: parentInstall } as any);
    app.mount('#app');

    expect(parentInstall).toHaveBeenCalledTimes(1);
    expect(nestedInstall).toHaveBeenCalledTimes(1);
    expect(vueMocks.installed).toHaveLength(2);
  });

  it('rejects direct app.use to keep plugin registration explicit', () => {
    const app = createApp(RootComponent);

    expect(() => (app as any).use({ install() {} })).toThrow(/usePlugin/);
  });
});
