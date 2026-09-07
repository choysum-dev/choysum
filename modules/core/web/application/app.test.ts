// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h } from 'vue';
import { createApp, getPlugins } from './index';
import { ensureConsole } from '../testing/qjs_polyfills';

ensureConsole();

const RootComponent = defineComponent({
  name: 'MigrateCoreRoot',
  render() {
    return h('div', { class: 'migrate-core-root' }, 'choysum');
  },
});

function makeHost() {
  const el = document.createElement('div');
  el.id = `app-${Math.random().toString(36).slice(2)}`;
  document.body.appendChild(el);
  return el;
}

function removeHost(host: { parentNode?: { removeChild: (n: unknown) => void } | null }) {
  host.parentNode?.removeChild(host);
}

test('createApp: runs setup immediately and returns the same app instance', () => {
  const host = makeHost();
  const app = createApp(RootComponent);
  let setupCalls = 0;
  const setup = (arg: unknown) => {
    setupCalls += 1;
    expect(arg).toBe(app);
  };

  const result = app.setup(setup);
  expect(result).toBe(app);
  expect(setupCalls).toBe(1);
  app.mount(host);
  app.unmount();
  removeHost(host);
});

test('createApp: exposes deferred plugins before mount and installs them once on mount', () => {
  const host = makeHost();
  const app = createApp(RootComponent);
  let installCalls = 0;
  let installOptions: unknown;
  const plugin = {
    install(_a: unknown, options?: unknown) {
      installCalls += 1;
      installOptions = options;
    },
  };

  app.usePlugin('demo', plugin as any, { locale: 'zh-CN' });
  expect((app as any).demo).toBe(plugin);
  expect(getPlugins(app, { demo: {} as any }).demo).toBe(plugin);
  expect(installCalls).toBe(0);

  const mounted = app.mount(host);
  expect(mounted).toBeTruthy();
  expect(installCalls).toBe(1);
  expect(installOptions).toEqual({ locale: 'zh-CN' });
  expect(app.mount(host)).toBe(mounted);
  expect(installCalls).toBe(1);
  app.unmount();
  removeHost(host);
});

test('createApp: deduplicates plugins by name before mount and after mount', () => {
  const host = makeHost();
  const app = createApp(RootComponent);
  let first = 0;
  let second = 0;
  app.usePlugin('demo', {
    install() {
      first += 1;
    },
  } as any);
  app.usePlugin('demo', {
    install() {
      second += 1;
    },
  } as any);
  app.mount(host);
  expect(first).toBe(1);
  expect(second).toBe(0);
  app.usePlugin(
    'demo',
    {
      install() {
        second += 1;
      },
    } as any,
    undefined,
    false,
  );
  expect(first).toBe(1);
  expect(second).toBe(0);
  app.unmount();
  removeHost(host);
});

test('createApp: allows nested app.use during plugin installation', () => {
  const host = makeHost();
  const app = createApp(RootComponent);
  let nested = 0;
  let parent = 0;
  const nestedPlugin = {
    install() {
      nested += 1;
    },
  };
  app.usePlugin('parent', {
    install() {
      parent += 1;
      app.use(nestedPlugin as any, { source: 'parent' });
    },
  } as any);
  app.mount(host);
  expect(parent).toBe(1);
  expect(nested).toBe(1);
  app.unmount();
  removeHost(host);
});

test('createApp: rejects direct app.use to keep plugin registration explicit', () => {
  const app = createApp(RootComponent);
  expect(() => (app as any).use({ install() {} })).toThrow(/usePlugin/);
});

test('createApp: mounts without plugins and validates usePlugin inputs', () => {
  const host = makeHost();
  const app = createApp(RootComponent);
  const prevError = console.error;
  console.error = () => {};
  try {
    expect(app.mount(host)).toBeTruthy();
    expect(() => app.usePlugin('', { install() {} } as any)).toThrow(/cannot be empty/);
    expect(() => app.usePlugin('broken', undefined as any)).toThrow(/no valid plugin/);
  } finally {
    console.error = prevError;
    app.unmount();
    removeHost(host);
  }
});

test('createApp: installs plugins immediately when deferred is false', () => {
  const host = makeHost();
  const app = createApp(RootComponent);
  let installCalls = 0;
  app.usePlugin(
    'eager',
    {
      install() {
        installCalls += 1;
      },
    } as any,
    { mode: 'sync' },
    false,
  );
  expect(installCalls).toBe(1);
  expect((app as any).eager).toBeTruthy();
  app.mount(host);
  app.unmount();
  removeHost(host);
});

test('createApp: installs plugins registered after mount', () => {
  const host = makeHost();
  const app = createApp(RootComponent);
  app.mount(host);
  let installCalls = 0;
  app.usePlugin('late', {
    install() {
      installCalls += 1;
    },
  } as any);
  expect(installCalls).toBe(1);
  app.unmount();
  removeHost(host);
});

test('createApp: logs and continues when deferred plugin install fails on mount', () => {
  const host = makeHost();
  const app = createApp(RootComponent);
  const errors: unknown[] = [];
  const prevError = console.error;
  console.error = (...args: unknown[]) => {
    errors.push(args[0]);
  };
  try {
    app.usePlugin('bad', {
      install() {
        throw new Error('install boom');
      },
    } as any);
    expect(app.mount(host)).toBeTruthy();
    expect(errors.length).toBeGreaterThan(0);
    expect(String(errors[0])).toContain('Plugin bad registration failed');
  } finally {
    console.error = prevError;
    app.unmount();
    removeHost(host);
  }
});

test('createApp: logs and continues when immediate plugin install fails', () => {
  const app = createApp(RootComponent);
  const errors: unknown[] = [];
  const prevError = console.error;
  console.error = (...args: unknown[]) => {
    errors.push(args[0]);
  };
  try {
    app.usePlugin(
      'eager-bad',
      {
        install() {
          throw new Error('eager boom');
        },
      } as any,
      undefined,
      false,
    );
    expect(errors.length).toBeGreaterThan(0);
    expect(String(errors[0])).toContain('Immediate registration failed for plugin eager-bad');
  } finally {
    console.error = prevError;
  }
});

test('createApp: forwards Vue App props and returns undefined for unknown keys', () => {
  const app = createApp(RootComponent) as any;
  expect(app.config).toBeTruthy();
  expect(typeof app.provide).toBe('function');
  expect(app.missingPlugin).toBeUndefined();
  expect(app[Symbol('choysum')]).toBeUndefined();
});
