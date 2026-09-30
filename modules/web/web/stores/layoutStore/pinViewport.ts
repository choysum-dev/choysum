// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Pin a desktop or mobile viewport for layoutStore / VueUse breakpoints.
 * Handles compound `(min-width) and (max-width)` queries, including fractional
 * px thresholds and non-width clauses (e.g. prefers-color-scheme).
 */
export function pinViewportWidth(width: number): void {
  Object.defineProperty(window, 'innerWidth', {
    configurable: true,
    writable: true,
    value: width,
  });
  Object.defineProperty(window, 'matchMedia', {
    configurable: true,
    value: (query: string) => {
      const parts = String(query)
        .replace(/[()]/g, '')
        .split(/\s+and\s+/i)
        .map((p) => p.trim())
        .filter(Boolean);
      const matches =
        parts.length === 0
          ? false
          : parts.every((part) => {
              const min = /min-width:\s*(\d+(?:\.\d+)?)px/i.exec(part);
              const max = /max-width:\s*(\d+(?:\.\d+)?)px/i.exec(part);
              // Non-width clauses do not affect the width pin.
              if (!min && !max) return true;
              if (min && width < Number(min[1])) return false;
              if (max && width > Number(max[1])) return false;
              return true;
            });
      return {
        matches,
        media: query,
        onchange: null,
        addEventListener: () => {},
        removeEventListener: () => {},
        addListener: () => {},
        removeListener: () => {},
        dispatchEvent: () => false,
      };
    },
  });
}
