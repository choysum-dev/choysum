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
  const originalMount = vueApp.mount.bind(vueApp);

  plugins.set('vueApp', vueApp);

  // Filled after Proxy construction so fluent methods always return the public app.
  let app!: ChoysumWebApp;

  function registerPlugin<T>(name: string, plugin: T): void {
    plugins.set(name, plugin);
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

  // Choysum fluent API lives on a plain object and is exposed via Proxy — never
  // assigned onto Vue App's this-typed methods (avoids typescript-go collisions).
  const choysumMethods = {
    setup(setup: WebPluginSetup): ChoysumWebApp {
      setup(app);
      return app;
    },

    use(plugin: VuePlugin<unknown[]>, ...options: unknown[]): ChoysumWebApp {
      if (allowDirectUseDepth > 0) {
        originalUse(plugin, ...options);
        return app;
      }

      throw new Error('ChoysumWebApp disables app.use(...); use app.usePlugin(name, plugin, options?) instead');
    },

    usePlugin(name: string, plugin: VuePlugin, options?: unknown, deferred: boolean = true): ChoysumWebApp {
      if (!name) {
        throw new Error('Plugin name cannot be empty');
      }

      if (!plugin) {
        throw new Error(`Failed to register plugin "${name}": no valid plugin object was provided`);
      }

      if (plugins.has(name)) {
        return app;
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

      return app;
    },

    mount(...args: Parameters<App['mount']>): ReturnType<App['mount']> {
      if (isMounted) {
        return mountedResult as ReturnType<App['mount']>;
      }

      flushPendingPlugins();
      isMounted = true;
      mountedResult = originalMount(...args);
      return mountedResult as ReturnType<App['mount']>;
    },
  };

  app = new Proxy(vueApp, {
    get(target, prop, receiver) {
      if (prop === 'setup' || prop === 'use' || prop === 'usePlugin' || prop === 'mount') {
        return choysumMethods[prop];
      }

      if (prop in target) {
        return Reflect.get(target, prop, receiver);
      }

      if (typeof prop === 'string' && plugins.has(prop)) {
        return plugins.get(prop);
      }

      return undefined;
    },
  }) as ChoysumWebApp;

  return app;
}

export function getPlugins<T extends ObjectRecord>(app: ChoysumWebApp, _typeHint: T): T {
  return app as unknown as T;
}
