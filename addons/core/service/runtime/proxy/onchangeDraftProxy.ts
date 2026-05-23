// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

type PatchSink = (path: string, value: unknown) => void;

function isObject(value: unknown): value is object {
  return value !== null && typeof value === 'object';
}

const MUTATING_METHODS = new Set(['push', 'pop', 'shift', 'unshift', 'splice', 'sort', 'reverse', 'fill', 'copyWithin']);

export function createWriteProxy<T>(root: T, sink: PatchSink, basePath = ''): T {
  const targetToProxy = new WeakMap<object, object>();
  const proxyToTarget = new WeakMap<object, object>();

  const makePath = (parent: string, key: PropertyKey) => {
    const k = typeof key === 'string' ? key : String(key);
    return parent ? `${parent}.${k}` : k;
  };

  const wrap = (obj: unknown, parentPath: string): unknown => {
    if (!isObject(obj)) return obj;
    const cached = targetToProxy.get(obj);
    if (cached) return cached;

    const proxy = new Proxy(obj, {
      get(target, prop, receiver) {
        // Wrap array mutation methods and emit a whole-array patch after execution.
        if (Array.isArray(target) && typeof prop === 'string' && MUTATING_METHODS.has(prop)) {
          const originalMethod = Reflect.get(target, prop, receiver);
          if (typeof originalMethod === 'function') {
            return (...args: unknown[]) => {
              const ret = (originalMethod as (...args: unknown[]) => unknown).apply(target, args);
              // Report the full array as the new value instead of tracking per-item changes.
              sink(parentPath, target);
              return ret;
            };
          }
        }
        const v = Reflect.get(target, prop, receiver);
        return isObject(v) ? wrap(v, makePath(parentPath, prop)) : v;
      },

      set(target, prop, value, receiver) {
        // Ignore internal writes to array length.
        if (Array.isArray(target) && prop === 'length') {
          return Reflect.set(target, prop, value, receiver);
        }
        // Apply the actual write first.
        const ok = Reflect.set(target, prop, value, receiver);
        // Then report the patch, unwrapping proxy -> raw target when needed.
        const raw = isObject(value) ? (proxyToTarget.get(value) ?? value) : value;
        sink(makePath(parentPath, prop), raw);
        return ok;
      },

      deleteProperty(target, prop) {
        const ok = Reflect.deleteProperty(target, prop);
        // Use undefined to represent deletion.
        sink(makePath(parentPath, prop), undefined);
        return ok;
      },
    });

    targetToProxy.set(obj, proxy);
    proxyToTarget.set(proxy, obj);
    return proxy;
  };

  return wrap(root, basePath) as T;
}
