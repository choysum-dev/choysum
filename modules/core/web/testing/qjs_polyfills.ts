// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** Minimal AbortController for QuickJS FE unit (not a full browser implementation). */
export function ensureAbortController(): void {
  const g = globalThis as any;
  if (typeof g.AbortController === 'function') {
    return;
  }
  g.AbortController = class AbortController {
    signal: {
      aborted: boolean;
      addEventListener: (type: string, fn: () => void, opts?: { once?: boolean }) => void;
      removeEventListener: (type: string, fn: () => void) => void;
    };
    private listeners: Array<() => void> = [];
    constructor() {
      const self = this;
      this.signal = {
        aborted: false,
        addEventListener(type: string, fn: () => void, opts?: { once?: boolean }) {
          if (type !== 'abort') return;
          const wrap = () => {
            fn();
            if (opts?.once) {
              self.signal.removeEventListener(type, wrap);
            }
          };
          (wrap as any).__src = fn;
          self.listeners.push(wrap);
        },
        removeEventListener(_type: string, fn: () => void) {
          self.listeners = self.listeners.filter(l => l !== fn && (l as any).__src !== fn);
        },
      };
    }
    abort() {
      if (this.signal.aborted) return;
      this.signal.aborted = true;
      for (const fn of this.listeners.slice()) {
        try {
          fn();
        } catch {
          // ignore
        }
      }
    }
  };
}

/** Minimal Headers for QuickJS FE unit (constructor init + get/has/set). */
export function ensureHeaders(): void {
  const g = globalThis as any;
  if (typeof g.Headers === 'function') {
    return;
  }
  g.Headers = class Headers {
    private map = new Map<string, string>();
    constructor(init?: unknown) {
      if (!init) return;
      if (Array.isArray(init)) {
        for (const pair of init) {
          if (pair && pair.length >= 2) this.set(String(pair[0]), String(pair[1]));
        }
        return;
      }
      if (typeof (init as any).forEach === 'function') {
        (init as Headers).forEach((value: string, key: string) => this.set(key, value));
        return;
      }
      for (const [key, value] of Object.entries(init as Record<string, string>)) {
        this.set(key, value);
      }
    }
    get(name: string): string | null {
      return this.map.get(String(name).toLowerCase()) ?? null;
    }
    has(name: string): boolean {
      return this.map.has(String(name).toLowerCase());
    }
    set(name: string, value: string): void {
      this.map.set(String(name).toLowerCase(), String(value));
    }
    forEach(fn: (value: string, key: string) => void): void {
      for (const [key, value] of this.map) fn(value, key);
    }
  };
}

/** Minimal console stub when QuickJS host did not install console. */
export function ensureConsole(): void {
  const g = globalThis as any;
  if (g.console && typeof g.console.error === 'function') {
    return;
  }
  const noop = () => {};
  g.console = {
    log: noop,
    info: noop,
    warn: noop,
    error: noop,
    debug: noop,
    trace: noop,
  };
}
