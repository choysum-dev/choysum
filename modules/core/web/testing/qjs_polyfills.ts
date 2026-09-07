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
