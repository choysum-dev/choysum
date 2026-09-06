// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Choysum enhanced application system.
 */

import { App, Component, createApp as vueCreateApp, Plugin as VuePlugin } from 'vue';
import { Router } from 'vue-router';
import { Pinia } from 'pinia';
import type { MenuPlugin } from '../menu/plugin';
import type { ObjectRecord } from '../../utils/types';

export type WebPluginSetup = (app: ChoysumWebApp) => void;

interface PluginRegistration {
  name: string;
  plugin: VuePlugin;
  options?: unknown;
}

export interface BuiltinPlugins {
  vueApp: App;
  router: Router;
  pinia: Pinia;
  menu: MenuPlugin;
}

export interface ChoysumWebApp extends App, BuiltinPlugins {
  usePlugin(name: string, plugin: VuePlugin, options?: unknown, deferred?: boolean): this;
  use<Options extends unknown[]>(plugin: VuePlugin<Options>, ...options: Options): this;
  setup(setup: WebPluginSetup): this;
}

export function createApp(rootComponent: Component, rootProps?: ObjectRecord): ChoysumWebApp {
  const vueApp = vueCreateApp(rootComponent, rootProps);
  const plugins = new Map<string, unknown>();
  const pendingPlugins: PluginRegistration[] = [];

  let isMounted = false;
  let mountedResult: unknown;
  let allowDirectUseDepth = 0;

  const originalUse = vueApp.use.bind(vueApp);

  plugins.set('vueApp', vueApp);

  const enhancedApp = vueApp as ChoysumWebApp;

  function registerPlugin<T>(name: string, plugin: T): ChoysumWebApp {
    if (plugins.has(name)) {
      return enhancedApp;
    }

    plugins.set(name, plugin);
    return enhancedApp;
  }

  function runWithDirectUseAllowed<T>(fn: () => T): T {
    allowDirectUseDepth += 1;
    try {
      return fn();
    } finally {
      allowDirectUseDepth -= 1;
    }
  }

  function installPlugin(plugin: VuePlugin, options?: unknown): void {
    runWithDirectUseAllowed(() => {
      originalUse(plugin, options);
    });
  }

  function flushPendingPlugins(): void {
    if (pendingPlugins.length === 0) {
      return;
    }

    for (const { name, plugin, options } of pendingPlugins) {
      try {
        installPlugin(plugin, options);
      } catch (error) {
        console.error(`Plugin ${name} registration failed:`, error);
      }
    }

    pendingPlugins.length = 0;
  }

  const mutableApp = enhancedApp as ChoysumWebApp & {
    setup: (setup: WebPluginSetup) => ChoysumWebApp;
    use: (plugin: VuePlugin, ...options: unknown[]) => ChoysumWebApp;
    usePlugin: (name: string, plugin: VuePlugin, options?: unknown, deferred?: boolean) => ChoysumWebApp;
    mount: ChoysumWebApp['mount'];
  };

  // Assign through `any` so typescript-go does not require fluent methods to
  // return the full mutableApp intersection (App method this-types collide).
  (mutableApp as any).setup = function (setup: WebPluginSetup): ChoysumWebApp {
    setup(this);
    return this;
  };

  (mutableApp as any).use = function (plugin: VuePlugin<unknown[]>, ...options: unknown[]): ChoysumWebApp {
    if (allowDirectUseDepth > 0) {
      return originalUse(plugin, ...options) as unknown as ChoysumWebApp;
    }

    throw new Error('ChoysumWebApp disables app.use(...); use app.usePlugin(name, plugin, options?) instead');
  };

  (mutableApp as any).usePlugin = function (name: string, plugin: VuePlugin, options?: unknown, deferred: boolean = true): ChoysumWebApp {
    if (!name) {
      throw new Error('Plugin name cannot be empty');
    }

    if (!plugin) {
      throw new Error(`Failed to register plugin "${name}": no valid plugin object was provided`);
    }

    if (plugins.has(name)) {
      return this;
    }

    registerPlugin(name, plugin);

    if (isMounted) {
      installPlugin(plugin, options);
    } else if (deferred) {
      pendingPlugins.push({ name, plugin, options });
    } else {
      try {
        installPlugin(plugin, options);
      } catch (error) {
        console.error(`Immediate registration failed for plugin ${name}:`, error);
      }
    }

    return this;
  };

  const originalMount = vueApp.mount;
  mutableApp.mount = function (...args: Parameters<typeof originalMount>) {
    if (isMounted) {
      return mountedResult as ReturnType<typeof originalMount>;
    }

    flushPendingPlugins();
    isMounted = true;
    mountedResult = originalMount.apply(vueApp, args);

    return mountedResult as ReturnType<typeof originalMount>;
  } as ChoysumWebApp['mount'];

  return new Proxy(enhancedApp, {
    get(target, prop: string | symbol) {
      if (prop in target) {
        return target[prop as keyof typeof target];
      }

      if (typeof prop === 'string' && plugins.has(prop)) {
        return plugins.get(prop);
      }

      return undefined;
    },
  }) as ChoysumWebApp;
}

export function getPlugins<T extends ObjectRecord>(app: ChoysumWebApp, _typeHint: T): T {
  return app as unknown as T;
}
